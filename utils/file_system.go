package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

func GetDirectorySize(fs afero.Fs, path string) (int64, error) {
	var size int64
	err := afero.Walk(fs, path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to calculate volume size: %w", err)
	}

	return size, nil
}

// WriteToFile writes data to a file.
func WriteToFile(fs afero.Fs, data []byte, filePath string) (string, error) {
	if err := fs.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to open path: %w", err)
	}
	file, err := fs.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create path: %w", err)
	}
	defer file.Close()
	n, err := file.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write data to path: %w", err)
	}

	if n != len(data) {
		return "", errors.New("failed to write the size of data to file")
	}
	return filePath, nil
}

// FileExists checks if destination file exists
func FileExists(fs afero.Fs, filename string) bool {
	info, err := fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
