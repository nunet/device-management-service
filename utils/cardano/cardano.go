// This package has some basic functionality to interact with the cardano block chain and the Escrow smart contract
// it currently assumed preprod.
//
// You must have a running cardano-node synchronized on preprod and the SPAddress must have mNTX and tADA for
// the tests to run properly.
//
// You must also have cardano-cli available on your PATH

package cardano

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
	// "gitlab.com/nunet/device-management-service/integrations/oracle"
)

// NOTE: This corresponds to
// https://gitlab.com/nunet/tokenomics-api/tokenomics-api-cardano-v2/-/blob/develop/src/SimpleEscrow.hs?ref_type=heads#L103
// and must match exactly with the deployed contract
type Redeemer int

const (
	Withdraw   Redeemer = 0
	Refund     Redeemer = 1
	Distribute Redeemer = 2
)

const (
	testnetMagic = "1"

	SPAccount = "tester"
	CPAccount = "cp"

	// Address of the testing account, corresponds to tester.addr.
	SPAddress = "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk"
	CPAddress = "addr_test1vz7j9m50apx99phkat3h4sm4t82wvap65ezjpe5xl3kcuxcr6e44h"

	SPCollateral = "1ad86a8ef29adf0aeac0a9c0282a2ee5bc9a3f9a3fc8e8b1c3f086b679aed882#0"
	CPCollateral = "eb455c103d609e7b9d16056486c8203c59a7e7af312f9ecf9b5b6afc899ab71a#0"

	// Current alpha preprod NTX native asset.
	mNTX = "8cafc9b387c9f6519cacdce48a8448c062670c810d8da4b232e56313.6d4e5458"

	// These are all hex encoded ascii / ByteString of a text string
	PreGenMetaDataHash     = "612072616E646F6D20737472696E63"
	PreGenWithdrawHash     = "612072616E646F6D20737472696E64"
	PreGenRefundHash       = "612072616E646F6D20737472696E65"
	PreGenDistribute75Hash = "612072616E646F6D20737472696E66"
	PreGenDistribute50Hash = "612072616E646F6D20737472696E67"

	DATUM_FORMAT_STRING = `{
      "constructor": 0,
      "fields": [
         {
            "bytes": "%s"
         },
         {
            "bytes": "%s"
         },
         {
            "int": %d
         },
         {
            "bytes": "%s"
         },
         {
            "bytes": "%s"
         },
         {
            "bytes": "%s"
         },
         {
            "bytes": "%s"
         },
         {
            "bytes": "%s"
         }
      ]
   }`

	REDEEMER_FORMAT_STRING = `{
     "constructor": %d,
     "fields": [
      {
   		"constructor": 0,
   		"fields": [
   			{
   				"bytes" : "%s"
   			},
   			{
   				"bytes" : "%s"
   			},
   			{
   				"bytes" : "%s"
   			},
   			{
   				"bytes" : "%s"
   			},
   			{
   				"bytes" : "%s"
   			},
   			{
   				"bytes" : "%s"
   			}
   	 	]
   	  }
     ]
   }`
)

// JSONInput is a intermediate unmarshalled format for custom unmarshalling logic for Inputs
// as Inputs do not have enough.
type JSONInput struct {
	Value map[string]interface{} `json:"value"`
}

// Inputs to a transaction.
type Input struct {
	Key          string           // The key is the '<transaction-hash>#<index>'.
	Value        map[string]int64 // Value is a map from '<policy-id>.<hex-asset-name>' or 'lovelace' to token amount.
	ScriptFile   string
	DatumFile    string
	RedeemerFile string
}

// An Output to be created by a transaction
type Output struct {
	To        string           // To Address aka who will own the output
	Value     map[string]int64 // Value is a map from '<policy-id>.<hex-asset-name>' or 'lovelace' to token amount.
	DatumFile string           // This is a json file encoding the datum that will be used or nil for no datum
}

// A Cardano Transaction.
type Transaction struct {
	Inputs        []Input           // The inputs to the transaction.
	Outputs       map[string]Output // The outputs created by this transaction.
	ChangeAddress string            // The address that should be given change from this transaction.
	Collateral    string
}

