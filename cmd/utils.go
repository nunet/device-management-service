package cmd

import (
	"archive/tar"
	"compress/gzip"
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

	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/models"

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
		if conn.Status == "LISTEN" && uint32(port) == conn.Laddr.Port {
			return true, nil
		}
	}

	return false, nil
}

// isDMSRunning is intended to be used as a PreRun hook and ensure that DMS
// is running before command execution
func isDMSRunning(net backend.NetworkManager) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("could not check onboard status: %w", err)
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
	reserved := models.CapacityForNunet{
		Memory:            memory,
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
func getIncomingChatList(body []byte) (string, error) {
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
	} else {
		chatID, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("argument is not integer")
		}
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
		return fmt.Errorf("no peer ID specified")
	} else if len(args) > 1 {
		return fmt.Errorf("cannot start multiple chats")
	} else {
		_, err := p2pService.Decode(args[0])
		if err != nil {
			return fmt.Errorf("invalid peer ID: %w", err)
		}
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
		return nil, fmt.Errorf(errMsg)
	}
	msg, err := jsonparser.GetString(bodyDht, "message")
	if err == nil {
		return nil, fmt.Errorf(msg)
	}

	_, err = jsonparser.ArrayEach(bodyDht, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
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
		return nil, fmt.Errorf(errMsg)
	}
	msg, err := jsonparser.GetString(bodyBoot, "message")
	if err == nil {
		return nil, fmt.Errorf(msg)

	}

	_, err = jsonparser.ArrayEach(bodyBoot, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		id, err := jsonparser.GetString(value, "ID")
		if err != nil {
			fmt.Fprintln(writer, "Error getting bootstrap peer ID string:", err)
			os.Exit(1)
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

// printMetadata takes models.Metadata struct as input and display it in YAML-like format for better readability
func printMetadata(w io.Writer, metadata *models.Metadata) {
	fmt.Fprintln(w, "metadata:")

	if metadata.Name != "" {
		fmt.Fprintf(w, "  name: %s\n", metadata.Name)
	}

	if metadata.UpdateTimestamp != 0 {
		fmt.Fprintf(w, "  update_timestamp: %d\n", metadata.UpdateTimestamp)
	}

	if metadata.Resource.MemoryMax != 0 || metadata.Resource.TotalCore != 0 || metadata.Resource.CPUMax != 0 {
		fmt.Fprintln(w, "  resource:")

		if metadata.Resource.MemoryMax != 0 {
			fmt.Fprintf(w, "    memory_max: %d\n", metadata.Resource.MemoryMax)
		}

		if metadata.Resource.TotalCore != 0 {
			fmt.Fprintf(w, "    total_core: %d\n", metadata.Resource.TotalCore)
		}

		if metadata.Resource.CPUMax != 0 {
			fmt.Fprintf(w, "    cpu_max: %d\n", metadata.Resource.CPUMax)
		}
	}

	if metadata.Available.CPU != 0 || metadata.Available.Memory != 0 {
		fmt.Fprintln(w, "  available:")

		if metadata.Available.CPU != 0 {
			fmt.Fprintf(w, "    cpu: %d\n", metadata.Available.CPU)
		}

		if metadata.Available.Memory != 0 {
			fmt.Fprintf(w, "    memory: %d\n", metadata.Available.Memory)
		}
	}

	if metadata.Reserved.CPU != 0 || metadata.Reserved.Memory != 0 {
		fmt.Fprintln(w, "  reserved:")

		if metadata.Reserved.CPU != 0 {
			fmt.Fprintf(w, "    cpu: %d\n", metadata.Reserved.CPU)
		}

		if metadata.Reserved.Memory != 0 {
			fmt.Fprintf(w, "    memory: %d\n", metadata.Reserved.Memory)
		}
	}

	if metadata.Network != "" {
		fmt.Fprintf(w, "  network: %s\n", metadata.Network)
	}

	if metadata.PublicKey != "" {
		fmt.Fprintf(w, "  public_key: %s\n", metadata.PublicKey)
	}

	if metadata.NodeID != "" {
		fmt.Fprintf(w, "  node_id: %s\n", metadata.NodeID)
	}

	if metadata.AllowCardano {
		fmt.Fprintf(w, "  allow_cardano: %v\n", metadata.AllowCardano)
	}

	if len(metadata.GpuInfo) > 0 {
		fmt.Fprintln(w, "  gpu_info:")
		for i, gpu := range metadata.GpuInfo {
			fmt.Fprintf(w, "    - gpu %d:\n", i+1)
			if gpu.Name != "" {
				fmt.Fprintf(w, "      name: %s\n", gpu.Name)
			}
			if gpu.TotVram != 0 {
				fmt.Fprintf(w, "      tot_vram: %d\n", gpu.TotVram)
			}
			if gpu.FreeVram != 0 {
				fmt.Fprintf(w, "      free_vram: %d\n", gpu.FreeVram)
			}
		}
	}

	if metadata.Dashboard != "" {
		fmt.Fprintf(w, "  dashboard: %s\n", metadata.Dashboard)
	}
}

// printWallet takes models.BlockchainAddressPrivKey struct as input and display it in YAML-like format for better readability
func printWallet(w io.Writer, pair *models.BlockchainAddressPrivKey) {
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

func setFullData(provisioned *models.Provisioned) []string {
	return []string{
		"Full",
		fmt.Sprintf("%d", provisioned.Memory),
		fmt.Sprintf("%.0f", provisioned.CPU),
		fmt.Sprintf("%d", provisioned.NumCores),
	}
}

func setAvailableData(metadata *models.Metadata) []string {
	return []string{
		"Available",
		fmt.Sprintf("%d", metadata.Available.Memory),
		fmt.Sprintf("%d", metadata.Available.CPU),
		"",
	}
}

func setOnboardedData(metadata *models.Metadata) []string {
	return []string{
		"Onboarded",
		fmt.Sprintf("%d", metadata.Reserved.Memory),
		fmt.Sprintf("%d", metadata.Reserved.CPU),
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

func handleAvailable(table *tablewriter.Table, utilsService backend.Utility) error {
	err := checkOnboarded(utilsService)
	if err != nil {
		return err
	}

	metadata, err := utilsService.ReadMetadataFile()
	if err != nil {
		return fmt.Errorf("cannot read metadata file: %w", err)
	}

	availableData := setAvailableData(metadata)
	table.Append(availableData)

	return nil
}

func handleOnboarded(table *tablewriter.Table, utilsService backend.Utility) error {
	err := checkOnboarded(utilsService)
	if err != nil {
		return err
	}

	metadata, err := utilsService.ReadMetadataFile()
	if err != nil {
		return fmt.Errorf("cannot read metadata file: %w", err)
	}

	onboardedData := setOnboardedData(metadata)
	table.Append(onboardedData)

	return nil
}

// appendToFile opens filename and write string data to it
func appendToFile(fs backend.FileSystem, filename, data string) error {
	f, err := fs.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
