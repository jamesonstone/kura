package install

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
)

const manifestSchemaVersion = 1

type ownershipManifest struct {
	SchemaVersion int                    `json:"schema_version"`
	Files         map[string]ownedRecord `json:"files"`
}

type ownedRecord struct {
	ToolID string `json:"tool_id"`
	Digest string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}

func (service *Service) loadManifest(path string) (ownershipManifest, fileExpectation, []byte, error) {
	manifest := ownershipManifest{SchemaVersion: manifestSchemaVersion, Files: map[string]ownedRecord{}}
	expectation, err := service.inspectFile(path)
	if err != nil {
		return manifest, fileExpectation{}, nil, err
	}
	if !expectation.exists {
		return manifest, expectation, nil, nil
	}
	if expectation.kind != 0 {
		return manifest, expectation, nil, fmt.Errorf("kura state %q is not a regular file", path)
	}
	content, err := service.files.readFile(path)
	if err != nil {
		return manifest, expectation, nil, fmt.Errorf("read Kura state %q: %w", path, err)
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, expectation, nil, fmt.Errorf("decode Kura state %q: %w", path, err)
	}
	if manifest.SchemaVersion != manifestSchemaVersion || manifest.Files == nil {
		return manifest, expectation, nil, fmt.Errorf("kura state %q has an unsupported schema", path)
	}
	return manifest, expectation, content, nil
}

func encodeManifest(manifest ownershipManifest) ([]byte, error) {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode Kura state: %w", err)
	}
	return append(content, '\n'), nil
}

func (service *Service) inspectFile(path string) (fileExpectation, error) {
	info, err := service.files.lstat(path)
	if os.IsNotExist(err) {
		return fileExpectation{}, nil
	}
	if err != nil {
		return fileExpectation{}, fmt.Errorf("inspect %q: %w", path, err)
	}
	if info.Mode().IsRegular() {
		content, readErr := service.files.readFile(path)
		if readErr != nil {
			return fileExpectation{}, fmt.Errorf("read %q: %w", path, readErr)
		}
		return fileExpectation{exists: true, perm: info.Mode().Perm(), digest: digest(content)}, nil
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		target, readErr := service.files.readlink(path)
		if readErr != nil {
			return fileExpectation{}, fmt.Errorf("read symlink %q: %w", path, readErr)
		}
		return fileExpectation{exists: true, kind: fs.ModeSymlink, perm: info.Mode().Perm(), link: target}, nil
	}
	return fileExpectation{exists: true, kind: info.Mode().Type(), perm: info.Mode().Perm()}, nil
}
