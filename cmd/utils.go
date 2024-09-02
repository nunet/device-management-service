package cmd

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func checkOnboarded(client *utils.HTTPClient) (bool, error) {
	resBody, resCode, err := client.MakeRequest("GET", "/onboarding/status", nil)
	if err != nil {
		return false, fmt.Errorf("unable to make request: %w", err)
	}

	if resCode != 200 {
		return false, fmt.Errorf("request failed with status code: %d", resCode)
	}

	onboarded, err := jsonparser.GetBoolean(resBody, "onboarded")
	if err != nil {
		return false, fmt.Errorf("could not get onboard status: %w", err)
	}
	return onboarded, nil
}

// promptReonboard is a wrapper of utils.PromptYesNo with custom prompt that return error if user declines reonboard
func promptReonboard(r io.Reader, w io.Writer) error {
	prompt := "Looks like your machine is already onboarded. Proceed with reonboarding?"

	confirmed, err := utils.PromptYesNo(r, w, prompt)
	if err != nil {
		return fmt.Errorf("could not confirm reonboarding: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("reonboarding aborted by user")
	}

	return nil
}

// getDHTPeers fetches API to retrieve info from DHT peers
func getDHTPeers(client *utils.HTTPClient) ([]string, error) {
	resBody, resCode, err := client.MakeRequest("GET", "/peers/dht", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot make request: %w", err)
	}

	if resCode != 200 {
		return nil, fmt.Errorf("request failed with status code: %d", resCode)
	}

	msg, err := jsonparser.GetString(resBody, "message")
	if err == nil {
		return nil, errors.New(msg)
	}

	var dhtSlice []string
	if _, err = jsonparser.ArrayEach(resBody, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
		dhtSlice = append(dhtSlice, string(value))
	}); err != nil {
		return nil, fmt.Errorf("cannot iterate over DHT peer list: %w", err)
	}

	if len(dhtSlice) == 0 {
		return nil, fmt.Errorf("no DHT peers available")
	}

	return dhtSlice, nil
}

// getBootstrapPeers fetches API to retrieve data from bootstrap peers
func getBootstrapPeers(w io.Writer, client *utils.HTTPClient) ([]string, error) {
	resBody, resCode, err := client.MakeRequest("GET", "/peers", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to make request: %w", err)
	}

	if resCode != 200 {
		return nil, fmt.Errorf("request failed with status code: %d", resCode)
	}

	msg, err := jsonparser.GetString(resBody, "message")
	if err == nil {
		return nil, errors.New(msg)
	}

	var bootSlice []string
	if _, err = jsonparser.ArrayEach(resBody, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
		id, err := jsonparser.GetString(value, "ID")
		if err != nil {
			fmt.Fprintln(w, "Error getting bootstrap peer ID string:", err)
			os.Exit(1)
		}

		bootSlice = append(bootSlice, id)
	}); err != nil {
		return nil, fmt.Errorf("cannot iterate over bootstrap peer list: %w", err)
	}

	if len(bootSlice) == 0 {
		return nil, fmt.Errorf("no bootstrap peers available")
	}

	return bootSlice, nil
}

// printWallet takes types.BlockchainAddressPrivKey struct as input and display it in YAML-like format for better readability
func printWallet(w io.Writer, pair *types.BlockchainAddressPrivKey) {
	if pair.Address != "" {
		fmt.Fprintf(w, "address: %s\n", pair.Address)
	}

	if pair.PrivateKey != "" {
		fmt.Fprintf(w, "private_key: %s\n", pair.PrivateKey)
	}

	if pair.Mnemonic != "" {
		fmt.Fprintf(w, "mnemonic: %s\n", pair.Mnemonic)
	}
}

func setupTable(w io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(w)
	headers := []string{"Resources", "Memory", "CPU", "Cores"}
	table.SetHeader(headers)
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	table.SetAutoFormatHeaders(false)
	return table
}

// appendToFile opens filename and write string data to it
func appendToFile(afs afero.Afero, filename, data string) error {
	// nolint:gofumpt
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
