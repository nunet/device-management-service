package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	gdb "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"

	// "gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/utils"
)

func listenDMSPort(net backend.NetworkManager) (bool, error) {
	port := config.GetConfig().Rest.Port

	conns, err := net.GetConnections("all")
	if err != nil {
		return false, err
	}

	for _, conn := range conns {
		if conn.Status == "LISTEN" && port == conn.Laddr.Port {
			return true, nil
		}
	}

	return false, nil
}

// isDMSRunning is intended to be used as a PreRun hook and ensure that DMS
// is running before command execution
func isDMSRunning(net backend.NetworkManager) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		open, err := listenDMSPort(net)
		if err != nil {
			return fmt.Errorf("unable to listen on DMS port: %w", err)
		}

		if !open {
			return fmt.Errorf("looks like DMS is not running... \n\nSee: systemctl status nunet-dms.service")
		}

		return nil
	}
}

// checkOnboarded is a wrapper of utils.IsOnboarded() that prevents command execution if not onboarded
func checkOnboarded(utilsService backend.Utility) error {
	onboarded, err := utilsService.IsOnboarded()
	if err != nil {
		return fmt.Errorf(
			"could not check onboard status. It may be an internal error or user is not onboarded: %w",
			err,
		)
	}

	if !onboarded {
		return fmt.Errorf("current machine is not onboarded")
	}

	return nil
}

// promptReonboard is a wrapper of utils.PromptYesNo with custom prompt that return error if user declines reonboard
func promptReonboard(reader io.Reader, writer io.Writer) error {
	reonboardPrompt := "Looks like your machine is already onboarded. Proceed with reonboarding?"

	confirmed, err := utils.PromptYesNo(reader, writer, reonboardPrompt)
	if err != nil {
		return fmt.Errorf("could not confirm reonboarding: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("reonboarding aborted by user")
	}

	return nil
}

// setOnboardData takes all onboarding parameters and marshal them into JSON
func setOnboardData(memory int64, cpu int64, ntxPrice float64, channel, address string, cardano, serverMode, isAvailable bool) ([]byte, error) {
	reserved := types.CapacityForNunet{
		Memory:            uint64(memory),
		CPU:               cpu,
		Channel:           channel,
		PaymentAddress:    address,
		NTXPricePerMinute: ntxPrice,
		Cardano:           cardano,
		ServerMode:        serverMode,
		IsAvailable:       isAvailable,
	}

	data, err := json.Marshal(reserved)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal JSON data: %w", err)
	}

	return data, nil
}

// TODO: Handle this after refactor
// getIncomingChatList unmarshal response body from API request into
// libp2p.OpenStream slice and return list of chats
// func getIncomingChatList(body []byte) ([]libp2p.OpenStream, error) {
// 	var chatList []libp2p.OpenStream
// 	err := json.Unmarshal(body, &chatList)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to unmarshal response body: %w", err)
// 	}

//		return chatList, nil
//	}
func getIncomingChatList(_ []byte) (string, error) {
	err := errors.New("getIncomingChatList not implemented")
	return "", err
}

// END

func validateJoinChatInput(args []string, chatList []byte) error {
	var chatID int
	var err error

	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("no chat ID specified")
	} else if len(args) > 1 {
		return fmt.Errorf("unable to join multiple chats")
	}

	chatID, err = strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("argument is not integer")
	}

	openChats, err := getIncomingChatList(chatList)
	if err != nil {
		return err
	}

	if chatID >= len(openChats) {
		return fmt.Errorf("no incoming stream match chat ID specified")
	}

	return nil
}

func validateStartChatInput(p2pService backend.PeerManager, args []string) error {
	if len(args) == 0 || args[0] == "" {
		return errors.New("no peer ID specified")
	} else if len(args) > 1 {
		return errors.New("cannot start multiple chats")
	}

	_, err := p2pService.Decode(args[0])
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	return nil
}

