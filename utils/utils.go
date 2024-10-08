package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/exp/slices"

	"gitlab.com/nunet/device-management-service/types"
)

const (
	KernelFileURL  = "https://d.nunet.io/fc/vmlinux"
	KernelFilePath = "/etc/nunet/vmlinux"
	FilesystemURL  = "https://d.nunet.io/fc/nunet-fc-ubuntu-20.04-0.ext4"
	FilesystemPath = "/etc/nunet/nunet-fc-ubuntu-20.04-0.ext4"
)

// DownloadFile downloads a file from a url and saves it to a filepath
func DownloadFile(url string, filepath string, maxBytes int64) (err error) {
	zlog.Sugar().Infof("Downloading file '", filepath, "' from '", url, "'")
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: server returned %s", resp.Status)
	}

	reader := io.LimitReader(resp.Body, maxBytes)
	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}
	log.Println("Finished downloading file '", filepath, "'")
	return nil
}

// ReadHTTPString GET request to http endpoint and return response as string
func ReadHTTPString(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}

// RandomString generates a random string of length n
func RandomString(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}

		sb.WriteByte(charset[n.Int64()])
	}
	return sb.String(), nil
}

// SliceContains checks if a string exists in a slice
func SliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// DeleteFile deletes a file, with or without a backup
func DeleteFile(path string, backup bool) (err error) {
	if backup {
		err = os.Rename(path, fmt.Sprintf("%s.bk.%d", path, time.Now().Unix()))
	} else {
		err = os.Remove(path)
	}
	return
}

// CreateDirectoryIfNotExists creates a directory if it does not exist
func CreateDirectoryIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0o755)
		if err != nil {
			return err
		}
	}
	return nil
}

// CalculateSHA256Checksum calculates the SHA256 checksum of a file
func CalculateSHA256Checksum(filePath string) (string, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a new SHA-256 hash
	hash := sha256.New()

	// Copy the file's contents into the hash object
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Calculate the checksum and return it as a hexadecimal string
	checksum := hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}

// put checksum in file
func CreateCheckSumFile(filePath string, checksum string) (string, error) {
	sha256FilePath := fmt.Sprintf("%s.sha256.txt", filePath)
	sha256File, err := os.Create(sha256FilePath)
	if err != nil {
		return "", fmt.Errorf("unable to create SHA-256 checksum file: %v", err)
	}

	defer sha256File.Close()

	_, err = sha256File.WriteString(checksum)
	if err != nil {
		return "", fmt.Errorf("unable to write to SHA-256 checksum file: %v", err)
	}

	return sha256FilePath, nil
}

// SanitizeArchivePath Sanitize archive file pathing from "G305: Zip Slip vulnerability"
func SanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

// ExtractTarGzToPath extracts a tar.gz file to a specified path
func ExtractTarGzToPath(tarGzFilePath, extractedPath string, maxBytes int64) error {
	// Ensure the target directory exists; create it if it doesn't.
	if err := os.MkdirAll(extractedPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	tarGzFile, err := os.Open(tarGzFilePath)
	if err != nil {
		return fmt.Errorf("error opening tar.gz file: %v", err)
	}
	defer tarGzFile.Close()

	gzipReader, err := gzip.NewReader(tarGzFile)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	var totalSize int64

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar header: %v", err)
		}

		if header.Size > maxBytes {
			return fmt.Errorf("file %s exceeds the maximum allowed size of %d bytes", header.Name, maxBytes)
		}

		// Construct the full target path by joining the target directory with
		// the name of the file or directory from the archive.
		fullTargetPath, err := SanitizeArchivePath(extractedPath, header.Name)
		if err != nil {
			return fmt.Errorf("failed to santize path %w", err)
		}

		// Ensure that the directory path leading to the file exists.
		if header.FileInfo().IsDir() {
			// Create the directory and any parent directories as needed.
			if err := os.MkdirAll(fullTargetPath, os.ModePerm); err != nil {
				return fmt.Errorf("error creating directory: %v", err)
			}
		} else {
			// Create the file and any parent directories as needed.
			if err := os.MkdirAll(filepath.Dir(fullTargetPath), os.ModePerm); err != nil {
				return fmt.Errorf("error creating directory: %v", err)
			}

			// Create a new file with the specified path.
			newFile, err := os.Create(fullTargetPath)
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}
			defer newFile.Close()

			// Copy the file contents from the tar archive to the new file.
			for {
				n, err := io.CopyN(newFile, tarReader, 1024)
				totalSize += n

				if totalSize > maxBytes {
					return fmt.Errorf("extracted data exceeds allowed limit of %d bytes", maxBytes)
				}

				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
			}
		}
	}

	return nil
}

// CheckWSL check if running in WSL
func CheckWSL(afs afero.Afero) (bool, error) {
	file, err := afs.Open("/proc/version")
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Microsoft") || strings.Contains(line, "WSL") {
			return true, nil
		}
	}

	if scanner.Err() != nil {
		return false, scanner.Err()
	}

	return false, nil
}

func RandomBool() (bool, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return false, err
	}
	// Return true if the number is 1, otherwise false
	return n.Int64() == 1, nil
}

func IsExecutorType(v interface{}) bool {
	_, ok := v.(types.ExecutorType)
	return ok
}

func IsGPUVendor(v interface{}) bool {
	_, ok := v.(types.GPUVendor)
	return ok
}

func IsJobType(v interface{}) bool {
	_, ok := v.(types.JobType)
	return ok
}

func IsJobTypes(v interface{}) bool {
	_, ok := v.(types.JobTypes)
	return ok
}

func IsExecutor(v interface{}) bool {
	_, ok := v.(types.Executor)
	return ok
}

// IsStrictlyContained checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContained(leftSlice, rightSlice []interface{}) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}

// IsStrictlyContainedInt checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContainedInt(leftSlice, rightSlice []int) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}

func NoIntersectionSlices(slice1, slice2 []interface{}) bool {
	result := false // the default result is false
	for _, subElement := range slice1 {
		if slices.Contains(slice2, subElement) {
			result = false
		} else {
			result = true
		}
	}
	return result
}

// IntersectionStringSlices returns the intersection of two slices of strings.
func IntersectionSlices(slice1, slice2 []interface{}) []interface{} {
	// Create a map to store strings from the first slice.
	executorMap := make(map[interface{}]bool)

	// Iterate through the first slice and add elements to the map.
	for _, str := range slice1 {
		executorMap[str] = true
	}

	// Create a slice to store the intersection of the strings.
	intersectionSlice := []interface{}{}

	// Iterate through the second slice and check for common elements.
	for _, str := range slice2 {
		if executorMap[str] {
			// If the string is found in the map, add to the intersection slice.
			intersectionSlice = append(intersectionSlice, str)
			// Remove the string from the map to avoid duplicates in the result.
			delete(executorMap, str)
		}
	}

	return intersectionSlice
}

func IsSameShallowType(a, b interface{}) bool {
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)
	result := aType == bType
	return result
}

func ConvertTypedSliceToUntypedSlice(typedSlice interface{}) []interface{} {
	s := reflect.ValueOf(typedSlice)
	if s.Kind() != reflect.Slice {
		return nil
	}
	result := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		result[i] = s.Index(i).Interface()
	}
	return result
}