// Get the Inputs owned by an address.
func GetUTXOs(address string) ([]Input, error) {
	cmd := exec.Command("cardano-cli",
		"query",
		"utxo",
		"--address",
		address,
		"--out-file",
		"/dev/stdout",
		"--testnet-magic",
		testnetMagic)
	output, err := execCommand(cmd)

	var result []Input

	if err != nil {
		return result, err
	}

	var dev map[string]JSONInput
	json.Unmarshal([]byte(output), &dev)

	for key, input := range dev {
		// Skip the tester collateral!
		if key == SPCollateral || key == CPCollateral {
			continue
		}

		var newInput Input
		newInput.Key = key
		newInput.Value = make(map[string]int64)

		for policy, value := range input.Value {
			if policy == "lovelace" {
				num := value.(float64)
				newInput.Value["lovelace"] = int64(num)
			} else {
				assets := value.(map[string]interface{})
				for assetHex, amount := range assets {
					newInput.Value[fmt.Sprintf("%s.%s", policy, assetHex)] = int64(amount.(float64))
				}
			}
		}

		result = append(result, newInput)
	}

	return result, err
}

func FindOutput(inputs []Input, tx_hash string, index int) (Input, bool) {
	key := fmt.Sprintf("%s#%d", tx_hash, index)
	for _, input := range inputs {
		if input.Key == key {
			return input, true
		}
	}

	return Input{}, false
}

// func getSignaturesFromOracle(redeemer Redeemer) (oracleResp *oracle.RewardResponse, err error) {

// 	logWithError := "https://log.nunet.io/api/v1/logbin/01472e48-c430-46f1-8b8a-40c6d1e5cfd1/raw"
// 	logWithoutErrors := "https://log.nunet.io/api/v1/logbin/13d26b81-4324-402b-9f7f-cbd05afafb67/raw"
// 	// For successful job
// 	jobStatus := "finished without errors"
// 	logPath := logWithoutErrors
// 	jobDuration := int64(55)

// 	switch redeemer {
// 	case Refund:
// 		logPath = logWithError
// 		jobStatus = "finished with errors"
// 		jobDuration = 1
// 	case Distribute:
// 		logPath = logWithError
// 		jobStatus = "finished with errors"
// 		jobDuration = 45
// 	case Withdraw: // noop
// 	}

// 	oracleResp, err = oracle.Oracle.WithdrawTokenRequest(&oracle.RewardRequest{
// 		JobStatus:            jobStatus,
// 		JobDuration:          jobDuration,
// 		EstimatedJobDuration: 60,
// 		LogPath:              logPath,
// 		MetadataHash:         PreGenMetaDataHash,
// 		WithdrawHash:         PreGenWithdrawHash,
// 		RefundHash:           PreGenRefundHash,
// 		Distribute_50Hash:    PreGenDistribute50Hash,
// 		Distribute_75Hash:    PreGenDistribute75Hash,
// 	})

// 	if err != nil {
// 		return &oracle.RewardResponse{}, fmt.Errorf("Connection to oracle failed : %v", err)
// 	}

// 	return oracleResp, nil
// }

func BuildPaymentScriptAddress() (address string, err error) {
	cmd := exec.Command("cardano-cli",
		"address",
		"build",
		"--payment-script-file",
		"script.json",
		"--out-file",
		"/dev/stdout",
		"--testnet-magic",
		testnetMagic)

	output, err := execCommand(cmd)

	address = strings.Trim(output, "\n")
	return
}

// Take money from script
// func SpendFromScript(tx_hash string, index int, redeemer Redeemer) error {
// 	currentContract, err := BuildPaymentScriptAddress()
// 	if err != nil {
// 		return err
// 	}

// 	scriptOutputs, err := GetUTXOs(currentContract)
// 	if err != nil {
// 		return err
// 	}

// 	scriptInput, success := FindOutput(scriptOutputs, tx_hash, index)
// 	scriptInput.ScriptFile = "script.json"
// 	scriptInput.DatumFile = "datum.json"
// 	scriptInput.RedeemerFile = "redeemer.json"

// 	if !success {
// 		panic("Failed to find the script output")
// 	}

// 	resp, err := getSignaturesFromOracle(redeemer)
// 	if err != nil {
// 		panic("Failed to contact oracle")
// 	}

// 	WriteRedeemerFile("redeemer.json", resp, redeemer)

