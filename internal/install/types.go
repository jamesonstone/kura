package install

import (
	"context"
	"io/fs"

	"github.com/jamesonstone/kura/internal/catalog"
)

type Status string

const (
	StatusInstalled Status = "installed"
	StatusUpdated   Status = "updated"
	StatusReplaced  Status = "replaced"
	StatusUnchanged Status = "unchanged"
)

type Result struct {
	ToolID string
	Path   string
	Status Status
}

type Report struct {
	Results  []Result
	BinDir   string
	ManDir   string
	StateDir string
	Warnings []string
}

type Options struct {
	BinDir   string
	ManDir   string
	StateDir string
	Force    bool
}

type Installer interface {
	Install(context.Context, []string, Options) (Report, error)
}

type Service struct {
	Catalog        *catalog.Catalog
	ExecutablePath string
	GOOS           string
	environment    pathEnvironment
	files          fileOperations
}

func NewService(toolCatalog *catalog.Catalog, executablePath, goos string) *Service {
	return &Service{
		Catalog:        toolCatalog,
		ExecutablePath: executablePath,
		GOOS:           goos,
		environment:    systemPathEnvironment(),
		files:          systemFileOperations(),
	}
}

type plannedArtifact struct {
	result   Result
	mode     fs.FileMode
	content  []byte
	digest   string
	expected fileExpectation
	write    bool
}

type fileExpectation struct {
	exists bool
	kind   fs.FileMode
	perm   fs.FileMode
	digest string
	link   string
}

type stagedWrite struct {
	destination string
	staged      string
	expected    fileExpectation
	backup      string
	committed   bool
}
