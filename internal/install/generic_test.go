package install

import (
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/jamesonstone/kura/internal/catalog"
)

func TestInstallMultipleGenericScripts(t *testing.T) {
	manifest := `{
  "schema_version": 1,
  "tools": [
    {"id":"hello","name":"hello","description":"hello script","artifacts":[
      {"source":"embedded","path":"assets/hello","destination":"bin","name":"hello","mode":493}
    ]},
    {"id":"bye","name":"bye","description":"bye script","artifacts":[
      {"source":"embedded","path":"assets/bye","destination":"bin","name":"bye","mode":493}
    ]}
  ]
}`
	toolCatalog := genericCatalog(t, manifest)
	root := t.TempDir()
	service := NewService(toolCatalog, filepath.Join(root, "kura"), "linux")
	service.environment.getenv = func(name string) string {
		if name == "PATH" {
			return filepath.Join(root, "bin")
		}
		return ""
	}
	service.environment.homeDir = func() (string, error) { return root, nil }
	options := Options{BinDir: filepath.Join(root, "bin"), ManDir: filepath.Join(root, "man"), StateDir: filepath.Join(root, "state")}
	report, err := service.Install(context.Background(), []string{"hello", "bye"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"hello": StatusInstalled, "bye": StatusInstalled})
	assertFile(t, filepath.Join(options.BinDir, "hello"), []byte("#!/bin/sh\necho hello\n"), 0o755)
	assertFile(t, filepath.Join(options.BinDir, "bye"), []byte("#!/bin/sh\necho bye\n"), 0o755)
}

func TestInstallRejectsDuplicateDestinationAcrossTools(t *testing.T) {
	manifest := `{
  "schema_version": 1,
  "tools": [
    {"id":"one","name":"one","description":"one","artifacts":[
      {"source":"embedded","path":"assets/hello","destination":"bin","name":"shared","mode":493}
    ]},
    {"id":"two","name":"two","description":"two","artifacts":[
      {"source":"embedded","path":"assets/bye","destination":"bin","name":"shared","mode":493}
    ]}
  ]
}`
	toolCatalog := genericCatalog(t, manifest)
	root := t.TempDir()
	service := NewService(toolCatalog, filepath.Join(root, "kura"), "linux")
	service.environment.homeDir = func() (string, error) { return root, nil }
	options := Options{BinDir: filepath.Join(root, "bin"), ManDir: filepath.Join(root, "man"), StateDir: filepath.Join(root, "state")}
	if _, err := service.Install(context.Background(), []string{"one", "two"}, options); err == nil {
		t.Fatal("duplicate destination was accepted")
	}
	if entries, err := service.files.lstat(options.BinDir); err == nil || entries != nil {
		t.Fatalf("failed preflight created bin directory: info=%v err=%v", entries, err)
	}
}

func TestInstallRejectsCaseVariantDestinationsOnWindows(t *testing.T) {
	manifest := `{
  "schema_version": 1,
  "tools": [
    {"id":"one","name":"one","description":"one","artifacts":[
      {"source":"embedded","path":"assets/hello","destination":"bin","name":"shared","mode":493}
    ]},
    {"id":"two","name":"two","description":"two","artifacts":[
      {"source":"embedded","path":"assets/bye","destination":"bin","name":"Shared","mode":493}
    ]}
  ]
}`
	toolCatalog := genericCatalog(t, manifest)
	root := t.TempDir()
	service := NewService(toolCatalog, filepath.Join(root, "kura.exe"), "windows")
	service.environment.homeDir = func() (string, error) { return root, nil }
	options := Options{BinDir: filepath.Join(root, "bin"), ManDir: filepath.Join(root, "man"), StateDir: filepath.Join(root, "state")}
	if _, err := service.Install(context.Background(), []string{"one", "two"}, options); err == nil {
		t.Fatal("case-variant Windows destinations were accepted")
	}
}

func genericCatalog(t *testing.T, manifest string) *catalog.Catalog {
	t.Helper()
	files := fstest.MapFS{
		"catalog.json": {Data: []byte(manifest)},
		"assets/hello": {Data: []byte("#!/bin/sh\necho hello\n")},
		"assets/bye":   {Data: []byte("#!/bin/sh\necho bye\n")},
	}
	toolCatalog, err := catalog.Load(files, "catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	return toolCatalog
}
