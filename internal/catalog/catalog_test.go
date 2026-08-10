package catalog

import (
	"testing"
	"testing/fstest"
)

func TestDefaultCatalogContainsGitWorktree(t *testing.T) {
	catalog, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := catalog.Tool("git-wt")
	if !ok {
		t.Fatal("default catalog does not contain git-wt")
	}
	if tool.Name != "git wt" || len(tool.Artifacts) != 2 {
		t.Fatalf("git-wt = %#v", tool)
	}
	content, err := catalog.Content(tool.Artifacts[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Fatal("embedded git-wt manpage is empty")
	}
}

func TestLoadAcceptsGenericEmbeddedScript(t *testing.T) {
	catalog, err := loadTestCatalog(`{
  "schema_version": 1,
  "tools": [{
    "id": "hello",
    "name": "hello",
    "description": "Print a greeting",
    "artifacts": [{
      "source": "embedded",
      "path": "assets/scripts/hello",
      "destination": "bin",
      "name": "hello",
      "mode": 493
    }]
  }]
}`, map[string]string{"assets/scripts/hello": "#!/bin/sh\necho hello\n"})
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := catalog.Tool("hello")
	if !ok {
		t.Fatal("generic script tool was not loaded")
	}
	content, err := catalog.Content(tool.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "#!/bin/sh\necho hello\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestLoadRejectsInvalidCatalogEntries(t *testing.T) {
	tests := map[string]string{
		"schema":        `{"schema_version":2,"tools":[]}`,
		"unknown field": `{"schema_version":1,"unexpected":true,"tools":[]}`,
		"trailing data": `{"schema_version":1,"tools":[]} {}`,
		"empty":         `{"schema_version":1,"tools":[]}`,
		"invalid id":    oneTool(`"id":"../bad"`, validArtifact()),
		"duplicate id":  `{"schema_version":1,"tools":[` + toolJSON("same", validArtifact()) + `,` + toolJSON("same", validArtifact()) + `]}`,
		"unsafe name":   oneTool(`"id":"test"`, artifactJSON("embedded", "assets/script", "bin", "../script", 493, "")),
		"windows name":  oneTool(`"id":"test"`, artifactJSON("embedded", "assets/script", "bin", `..\\script`, 493, "")),
		"missing asset": oneTool(`"id":"test"`, artifactJSON("embedded", "assets/missing", "bin", "script", 493, "")),
		"destination":   oneTool(`"id":"test"`, artifactJSON("embedded", "assets/script", "other", "script", 493, "")),
		"mode":          oneTool(`"id":"test"`, artifactJSON("embedded", "assets/script", "bin", "script", 420, "")),
		"self alias":    oneTool(`"id":"test"`, artifactJSON("self", "", "bin", "unknown", 493, "")),
		"platform":      oneTool(`"id":"test"`, artifactJSON("embedded", "assets/script", "bin", "script", 493, `,"platforms":["linux","linux"]`)),
	}
	for name, manifest := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := loadTestCatalog(manifest, map[string]string{"assets/script": "echo test"}); err == nil {
				t.Fatalf("Load() unexpectedly accepted %s", manifest)
			}
		})
	}
}

func loadTestCatalog(manifest string, assets map[string]string) (*Catalog, error) {
	files := fstest.MapFS{"catalog.json": {Data: []byte(manifest)}}
	for name, content := range assets {
		files[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return Load(files, "catalog.json")
}

func oneTool(idField, artifact string) string {
	return `{"schema_version":1,"tools":[{` + idField + `,"name":"test","description":"test","artifacts":[` + artifact + `]}]}`
}

func toolJSON(id, artifact string) string {
	return `{"id":"` + id + `","name":"test","description":"test","artifacts":[` + artifact + `]}`
}

func validArtifact() string {
	return artifactJSON("embedded", "assets/script", "bin", "script", 493, "")
}

func artifactJSON(source, path, destination, name string, mode int, suffix string) string {
	return `{"source":"` + source + `","path":"` + path + `","destination":"` + destination + `","name":"` + name + `","mode":` + integer(mode) + suffix + `}`
}

func integer(value int) string {
	if value == 493 {
		return "493"
	}
	return "420"
}
