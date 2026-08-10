package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jamesonstone/kura/internal/catalog"
)

func (service *Service) Install(ctx context.Context, toolIDs []string, options Options) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	if service.Catalog == nil {
		return Report{}, fmt.Errorf("tool catalog is unavailable")
	}
	if service.GOOS == "" {
		service.GOOS = runtime.GOOS
	}
	paths, err := service.resolvePaths(options)
	if err != nil {
		return Report{}, err
	}
	report := Report{BinDir: paths.bin, ManDir: paths.man, StateDir: paths.state}
	statePath := filepath.Join(paths.state, "installations.json")
	manifest, stateExpectation, previousState, err := service.loadManifest(statePath)
	if err != nil {
		return report, err
	}
	plans, err := service.plan(toolIDs, options.Force, paths, manifest)
	if err != nil {
		return report, err
	}
	for _, plan := range plans {
		report.Results = append(report.Results, plan.result)
		manifest.Files[plan.result.Path] = ownedRecord{
			ToolID: plan.result.ToolID,
			Digest: plan.digest,
			Mode:   uint32(plan.mode.Perm()),
		}
	}
	stateContent, err := encodeManifest(manifest)
	if err != nil {
		return report, err
	}
	writes := make([]plannedArtifact, 0, len(plans)+1)
	for _, plan := range plans {
		if plan.write {
			writes = append(writes, plan)
		}
	}
	if !bytes.Equal(previousState, stateContent) {
		writes = append(writes, plannedArtifact{
			result:   Result{ToolID: "kura", Path: statePath, Status: StatusUpdated},
			mode:     0o600,
			content:  stateContent,
			digest:   digest(stateContent),
			expected: stateExpectation,
			write:    true,
		})
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := service.apply(writes); err != nil {
		return report, err
	}
	if !PathContains(paths.bin, service.environment.getenv("PATH")) {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("%s is not on PATH; add it before using installed commands", paths.bin))
	}
	return report, nil
}

func (service *Service) plan(
	toolIDs []string,
	force bool,
	paths resolvedPaths,
	manifest ownershipManifest,
) ([]plannedArtifact, error) {
	if len(toolIDs) == 0 {
		return nil, fmt.Errorf("select at least one tool")
	}
	seenTools := make(map[string]bool, len(toolIDs))
	seenPaths := make(map[string]string)
	var plans []plannedArtifact
	for _, id := range toolIDs {
		if seenTools[id] {
			return nil, fmt.Errorf("tool %q was selected more than once", id)
		}
		seenTools[id] = true
		tool, ok := service.Catalog.Tool(id)
		if !ok {
			return nil, fmt.Errorf("unknown tool %q", id)
		}
		toolPlans, err := service.planTool(tool, force, paths, manifest)
		if err != nil {
			return nil, err
		}
		for _, plan := range toolPlans {
			key := service.destinationKey(plan.result.Path)
			if prior, exists := seenPaths[key]; exists {
				return nil, fmt.Errorf("tools %q and %q target the same path %q", prior, id, plan.result.Path)
			}
			seenPaths[key] = id
			plans = append(plans, plan)
		}
	}
	sort.SliceStable(plans, func(left, right int) bool {
		return plans[left].result.Path < plans[right].result.Path
	})
	return plans, nil
}

func (service *Service) destinationKey(path string) string {
	path = filepath.Clean(path)
	if service.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func artifactMode(value uint32) fs.FileMode {
	return fs.FileMode(value) & fs.ModePerm
}

func executableName(artifact catalog.Artifact, goos string) string {
	if goos == "windows" && artifact.Source == catalog.SourceSelf && filepath.Ext(artifact.Name) == "" {
		return artifact.Name + ".exe"
	}
	return artifact.Name
}
