package catalog

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
)

//go:embed assets
var defaultAssets embed.FS

type Source string

const (
	SourceEmbedded Source = "embedded"
	SourceSelf     Source = "self"
)

type Destination string

const (
	DestinationBin  Destination = "bin"
	DestinationMan1 Destination = "man1"
)

type Artifact struct {
	Source      Source      `json:"source"`
	Path        string      `json:"path,omitempty"`
	Destination Destination `json:"destination"`
	Name        string      `json:"name"`
	Mode        uint32      `json:"mode"`
	Platforms   []string    `json:"platforms,omitempty"`
}

func (artifact Artifact) Supports(goos string) bool {
	if len(artifact.Platforms) == 0 {
		return true
	}
	for _, platform := range artifact.Platforms {
		if platform == goos {
			return true
		}
	}
	return false
}

type Tool struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Artifacts   []Artifact `json:"artifacts"`
}

type Catalog struct {
	Tools  []Tool
	assets fs.FS
}

type document struct {
	SchemaVersion int    `json:"schema_version"`
	Tools         []Tool `json:"tools"`
}

func Default() (*Catalog, error) {
	return Load(defaultAssets, "assets/catalog.json")
}

func Load(assets fs.FS, manifestPath string) (*Catalog, error) {
	content, err := fs.ReadFile(assets, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var manifest document
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("decode catalog: trailing content")
	}
	result := &Catalog{Tools: manifest.Tools, assets: assets}
	if err := result.validate(manifest.SchemaVersion); err != nil {
		return nil, err
	}
	return result, nil
}

func (catalog *Catalog) Tool(id string) (Tool, bool) {
	for _, tool := range catalog.Tools {
		if tool.ID == id {
			return tool, true
		}
	}
	return Tool{}, false
}

func (catalog *Catalog) Content(artifact Artifact) ([]byte, error) {
	if artifact.Source != SourceEmbedded {
		return nil, fmt.Errorf("artifact %q is not embedded", artifact.Name)
	}
	content, err := fs.ReadFile(catalog.assets, artifact.Path)
	if err != nil {
		return nil, fmt.Errorf("read embedded artifact %q: %w", artifact.Path, err)
	}
	return content, nil
}
