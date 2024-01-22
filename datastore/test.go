package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
)

func (m *StorageManager) TestInput(testID int) (io.ReadCloser, error) {
	return openGzipOrNormal(m.TestInputPath(testID))
}

func (m *StorageManager) TestOutput(testID int) (io.ReadCloser, error) {
	return openGzipOrNormal(m.TestOutputPath(testID))
}

func (m *StorageManager) TestInputPath(testID int) string {
	return path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".in")
}

func (m *StorageManager) TestOutputPath(testID int) string {
	return path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".out")
}

func (m *StorageManager) SaveTestInput(testID int, input io.Reader) error {
	return writeCompressedFile(m.TestInputPath(testID), input, 0644)
}

func (m *StorageManager) SaveTestOutput(testID int, output io.Reader) error {
	return writeCompressedFile(m.TestOutputPath(testID), output, 0644)
}

func (m *StorageManager) PurgeTestData(testID int) error {
	err1 := os.Remove(m.TestInputPath(testID) + ".gz")
	err2 := os.Remove(m.TestOutputPath(testID) + ".gz")
	if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
		return err1
	}
	if err2 != nil && !errors.Is(err2, fs.ErrNotExist) {
		return err2
	}
	err3 := os.Remove(m.TestInputPath(testID))
	err4 := os.Remove(m.TestOutputPath(testID))
	if err3 != nil && !errors.Is(err3, fs.ErrNotExist) {
		return err3
	}
	if err4 != nil && !errors.Is(err4, fs.ErrNotExist) {
		return err4
	}
	return nil
}
