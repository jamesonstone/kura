package install

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/jamesonstone/kura/internal/catalog"
)

func (service *Service) planTool(
	tool catalog.Tool,
	force bool,
	paths resolvedPaths,
	manifest ownershipManifest,
) ([]plannedArtifact, error) {
	var plans []plannedArtifact
	for _, artifact := range tool.Artifacts {
		if !artifact.Supports(service.GOOS) {
			continue
		}
		content, err := service.artifactContent(artifact)
		if err != nil {
			return nil, fmt.Errorf("prepare tool %q: %w", tool.ID, err)
		}
		directory := paths.bin
		if artifact.Destination == catalog.DestinationMan1 {
			directory = paths.man
		}
		path := filepath.Join(directory, executableName(artifact, service.GOOS))
		plan, err := service.planArtifact(tool.ID, path, artifactMode(artifact.Mode), content, force, manifest)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("tool %q has no artifacts for %s", tool.ID, service.GOOS)
	}
	return plans, nil
}

func (service *Service) artifactContent(artifact catalog.Artifact) ([]byte, error) {
	switch artifact.Source {
	case catalog.SourceEmbedded:
		return service.Catalog.Content(artifact)
	case catalog.SourceSelf:
		content, err := service.files.readFile(service.ExecutablePath)
		if err != nil {
			return nil, fmt.Errorf("read Kura executable %q: %w", service.ExecutablePath, err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unsupported artifact source %q", artifact.Source)
	}
}

func (service *Service) planArtifact(
	toolID, path string,
	mode fs.FileMode,
	content []byte,
	force bool,
	manifest ownershipManifest,
) (plannedArtifact, error) {
	desiredDigest := digest(content)
	expectation, err := service.inspectFile(path)
	if err != nil {
		return plannedArtifact{}, err
	}
	plan := plannedArtifact{
		result:   Result{ToolID: toolID, Path: path, Status: StatusInstalled},
		mode:     mode,
		content:  content,
		digest:   desiredDigest,
		expected: expectation,
		write:    true,
	}
	if !expectation.exists {
		return plan, nil
	}
	if expectation.kind == 0 && expectation.digest == desiredDigest && expectation.perm == mode.Perm() {
		plan.result.Status = StatusUnchanged
		plan.write = false
		return plan, nil
	}
	if expectation.kind != 0 && expectation.kind != fs.ModeSymlink {
		return plannedArtifact{}, fmt.Errorf("refuse to replace non-file destination %q", path)
	}
	owner, owned := manifest.Files[path]
	if owned && expectation.kind == 0 && owner.ToolID == toolID &&
		owner.Digest == expectation.digest && owner.Mode == uint32(expectation.perm) {
		plan.result.Status = StatusUpdated
		return plan, nil
	}
	if force {
		plan.result.Status = StatusReplaced
		return plan, nil
	}
	return plannedArtifact{}, fmt.Errorf(
		"refuse to overwrite unowned or modified destination %q; use --force to replace it", path)
}