// 	var collateral string
// 	var account string
// 	var address string
// 	if redeemer == Refund {
// 		address = SPAddress
// 		account = SPAccount
// 		collateral = SPCollateral
// 	} else {
// 		address = CPAddress
// 		account = CPAccount
// 		collateral = CPCollateral
// 	}

// 	outputs, err := GetUTXOs(address)
// 	if err != nil {
// 		return err
// 	}

// 	outputs = append(outputs, scriptInput)

// 	transaction := Transaction{
// 		Inputs:        outputs,
// 		Outputs:       make(map[string]Output),
// 		ChangeAddress: address,
// 		Collateral:    collateral,
// 	}

// 	transaction.Outputs[address] = Output{
// 		To:    address,
// 		Value: make(map[string]int64),
// 	}

// 	transaction.Outputs[address].Value[mNTX] = scriptInput.Value[mNTX]
// 	transaction.Outputs[address].Value["lovelace"] = minOutputLovelace

// 	var txFees = int64(2200000)
// 	BuildTransaction(transaction, txFees)
// 	SignTransaction(account)
// 	hash, err := GetTransactionHash()
// 	if err != nil {
// 		return err
// 	}
// 	err = SubmitTransaction()
// 	if err != nil {
// 		return err
// 	}
// 	log.Printf("Transaction Hash %s", hash)
// 	return err
// }

func WaitForTransaction(tx_hash string, max_timeout_minutes int) error {

	log.Printf("Waiting for Transaction Confirmation %s", tx_hash)
	if max_timeout_minutes == 0 {
		return errors.New("Max timeout reached, transaction not observed")
	}

	cmd := exec.Command("cardano-cli",
		"query",
		"utxo",
		"--tx-in",
		fmt.Sprintf("%s#0", tx_hash),
		"--out-file",
		"/dev/stdout",
		"--testnet-magic",
		testnetMagic,
	)

	output, err := execCommand(cmd)
	if err != nil || string(output) == "{}" {
		time.Sleep(1 * time.Minute)
		return WaitForTransaction(tx_hash, max_timeout_minutes-1)
	}

	return nil
}

// Pay to the current escrow smart contract an amount in NTX
func PayToScript(ntx int64, spPubKey string, cpPubKey string) (string, error) {
	outputs, err := GetUTXOs(SPAddress)
	if err != nil {
		return "", err
	}

	currentContract, err := BuildPaymentScriptAddress()
	if err != nil {
		return "", err
	}

	transaction := Transaction{
		Inputs:        outputs,
		Outputs:       make(map[string]Output),
		ChangeAddress: SPAddress,
	}

	const datumFile = "datum.json"
	WriteDatumFile(datumFile, ntx, spPubKey, cpPubKey)

	transaction.Outputs[currentContract] = Output{
		To:        currentContract,
		Value:     make(map[string]int64),
		DatumFile: datumFile,
	}

	transaction.Outputs[currentContract].Value[mNTX] = ntx

	// Set the min ada to be held in the output
	transaction.Outputs[currentContract].Value["lovelace"] = minOutputLovelace

	var txFees = int64(0) // do estimation
	BuildTransaction(transaction, txFees)
	SignTransaction(SPAccount)
	hash, err := GetTransactionHash()
	if err != nil {
		return "", err
	}
	err = SubmitTransaction()
	return hash, err
}

func execCommand(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Command:", cmd.String()) // Print the command
		fmt.Println("Output:", stdout.String())
		fmt.Println("Error:", stderr.String())
		// if exitError, ok := err.(*exec.ExitError); ok {
		// 	if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
		// 		log.Fatal("Exit Code:", status.ExitStatus())
		// 	}
		// }
	}
	return stdout.String(), err
}

func SubmitTransaction() error {
	cmd := exec.Command("cardano-cli",
		"transaction",
		"submit",
		"--tx-file",
		"tx.signed",
		"--testnet-magic",
		testnetMagic,
	)
	_, err := execCommand(cmd)
	return err
}

var minOutputLovelace = int64(5500000)

// Helper to build a valid transaction
func BuildTransaction(tx Transaction, txFees int64) (err error) {
	EnsureProtocolParameters()
	fee := int64(200000)
	if txFees == 0 { // Estimate fees if not specified
		// We pass in a wide enough lovelace value so the
		// size is approximated correctly.
		BuildTransactionRaw(tx, 200000)
		fee, err = EstimateFee(tx)
		if err != nil {
			return err
		}
	} else {
		// Use specified fee
		fee = txFees
	}
	BalanceTransaction(&tx, fee)
	BuildTransactionRaw(tx, fee)
	return err
}

