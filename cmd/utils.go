package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
)

// appendToFile opens filename and write string data to it
func appendToFile(afs afero.Afero, filename, data string) error {
	f, err := afs.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s file failed: %w", filename, err)
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return fmt.Errorf("write string data to file %s failed: %w", filename, err)
	}

	return nil
}

func createTar(afs afero.Afero, tarGzPath string, sourceDir string) error {
	tarGzFile, err := afs.Create(tarGzPath)
	if err != nil {
		return fmt.Errorf("create %s file failed: %w", tarGzPath, err)
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return afs.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == tarGzPath {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, sourceDir)
		if header.Name == "" || header.Name == "/" {
			return nil
		}

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			data, err := afs.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = tarWriter.Write(data)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
