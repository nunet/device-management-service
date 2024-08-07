package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/models"
	"golang.org/x/exp/slices"

	"reflect"
)

var KernelFileURL = "https://d.nunet.io/fc/vmlinux"
var KernelFilePath = "/etc/nunet/vmlinux"
var FilesystemURL = "https://d.nunet.io/fc/nunet-fc-ubuntu-20.04-0.ext4"
var FilesystemPath = "/etc/nunet/nunet-fc-ubuntu-20.04-0.ext4"

// DownloadFile downloads a file from a url and saves it to a filepath
func DownloadFile(url string, filepath string) (err error) {
	zlog.Sugar().Infof("Downloading file '", filepath, "' from '", url, "'")
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	log.Println("Finished downloading file '", filepath, "'")
	return nil
}

// ReadHttpString GET request to http endpoint and return response as string
func ReadHttpString(url string) (string, error) {
	resp, err := http.Get(url)
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
func RandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// GenerateMachineUUID generates a machine uuid
func GenerateMachineUUID() (string, error) {
	var machine models.MachineUUID

	machineUUID, err := uuid.NewDCEGroup()
	if err != nil {
		return "", err
	}
	machine.UUID = machineUUID.String()

	return machine.UUID, nil
}

// GetMachineUUID returns the machine uuid from the DB
func GetMachineUUID() string {
	var machine models.MachineUUID
	uuid, err := GenerateMachineUUID()
	if err != nil {
		zlog.Sugar().Errorf("could not generate machine uuid: %v", err)
	}

	machine.UUID = uuid

	result := db.DB.FirstOrCreate(&machine)
	if result.Error != nil {
		zlog.Sugar().Errorf("could not find or create machine uuid record in DB: %v", result.Error)
	}
	return machine.UUID

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

// ReadyForElastic checks if the device is ready to send logs to elastic
func ReadyForElastic() bool {
	elasticToken := models.ElasticToken{}
	db.DB.Find(&elasticToken)
	return elasticToken.NodeId != "" && elasticToken.ChannelName != ""
}

// PromptYesNo loops on confirmation from user until valid answer
func PromptYesNo(in io.Reader, out io.Writer, prompt string) (bool, error) {
	reader := bufio.NewReader(in)

	for {
		fmt.Fprintf(out, "%s (y/N): ", prompt)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read response string failed: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true, nil
		} else if response == "n" || response == "no" {
			return false, nil
		}
	}
}

// CreateDirectoryIfNotExists creates a directory if it does not exist
func CreateDirectoryIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
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

// ExtractTarGzToPath extracts a tar.gz file to a specified path
func ExtractTarGzToPath(tarGzFilePath, extractedPath string) error {
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

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar header: %v", err)
		}

		// Construct the full target path by joining the target directory with
		// the name of the file or directory from the archive.
		fullTargetPath := filepath.Join(extractedPath, header.Name)

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
			if _, err := io.Copy(newFile, tarReader); err != nil {
				return fmt.Errorf("error copying file contents: %v", err)
			}
		}
	}

	return nil
}

// CheckWSL check if running in WSL
func CheckWSL() (bool, error) {
	file, err := os.Open("/proc/version")
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

// SaveServiceInfo updates service info into SP's DMS for claim Reward by SP user
func SaveServiceInfo(cpService models.Services) error {

	var spService models.Services
	err := db.DB.Model(&models.Services{}).Where("tx_hash = ?", cpService.TxHash).Find(&spService).Error
	if err != nil {
		return fmt.Errorf("Unable to find service on SP side: %v", err)
	}
	cpService.ID = spService.ID
	cpService.CreatedAt = spService.CreatedAt

	result := db.DB.Model(&models.Services{}).Where("tx_hash = ?", cpService.TxHash).Updates(&cpService)
	if result.Error != nil {
		return fmt.Errorf("Unable to update service info on SP side: %v", result.Error.Error())
	}

	return nil
}

func RandomBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 1
}

func IsExecutorType(v interface{}) bool {
	_, ok := v.(models.ExecutorType)
	return ok
}

func IsGPUVendor(v interface{}) bool {
	_, ok := v.(models.GPUVendor)
	return ok
}

func IsJobType(v interface{}) bool {
	_, ok := v.(models.JobType)
	return ok
}

func IsJobTypes(v interface{}) bool {
	_, ok := v.(models.JobTypes)
	return ok
}

func IsExecutor(v interface{}) bool {
	_, ok := v.(models.Executor)
	return ok
}

// IsStrictlyContained checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContained(leftSlice, rightSlice []interface{}) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		} else {
			result = true
		}
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