func setupChatTable(writer io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(writer)

	table.SetHeader([]string{"ID", "Stream ID", "From Peer", "Time Opened"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	return table
}

// getDHTPeers fetches API to retrieve info from DHT peers
func getDHTPeers(utilsService backend.Utility) ([]string, error) {
	var dhtSlice []string

	bodyDht, err := utilsService.ResponseBody(nil, "GET", "/api/v1/peers/dht", "", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get response body: %w", err)
	}

	errMsg, err := jsonparser.GetString(bodyDht, "error")
	if err == nil {
		return nil, errors.New(errMsg)
	}
	msg, err := jsonparser.GetString(bodyDht, "message")
	if err == nil {
		return nil, errors.New(msg)
	}

	_, err = jsonparser.ArrayEach(bodyDht, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
		dhtSlice = append(dhtSlice, string(value))
	})
	if err != nil {
		return nil, fmt.Errorf("cannot iterate over DHT peer list: %w", err)
	}

	if len(dhtSlice) == 0 {
		return nil, fmt.Errorf("no DHT peers available")
	}

	return dhtSlice, nil
}

// getBootstrapPeers fetches API to retrieve data from bootstrap peers
func getBootstrapPeers(writer io.Writer, utilsService backend.Utility) ([]string, error) {
	var bootSlice []string

	bodyBoot, err := utilsService.ResponseBody(nil, "GET", "/api/v1/peers", "", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get response body: %w", err)
	}

	errMsg, err := jsonparser.GetString(bodyBoot, "error")
	if err == nil {
		return nil, errors.New(errMsg)
	}
	msg, err := jsonparser.GetString(bodyBoot, "message")
	if err == nil {
		return nil, errors.New(msg)
	}

	_, err = jsonparser.ArrayEach(bodyBoot, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
		id, err := jsonparser.GetString(value, "ID")
		if err != nil {
			fmt.Fprintln(writer, "Error getting bootstrap peer ID string:", err)
			return
		}

		bootSlice = append(bootSlice, id)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot iterate over bootstrap peer list: %w", err)
	}

	if len(bootSlice) == 0 {
		return nil, fmt.Errorf("no bootstrap peers available")
	}

	return bootSlice, nil
}

func selfPeerID(body []byte) (string, error) {
	id, err := jsonparser.GetString(body, "ID")
	if err != nil {
		return "", fmt.Errorf("failed to get ID string: %w", err)
	}

	return id, nil
}

func selfPeerAddrs(body []byte) (addrsByte []byte, err error) {
	addrsByte, dataType, _, err := jsonparser.Get(body, "Addrs")
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses field: %w", err)
	}

	if dataType != jsonparser.Array {
		return nil, fmt.Errorf("invalid data type: expected addresses field is not an array")
	}

	return addrsByte, nil
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

func setFullData(provisioned *types.Provisioned) []string {
	return []string{
		"Full",
		fmt.Sprintf("%d", provisioned.Memory),
		fmt.Sprintf("%.0f", provisioned.CPU),
		fmt.Sprintf("%d", provisioned.NumCores),
	}
}

func setOnboardedData(oConf *types.OnboardingConfig) []string {
	return []string{
		"Onboarded",
		fmt.Sprintf("%d", oConf.OnboardedResources.RAM),
		fmt.Sprintf("%.2f", oConf.OnboardedResources.CPU),
		"",
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

func handleFull(table *tablewriter.Table, resources backend.ResourceManager) {
	totalProvisioned := resources.GetTotalProvisioned()

	fullData := setFullData(totalProvisioned)
	table.Append(fullData)
}

func handleOnboarded(table *tablewriter.Table, utilsService backend.Utility) error {
	err := checkOnboarded(utilsService)
	if err != nil {
		return err
	}

	// XXX: don't leave me like this
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir)), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	onboardR := gdb.NewOnboardingParams(db)
	oConf, err := onboardR.Get(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get onboarding config: %w", err)
	}

	onboardedData := setOnboardedData(&oConf)
	table.Append(onboardedData)

	return nil
}

// appendToFile opens filename and write string data to it
func appendToFile(fs backend.FileSystem, filename, data string) error {
	f, err := fs.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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

func createTar(fs backend.FileSystem, tarGzPath string, sourceDir string) error {
	tarGzFile, err := fs.Create(tarGzPath)
	if err != nil {
		return fmt.Errorf("create %s file failed: %w", tarGzPath, err)
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return fs.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
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
			data, err := fs.ReadFile(path)
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
