package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type resolvedPaths struct {
	bin   string
	man   string
	state string
}

type pathEnvironment struct {
	getenv   func(string) string
	homeDir  func() (string, error)
	lookPath func(string) (string, error)
	stat     func(string) (os.FileInfo, error)
}

func systemPathEnvironment() pathEnvironment {
	return pathEnvironment{
		getenv:   os.Getenv,
		homeDir:  os.UserHomeDir,
		lookPath: exec.LookPath,
		stat:     os.Stat,
	}
}

func (service *Service) resolvePaths(options Options) (resolvedPaths, error) {
	home, err := service.environment.homeDir()
	if err != nil {
		return resolvedPaths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	bin := firstSet(options.BinDir, service.environment.getenv("KURA_BIN_DIR"))
	if bin == "" {
		bin = service.discoverableExecutableDir()
	}
	if bin == "" {
		if service.GOOS == "windows" {
			base := firstSet(service.environment.getenv("LOCALAPPDATA"), home)
			bin = filepath.Join(base, "Kura", "bin")
		} else {
			bin = filepath.Join(home, ".local", "bin")
		}
	}
	localAppData := firstSet(service.environment.getenv("LOCALAPPDATA"), home)
	dataHome := service.environment.getenv("XDG_DATA_HOME")
	if service.GOOS == "windows" {
		dataHome = filepath.Join(localAppData, "Kura")
	} else if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	man := firstSet(options.ManDir, service.environment.getenv("KURA_MAN_DIR"))
	if man == "" {
		man = filepath.Join(dataHome, "man", "man1")
	}
	state := firstSet(options.StateDir, service.environment.getenv("KURA_STATE_DIR"))
	if state == "" {
		if service.GOOS == "windows" {
			state = filepath.Join(localAppData, "Kura", "state")
		} else {
			stateHome := service.environment.getenv("XDG_STATE_HOME")
			if stateHome == "" {
				stateHome = filepath.Join(home, ".local", "state")
			}
			state = filepath.Join(stateHome, "kura")
		}
	}
	return absolutePaths(resolvedPaths{bin: bin, man: man, state: state})
}

func (service *Service) discoverableExecutableDir() string {
	if service.ExecutablePath == "" {
		return ""
	}
	found, err := service.environment.lookPath(filepath.Base(service.ExecutablePath))
	if err != nil {
		return ""
	}
	actualInfo, actualErr := service.environment.stat(service.ExecutablePath)
	foundInfo, foundErr := service.environment.stat(found)
	if actualErr != nil || foundErr != nil || !os.SameFile(actualInfo, foundInfo) {
		return ""
	}
	found, err = filepath.Abs(found)
	if err != nil {
		return ""
	}
	return filepath.Dir(found)
}

func absolutePaths(paths resolvedPaths) (resolvedPaths, error) {
	values := []*string{&paths.bin, &paths.man, &paths.state}
	for _, value := range values {
		absolute, err := filepath.Abs(*value)
		if err != nil {
			return resolvedPaths{}, fmt.Errorf("resolve install path %q: %w", *value, err)
		}
		*value = filepath.Clean(absolute)
	}
	return paths, nil
}

func firstSet(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func PathContains(directory, pathValue string) bool {
	for _, entry := range filepath.SplitList(pathValue) {
		left, leftErr := filepath.Abs(entry)
		right, rightErr := filepath.Abs(directory)
		if leftErr == nil && rightErr == nil && filepath.Clean(left) == filepath.Clean(right) {
			return true
		}
	}
	return false
}
