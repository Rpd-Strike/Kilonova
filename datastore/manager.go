package datastore

import (
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/KiloProjects/kilonova"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	RootPath string

	attMu sync.RWMutex
}

var _ kilonova.DataStore = &StorageManager{}

// NewManager returns a new manager instance
func NewManager(p string) (kilonova.DataStore, error) {
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "subtests"), 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "tests"), 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "attachments"), 0755); err != nil {
		return nil, err
	}

	return &StorageManager{RootPath: p}, nil
}

func openNormalOrGzip(fpath string) (io.ReadCloser, error) {
	f, err := os.Open(fpath)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		f2, err := os.Open(fpath + ".gz")
		if err != nil {
			return f2, err
		}
		return gzip.NewReader(f2)
	}
	return f, err
}
