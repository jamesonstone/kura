package worktree

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type sweepConfigDocument struct {
	raw      sweepYAML
	node     *yaml.Node
	contents []byte
	exists   bool
}

func newSweepYAMLDefaults() sweepYAML {
	raw := sweepYAML{Version: 1}
	includeBuiltins, sizes := true, true
	raw.Sweep.IncludeBuiltinRoots = &includeBuiltins
	raw.Sweep.Sizes.Enabled = &sizes
	return raw
}

func readSweepConfigDocument(path string) (sweepConfigDocument, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return sweepConfigDocument{raw: newSweepYAMLDefaults(), node: newSweepYAMLNode()}, nil
	}
	if err != nil {
		return sweepConfigDocument{}, fmt.Errorf("inspect sweep config %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return sweepConfigDocument{}, fmt.Errorf("sweep config must be a regular non-symlink file: %s", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return sweepConfigDocument{}, fmt.Errorf("read sweep config %s: %w", path, err)
	}
	raw, err := decodeSweepYAML(contents)
	if err != nil {
		return sweepConfigDocument{}, fmt.Errorf("decode sweep config %s: %w", path, err)
	}
	node := &yaml.Node{}
	if err := yaml.Unmarshal(contents, node); err != nil {
		return sweepConfigDocument{}, fmt.Errorf("decode sweep config nodes %s: %w", path, err)
	}
	return sweepConfigDocument{raw: raw, node: node, contents: contents, exists: true}, nil
}

func decodeSweepYAML(contents []byte) (sweepYAML, error) {
	raw := sweepYAML{}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return sweepYAML{}, err
	}
	if raw.Version != 1 {
		return sweepYAML{}, fmt.Errorf("sweep config version must be 1")
	}
	if raw.Sweep.IncludeBuiltinRoots == nil {
		raw.Sweep.IncludeBuiltinRoots = boolSweepPointer(true)
	}
	if raw.Sweep.Sizes.Enabled == nil {
		raw.Sweep.Sizes.Enabled = boolSweepPointer(true)
	}
	return raw, nil
}

func renderSweepConfig(document sweepConfigDocument, raw sweepYAML) ([]byte, error) {
	node := document.node
	if node == nil || len(node.Content) == 0 {
		node = newSweepYAMLNode()
	}
	root, err := sweepYAMLRoot(node)
	if err != nil {
		return nil, err
	}
	setSweepYAMLValue(root, "version", scalarSweepYAML("!!int", "1"))
	sweep := sweepYAMLMapping(root, "sweep")
	include := true
	if raw.Sweep.IncludeBuiltinRoots != nil {
		include = *raw.Sweep.IncludeBuiltinRoots
	}
	setSweepYAMLValue(sweep, "include_builtin_roots", scalarSweepYAML("!!bool", strconv.FormatBool(include)))
	setSweepYAMLValue(sweep, "roots", sequenceSweepYAML(findSweepYAMLValue(sweep, "roots"), raw.Sweep.Roots))
	setSweepYAMLValue(sweep, "project_roots", sequenceSweepYAML(findSweepYAMLValue(sweep, "project_roots"), raw.Sweep.ProjectRoots))
	setSweepYAMLValue(sweep, "exclude_roots", sequenceSweepYAML(findSweepYAMLValue(sweep, "exclude_roots"), raw.Sweep.ExcludeRoots))
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func newSweepYAMLNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
}

func sweepYAMLRoot(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.DocumentNode || len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("sweep config root must be a mapping")
	}
	return node.Content[0], nil
}

func sweepYAMLMapping(parent *yaml.Node, key string) *yaml.Node {
	if existing := findSweepYAMLValue(parent, key); existing != nil && existing.Kind == yaml.MappingNode {
		return existing
	}
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setSweepYAMLValue(parent, key, mapping)
	return mapping
}

func findSweepYAMLValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func setSweepYAMLValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value != key {
			continue
		}
		copySweepYAMLComments(value, mapping.Content[index+1])
		mapping.Content[index+1] = value
		return
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

func copySweepYAMLComments(destination, source *yaml.Node) {
	destination.HeadComment = source.HeadComment
	destination.LineComment = source.LineComment
	destination.FootComment = source.FootComment
}

func scalarSweepYAML(tag, value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
}

func sequenceSweepYAML(existing *yaml.Node, values []string) *yaml.Node {
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	if existing != nil {
		copySweepYAMLComments(sequence, existing)
	}
	for _, value := range values {
		var item *yaml.Node
		if existing != nil && existing.Kind == yaml.SequenceNode {
			for _, candidate := range existing.Content {
				if candidate.Value == value {
					item = candidate
					break
				}
			}
		}
		if item == nil {
			item = scalarSweepYAML("!!str", value)
		}
		sequence.Content = append(sequence.Content, item)
	}
	return sequence
}

func writeSweepConfig(path string, document sweepConfigDocument, contents []byte) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to replace symlink sweep config %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create sweep config directory: %w", err)
	}
	if document.exists {
		if err := atomicWriteSweepFile(path+".bak", document.contents, 0o600); err != nil {
			return fmt.Errorf("write sweep config backup: %w", err)
		}
	}
	if err := atomicWriteSweepFile(path, contents, 0o600); err != nil {
		return fmt.Errorf("write sweep config: %w", err)
	}
	return nil
}

func atomicWriteSweepFile(path string, contents []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".git-wt-config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return replaceSweepFile(temporaryPath, path)
}