// Sign a transaction with the SPAddress.
func SignTransaction(account string) {
	cmd := exec.Command("cardano-cli",
		"transaction",
		"sign",
		"--tx-body-file",
		"tx.draft",
		"--signing-key-file",
		fmt.Sprintf("%s.sk", account),
		"--out-file",
		"tx.signed",
	)

	execCommand(cmd)
}

// Estimate fee of transaction
func EstimateFee(tx Transaction) (fee int64, err error) {

	witness_count := 1

	// Factor in script witnesses
	for _, input := range tx.Inputs {
		if input.ScriptFile != "" {
			witness_count += 1
		}
	}

	cmd := exec.Command("cardano-cli",
		"transaction",
		"calculate-min-fee",
		"--tx-body-file",
		"tx.draft",
		"--tx-in-count",
		fmt.Sprintf("%d", len(tx.Inputs)),
		"--tx-out-count",
		fmt.Sprintf("%d", len(tx.Outputs)),
		"--witness-count",
		fmt.Sprintf("%d", witness_count),
		"--protocol-params-file",
		"protocol.json",
		"--testnet-magic",
		testnetMagic,
	)

	output, err := execCommand(cmd)
	if err != nil {
		return 0, err
	}

	_, fee_err := fmt.Sscan(output, &fee)
	if fee_err != nil {
		log.Fatal(fee_err)
	}

	return fee, err
}

func BalanceTransaction(tx *Transaction, fee int64) {
	var change map[string]int64 = make(map[string]int64)

	// Seed change with inputs
	for _, input := range tx.Inputs {
		for token, amount := range input.Value {
			if change[token] == -1 {
				change[token] = amount
			} else {
				change[token] += amount
			}
		}
	}

	// Subtract outputs
	for _, output := range tx.Outputs {
		for token, amount := range output.Value {
			change[token] -= amount
		}
	}

	// Deduct fees
	change["lovelace"] -= fee

	// Add entry if it doesn't exist yet
	entry := tx.Outputs[tx.ChangeAddress]
	if entry.Value == nil {
		entry.To = tx.ChangeAddress
		entry.Value = make(map[string]int64)
		tx.Outputs[tx.ChangeAddress] = entry
	}

	// Apply to ChangeAddress output
	for token, amount := range change {
		if tx.Outputs[tx.ChangeAddress].Value[token] == -1 {
			tx.Outputs[tx.ChangeAddress].Value[token] = amount
		} else {
			tx.Outputs[tx.ChangeAddress].Value[token] += amount
		}
	}
}

// Ensures a valid protocol.json file exists and creates one from the connected chain otherwise.
func EnsureProtocolParameters() {
	cmd := exec.Command("cardano-cli",
		"query",
		"protocol-parameters",
		"--out-file",
		"protocol.json",
		"--testnet-magic",
		testnetMagic)

	execCommand(cmd)
}

// Get the transaction hash of the most recently built transaction.
func GetTransactionHash() (hash string, err error) {
	cmd := exec.Command("cardano-cli", "transaction", "txid", "--tx-file", "tx.signed")

	output, err := execCommand(cmd)

	hash = strings.Trim(output, "\n")
	return
}

func EstimatePlutusInteraction(tx Transaction) {
	args := make([]string, 0)

	args = append(args,
		"transaction",
		"build",
		"--calculate-plutus-script-cost",
		"script_cost",
		"--testnet-magic",
		testnetMagic,
		"--change-address",
		SPAddress,
		"--required-signer",
		"tester.sk",
	)

	for _, input := range tx.Inputs {
		args = append(args, "--tx-in")
		args = append(args, input.Key)

		if input.ScriptFile != "" {
			args = append(args, "--tx-in-script-file")
			args = append(args, input.ScriptFile)
		}

		// TODO(skylar): See if inline datum is present
		if input.DatumFile != "" {
			args = append(args, "--tx-in-inline-datum-present")
		}

		if input.RedeemerFile != "" {
			args = append(args, "--tx-in-redeemer-file")
			args = append(args, input.RedeemerFile)
		}
	}

	if tx.Collateral != "" {
		args = append(args, "--tx-in-collateral")
		args = append(args, tx.Collateral)
	}

	for _, output := range tx.Outputs {
		args = append(args, "--tx-out")
		args = append(args, fmt.Sprintf(`%s+%s`, output.To, ValueStr(output.Value)))

		if output.DatumFile != "" {
			args = append(args, "--tx-out-inline-datum-file")
			args = append(args, output.DatumFile)
		}
	}

	cmd := exec.Command("cardano-cli", args...)

	pipe, _ := cmd.StderrPipe()
	scanner := bufio.NewScanner(pipe)

	go func() {
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
	}()

	cmd.Start()
	cmd.Wait()
}

