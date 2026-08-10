package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
)

var safeID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
var safeFilename = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var supportedSelfAliases = map[string]struct{}{
	"git-wt": {},
}

func (catalog *Catalog) validate(schemaVersion int) error {
	if schemaVersion != 1 {
		return fmt.Errorf("unsupported catalog schema version %d", schemaVersion)
	}
	if len(catalog.Tools) == 0 {
		return fmt.Errorf("catalog contains no tools")
	}
	seen := make(map[string]struct{}, len(catalog.Tools))
	for index, tool := range catalog.Tools {
		if !safeID.MatchString(tool.ID) {
			return fmt.Errorf("tool %d has invalid id %q", index, tool.ID)
		}
		if _, exists := seen[tool.ID]; exists {
			return fmt.Errorf("duplicate tool id %q", tool.ID)
		}
		seen[tool.ID] = struct{}{}
		if tool.Name == "" || tool.Description == "" || len(tool.Artifacts) == 0 {
			return fmt.Errorf("tool %q requires name, description, and artifacts", tool.ID)
		}
		for artifactIndex, artifact := range tool.Artifacts {
			if err := catalog.validateArtifact(artifact); err != nil {
				return fmt.Errorf("tool %q artifact %d: %w", tool.ID, artifactIndex, err)
			}
		}
	}
	return nil
}

func (catalog *Catalog) validateArtifact(artifact Artifact) error {
	if !safeArtifactName(artifact.Name) {
		return fmt.Errorf("invalid destination name %q", artifact.Name)
	}
	if artifact.Mode == 0 || artifact.Mode > 0o777 {
		return fmt.Errorf("invalid file mode %#o", artifact.Mode)
	}
	switch artifact.Destination {
	case DestinationBin:
		if artifact.Mode&0o111 == 0 {
			return fmt.Errorf("bin artifact %q is not executable", artifact.Name)
		}
	case DestinationMan1:
	default:
		return fmt.Errorf("invalid destination %q", artifact.Destination)
	}
	if err := validatePlatforms(artifact.Platforms); err != nil {
		return err
	}
	switch artifact.Source {
	case SourceEmbedded:
		if artifact.Path == "" || path.Clean(artifact.Path) != artifact.Path || path.IsAbs(artifact.Path) {
			return fmt.Errorf("invalid embedded path %q", artifact.Path)
		}
		info, err := fs.Stat(catalog.assets, artifact.Path)
		if err != nil {
			return fmt.Errorf("stat embedded path %q: %w", artifact.Path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("embedded path %q is not a regular file", artifact.Path)
		}
	case SourceSelf:
		if artifact.Path != "" || artifact.Destination != DestinationBin {
			return fmt.Errorf("self artifact must be a bin alias without a path")
		}
		if _, supported := supportedSelfAliases[artifact.Name]; !supported {
			return fmt.Errorf("self alias %q has no compiled dispatch", artifact.Name)
		}
	default:
		return fmt.Errorf("invalid source %q", artifact.Source)
	}
	return nil
}

func safeArtifactName(value string) bool {
	return value != "." && value != ".." && path.Base(value) == value && safeFilename.MatchString(value)
}

func validatePlatforms(platforms []string) error {
	allowed := map[string]bool{"darwin": true, "linux": true, "windows": true}
	seen := make(map[string]bool, len(platforms))
	for _, platform := range platforms {
		if !allowed[platform] || seen[platform] {
			return fmt.Errorf("invalid or duplicate platform %q", platform)
		}
		seen[platform] = true
	}
	return nil
}
