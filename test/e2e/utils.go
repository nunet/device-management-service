package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

type DIDData struct {
	DID struct {
		URI string `json:"uri"`
	} `json:"DID"`
	Roots   []interface{}          `json:"Roots"`
	Require map[string]interface{} `json:"Require"`
	Provide map[string]interface{} `json:"Provide"`
}

func countDIDOccurrences(input string) int {
	pattern := `"DID":\s*{[^}]*}`

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return 0
	}

	matches := re.FindAllString(input, -1)

	return len(matches)
}

func getCurrentFileDirectory() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get current file info")
	}
	return filepath.Dir(file)
}

func extractURIFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var data DIDData
	if err := json.Unmarshal(content, &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data.DID.URI, nil
}

func extractEnsembleID(input string) string {
	re := regexp.MustCompile(`"EnsembleID":\s*"(.*?)"`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func extractStatus(input string) string {
	re := regexp.MustCompile(`"Status":\s*"(.*?)"`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