// Builds a transaction file with the given fee.
func BuildTransactionRaw(tx Transaction, fee int64) {
	args := make([]string, 0)

	args = append(args,
		"transaction",
		"build-raw",
		"--fee",
		fmt.Sprintf("%d", fee),
		"--protocol-params-file",
		"protocol.json",
		"--out-file",
		"tx.draft",
	)

	for _, input := range tx.Inputs {
		args = append(args, "--tx-in")
		args = append(args, input.Key)

		if input.ScriptFile != "" {
			args = append(args, "--tx-in-script-file")
			args = append(args, input.ScriptFile)
		}

		// NOTE: Inline datum is expect to be standard
		if input.DatumFile != "" {
			args = append(args, "--tx-in-inline-datum-present")
		}

		if input.RedeemerFile != "" {
			args = append(args, "--tx-in-redeemer-file")
			args = append(args, input.RedeemerFile)
			args = append(args, "--tx-in-execution-units")
			args = append(args, "(3000000000, 7000000)")
		}
	}

	if tx.Collateral != "" {
		args = append(args, "--tx-in-collateral")
		args = append(args, tx.Collateral)
	}

	for _, output := range tx.Outputs {
		args = append(args, "--tx-out")
		args = append(args, fmt.Sprintf(`%s+%s`, output.To, ValueStr(output.Value)))

		if output.DatumFile != "" {
			args = append(args, "--tx-out-inline-datum-file")
			args = append(args, output.DatumFile)
		}
	}

	cmd := exec.Command("cardano-cli", args...)

	pipe, _ := cmd.StderrPipe()
	scanner := bufio.NewScanner(pipe)

	go func() {
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
	}()

	cmd.Start()
	cmd.Wait()
}

// Create a cardano-cli compatible multi-asset vaule string from a map of asset to amount
func ValueStr(value map[string]int64) (str string) {
	builder := strings.Builder{}

	count := 0
	output_count := len(value)
	for token, amount := range value {
		if token == "lovelace" {
			builder.WriteString(fmt.Sprintf("%d", amount))
		} else {
			builder.WriteString(fmt.Sprintf("%d %s", amount, token))
		}

		if count < output_count-1 {
			builder.WriteString("+")
		}

		count += 1
	}

	return builder.String()
}

// func WriteRedeemerFile(path string, response *oracle.RewardResponse, redeemer Redeemer) {
// 	r, _ := regexp.Compile(`\\\"(.*?)\\\"`)
// 	// NOTE: The submatch or capture group is the second argument, the first is the whole matched expression
// 	action_capture := r.FindStringSubmatch(response.Action)[1]
// 	datum_capture := r.FindStringSubmatch(response.Datum)[1]

// 	var action_hex = hex.EncodeToString([]byte(action_capture))
// 	var datum_hex = hex.EncodeToString([]byte(datum_capture))

// 	if err := os.WriteFile(path, []byte(fmt.Sprintf(
// 		REDEEMER_FORMAT_STRING,
// 		redeemer,
// 		response.SignatureDatum,
// 		response.MessageHashDatum,
// 		datum_hex,
// 		response.SignatureAction,
// 		response.MessageHashAction,
// 		action_hex)), 0666); err != nil {
// 		log.Fatal(err)
// 	}
// }

func WriteDatumFile(path string, ntx int64, spPubKeyHash string, cpPubKeyHash string) {
	if err := os.WriteFile(path, []byte(fmt.Sprintf(DATUM_FORMAT_STRING, spPubKeyHash, cpPubKeyHash, ntx, PreGenMetaDataHash, PreGenWithdrawHash, PreGenRefundHash, PreGenDistribute75Hash, PreGenDistribute50Hash)), 0666); err != nil {
		log.Fatal(err)
	}
}
