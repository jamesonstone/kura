package install

import (
	"os"
)

type fileOperations struct {
	lstat      func(string) (os.FileInfo, error)
	readFile   func(string) ([]byte, error)
	readlink   func(string) (string, error)
	mkdirAll   func(string, os.FileMode) error
	createTemp func(string, string) (*os.File, error)
	rename     func(string, string) error
	remove     func(string) error
}

func systemFileOperations() fileOperations {
	return fileOperations{
		lstat:      os.Lstat,
		readFile:   os.ReadFile,
		readlink:   os.Readlink,
		mkdirAll:   os.MkdirAll,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
	}
}
