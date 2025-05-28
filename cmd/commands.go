package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// For terminal display

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type ConfigCliParams struct {
	PersistentPeers string `json:"persistent_peers"`
	GenesisUrl      string `json:"genesisUrl"`
	ConfigTomlUrl   string `json:"configToml"`
	ChindId         string `json:"chindId"`
	MinStakeFund    int64  `json:"minStakeFund"`
	BootNodeRpc     string `json:"bootNodeRpc"`
}

var configCliParams ConfigCliParams

func runCmd(command string, args ...string) error {
	fmt.Printf("Running: %s %v\n", command, args)
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runCmdCaptureOutput(command string, args ...string) (string, error) {
	fmt.Printf("Running: %s %v\n", command, args)
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func initNodeCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "init-node",
		Short: "Initialize Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initNodeLogic(mynode)
		},
	}

	cmd.Flags().StringVar(&mynode, "mynode", "mrmintchainNode001", "Your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

// ✅ Extracted logic to reuse in auto-run
func initNodeLogic(mynode string) error {
	configCliParams = getConfigCliParams()

	validatorName := mynode
	mynode = "" + mynode

	genesisPath := mynode + "/config/genesis.json"

	if exists(genesisPath) {
		log.Info("⚠️  genesis.json already exists: " + genesisPath)
		if !yesNo("Delete and proceed?") {
			log.Error("Cancelled")
			return nil
		}
		if err := os.RemoveAll(mynode); err != nil {
			log.Error("failed to remove node folder: ")
			return err
		}
	}

	if output, err := runCmdCaptureOutput("ethermintd", "init", validatorName, "--chain-id", configCliParams.ChindId, "--home", mynode); err != nil {
		log.Errorf("init command failed: %s", output)
		return err
	}

	updateGenesis(mynode)
	updateConfigToml(mynode)

	fmt.Println("✅ Node initialized.")
	return nil
}

func addKeyCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "add-key",
		Short: "Add key to keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			return addKeyCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func addKeyCmdLogic(mynode string) error {
	permission := yesNo("Are you want to generate wallet ?")
	if !permission {
		log.Fatalf("Key Generation process stop.")
	}

	validatorName := mynode
	mynode = "" + mynode

	output, err := runCmdCaptureOutput("ethermintd", "keys", "add", validatorName, "--algo", "eth_secp256k1", "--keyring-backend", "test", "--home", mynode)
	if err != nil {
		log.Errorf("keys add command failed: %s\nOutput: %s", err, output)
		return err
	}

	log.Printf("Key generation output: %s\n", output)
	log.Print("🔑 Please copy your key output above. Press Enter to continue...")
	fmt.Scanln()
	return nil
}

func addGenesisAccountCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "add-genesis-account",
		Short: "Add validator account to genesis",
		RunE: func(cmd *cobra.Command, args []string) error {
			return addGenesisAccountLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func addGenesisAccountLogic(mynode string) error {

	validatorName := mynode
	mynode = "" + mynode

	getAddr := exec.Command("ethermintd", "keys", "show", validatorName, "-a", "--home", mynode, "--keyring-backend", "test")
	addrOut, err := getAddr.Output()
	if err != nil {
		return err
	}
	ethm1Address := strings.TrimSpace(string(addrOut))
	ethAddress, err := Bech32ToEthAddress(ethm1Address)
	if err != nil {
		log.Fatalf("Invalid bech32 address: %v", err)
	}

	log.Info("Its your validator wallet : ")
	log.Infof("Default ethm1 format : %s", ethm1Address)
	log.Infof("Converted into Ethereum(0x) format : %s", ethAddress)

	qrterminal.GenerateHalfBlock(ethAddress, qrterminal.L, os.Stdout)

	log.Infof("📲 QR Code (scan it securely): Please send %d MNT coin to your validator wallet for validator staking.", configCliParams.MinStakeFund)

	getConfirmationForPayment("Have you deposited MNT?", ethm1Address)
	return nil
}

func getBalanceCmdLogic(walletEthmAddress string) (bool, int64) {
	bootRpc := configCliParams.BootNodeRpc
	if bootRpc == "" {
		bootRpc = getEnvOrFail("BOOT_NODE_RPC")
	}
	if bootRpc == "" {
		log.Errorf("Boot node rpc not provided")
		return false, 0
	}
	output, err := runCmdCaptureOutput("ethermintd", "query", "bank", "balances", walletEthmAddress, "--node", bootRpc)
	if err != nil {
		log.Errorf("Get balance command failed: %s\nOutput: %s", err, output)
		return false, 0
	}
	var cResp CmdOutput

	err = yaml.Unmarshal([]byte(output), &cResp)
	if err != nil {
		log.Errorf("Get balance command failed: %s", err)
		return false, 0
	}

	if len(cResp.Balances) == 0 {
		log.Error("The balances array is indeed empty, as expected. Please deposit fund then proceed")
		return false, 0
	} else {

		bigAmount := new(big.Int)
		bigAmount, ok := bigAmount.SetString(cResp.Balances[0].Amount, 10)
		if !ok {
			log.Error("Invalid number")
			return false, 0
		}

		// Divide by Wei (as big.Int)
		wei := big.NewInt(1e18)
		exactBalance := new(big.Int).Div(bigAmount, wei)

		// This block would only execute if balances was not empty.
		log.Infof("💸 The balances is : %d %s", exactBalance, cResp.Balances[0].Denom)
		log.Infof("💸 The Exact balances is : %s %s", cResp.Balances[0].Amount, cResp.Balances[0].Denom)

		return true, exactBalance.Int64()

	}
}

func portsAndEnvGenerationCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "port-set",
		Short: "To start node, ports and env generation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return portsAndEnvGenerationLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func portsAndEnvGenerationLogic(mynode string) error {
	fmt.Print("\n Please enter port - \n")
	portsArray := []string{}

	p2p := getPortInputAndCheck("P2P_PORT", "26666", portsArray)
	portsArray = append(portsArray, p2p)
	log.Infof("✅ p2p-laddr: %s", p2p)

	rpc := getPortInputAndCheck("RPC_PORT", "26667", portsArray)
	portsArray = append(portsArray, rpc)
	log.Infof("✅ rpc-laddr: %s", rpc)

	grpc := getPortInputAndCheck("GRPC_PORT", "9092", portsArray)
	portsArray = append(portsArray, grpc)
	log.Infof("✅ grpc-address: %s", grpc)

	grpcWeb := getPortInputAndCheck("GRPC_WEB_PORT", "9093", portsArray)
	portsArray = append(portsArray, grpcWeb)
	log.Infof("✅ grpc-web-address: %s", grpcWeb)

	jsonRpc := getPortInputAndCheck("JSON_RPC_PORT", "8547", portsArray)
	log.Infof("✅ json-rpc-address: %s", jsonRpc)

	// Construct .env content

	envContent := fmt.Sprintf(`
		P2P_PORT=%s
		RPC_PORT=%s
		GRPC_PORT=%s
		GRPC_WEB_PORT=%s
		JSON_RPC_PORT=%s
		PERSISTENT_PEERS=%s
		BOOT_NODE_RPC=%s`,
		p2p,
		rpc,
		grpc,
		grpcWeb,
		jsonRpc,
		configCliParams.PersistentPeers,
		configCliParams.BootNodeRpc,
	)

	// Path to write .env
	envPath := filepath.Join(mynode, ".env")

	// Ensure the directory exists
	if err := os.MkdirAll(mynode, os.ModePerm); err != nil {
		log.Errorf("❌ Failed to create node directory: %v\n", err)
		os.Exit(1)
	}

	// Write to .env
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		log.Infof("❌ Failed to write .env file: %v\n", err)
		os.Exit(1)
	}

	log.Infof("✅ .env file generated at %s\n", envPath)
	return err
}

func startNodeCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "start-node",
		Short: "Start the Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startNodeCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func startNodeCmdLogic(mynode string) error {
	// Load the .env file
	mynode = "" + mynode

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}

	// Read from environment variables (set during auto-setup)
	p2pPort := getEnvOrFail("P2P_PORT")
	rpcPort := getEnvOrFail("RPC_PORT")
	grpcPort := getEnvOrFail("GRPC_PORT")
	grpcWebPort := getEnvOrFail("GRPC_WEB_PORT")
	jsonRpcPort := getEnvOrFail("JSON_RPC_PORT")
	PersistentPeers := getEnvOrFail("PERSISTENT_PEERS")

	p2pLaddr := "tcp://0.0.0.0:" + p2pPort
	rpcLaddr := "tcp://0.0.0.0:" + rpcPort
	grpcAddress := "0.0.0.0:" + grpcPort
	grpcWebAddress := "0.0.0.0:" + grpcWebPort
	jsonRpcAddress := "0.0.0.0:" + jsonRpcPort

	log.Info("✅ Using Ports from ENV:")
	log.Infof("  - p2p-laddr: %s", p2pLaddr)
	log.Infof("  - rpc-laddr: %s", rpcLaddr)
	log.Infof("  - grpc-address: %s", grpcAddress)
	log.Infof("  - grpc-web-address: %s", grpcWebAddress)
	log.Infof("  - json-rpc-address: %s", jsonRpcAddress)
	log.Infof("  - persistent-peers: %s \n", PersistentPeers)

	imageName := os.Getenv("IMAGE_NAME")
	if imageName == "" {
		log.Fatal("❌ IMAGE_NAME is not set in environment")
	}

	// Run the command with ports from ENV
	err = runCmd("docker", "run", "-d", "-it", "--name", mynode, "-v", fmt.Sprintf("./%s:/app/%s", mynode, mynode),
		"-p", p2pPort+":"+p2pPort, // P2P port
		"-p", rpcPort+":"+rpcPort, // RPC port
		"-p", grpcPort+":"+grpcPort, // Ethereum JSON-RPC
		"-p", grpcWebPort+":"+grpcWebPort, // gRPC
		"-p", jsonRpcPort+":"+jsonRpcPort, // gRPC-Web
		imageName, "ethermintd", "start",
		"--home", mynode,
		"--p2p.laddr", p2pLaddr,
		"--rpc.laddr", rpcLaddr,
		"--grpc.address", grpcAddress,
		"--grpc-web.address", grpcWebAddress,
		"--json-rpc.address", jsonRpcAddress,
		"--p2p.persistent_peers", PersistentPeers)
	if err != nil {
		log.Errorf("❌ node start command failed: %s", err)
		return err
	}

	log.Info("🚀 Node started successfully!")
	log.Infof("🚀 Now you can check logs, stop, start, remove container with following commands: ")
	log.Infof("===> docker logs %s", mynode)
	log.Infof("===> docker stop %s", mynode)
	log.Infof("===> docker start %s", mynode)
	log.Infof("===> docker rm %s", mynode)
	return nil
}

func stopNodeCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "stop-node",
		Short: "Stop the Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmd("docker", "stop", mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func restartNodeCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "restart-node",
		Short: "Re-start the Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmd("docker", "start", mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func getValidatorBalanceCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "validator-balance",
		Short: "Validator balance from Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getValidatorBalanceCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func getValidatorBalanceCmdLogic(mynode string) error {
	//Load env
	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}

	fmt.Println()
	getAddr := exec.Command("ethermintd", "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	addrOut, err := getAddr.Output()
	if err != nil {
		return err
	}
	ethm1Address := strings.TrimSpace(string(addrOut))
	getBalanceCmdLogic(ethm1Address)
	return err
}

func stakeFundCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "stake",
		Short: "Stake the Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkBlockBeforeStake(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

type BlockResponse struct {
	Block struct {
		Header struct {
			Height string `json:"height"`
		} `json:"header"`
	} `json:"block"`
}

func checkBlockBeforeStake(mynode string) error {
	// Load the .env file

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}
	rpcPort := getEnvOrFail("RPC_PORT")
	bootRpc := getEnvOrFail("BOOT_NODE_RPC")

	outputLocal, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, "ethermintd", "query", "block", "--node", "http://localhost:"+rpcPort)
	if err != nil {
		log.Errorf("Query block command error : %s \n", outputLocal)
		return err
	}

	outputBootNode, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, "ethermintd", "query", "block", "--node", bootRpc)
	if err != nil {
		log.Errorf("Query block command error : %s \n", outputBootNode)
		return err
	}

	var res BlockResponse
	err = json.Unmarshal([]byte(outputBootNode), &res)
	if err != nil {
		panic(err)
	}

	bootBlockInt := new(big.Int)
	bootBlockInt, ok := bootBlockInt.SetString(res.Block.Header.Height, 10)
	if !ok {
		log.Error("Invalid number")
	}

	var resLocal BlockResponse
	err = json.Unmarshal([]byte(outputLocal), &resLocal)
	if err != nil {
		panic(err)
	}

	localBlockInt := new(big.Int)
	localBlockInt, ok = localBlockInt.SetString(resLocal.Block.Header.Height, 10)
	if !ok {
		log.Error("Invalid number")
	}

	log.Infof("Boot node latest block height: %s", bootBlockInt)
	log.Infof("Your node latest block height: %s", localBlockInt)

	bootBlockInt = new(big.Int).Sub(bootBlockInt, big.NewInt(5))

	if localBlockInt.Int64() < bootBlockInt.Int64() {
		log.Error("Please wait for complete syncing then stake fund for validator")
		return err
	}
	log.Info("\xE2\x9C\x94 The node is properly synced with the bootnode!")

	return stakeFundCmdLogic(mynode)
}

type DepositParams struct {
	MinDeposit []struct {
		Amount string `yaml:"amount"`
		Denom  string `yaml:"denom"`
	} `yaml:"min_deposit"`
}

func stakeFundCmdLogic(mynode string) error {
	rpcPort := getEnvOrFail("RPC_PORT")
	// bootRpc := getEnvOrFail("BOOT_NODE_RPC")
	fmt.Println()
	output, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, "ethermintd", "query", "gov", "param", "deposit", "--node", "tcp://localhost:"+rpcPort)
	if err != nil {
		log.Fatalf("failed to get deposit params: %v", err)
	}
	var cResp DepositParams

	err = yaml.Unmarshal([]byte(output), &cResp)
	if err != nil {
		log.Errorf("Get balance command failed: %s", err)
	}

	log.Infof("Minimum Deposit: %s%s", cResp.MinDeposit[0].Amount, cResp.MinDeposit[0].Denom)
	fmt.Println()

	getAddr := exec.Command("ethermintd", "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	addrOut, err := getAddr.Output()
	if err != nil {
		return err
	}
	ethm1Address := strings.TrimSpace(string(addrOut))
	_, balance := getBalanceCmdLogic(ethm1Address)
	ethAddress, _ := Bech32ToEthAddress(ethm1Address)
	log.Printf("balance %d Of wallet : %s", balance, ethAddress)

	pubkey, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, "ethermintd", "tendermint", "show-validator", "--home", mynode)
	if err != nil {
		log.Fatalf("Failed to get pubkey: %v", err)
	}
	pubkey = strings.TrimSpace(pubkey)
	if !yesNo("Are you want to proceed now for staking?") {
		log.Info("Staking process cancelled!")
		return err
	}

	commissionRate := getStakingInputs("Please enter commission rate", "0.10")
	log.Infof("✅ commission-rate: %s", commissionRate)

	commissionMaxRate := getStakingInputs("Please enter commission max rate", "0.20")
	log.Infof("✅ commission-max-rate: %s", commissionMaxRate)

	commissionMaxChangeRate := getStakingInputs("Please enter commission max change rate", "0.01")
	log.Infof("✅ commission-max-change-rate: %s", commissionMaxChangeRate)

	fmt.Println()
	log.Print("🔑 Your staking process almost done. Press Enter to continue...")
	fmt.Scanln()

	output, err = runCmdCaptureOutput("docker", "exec", "-i", mynode,
		"ethermintd",
		"tx", "staking", "create-validator",
		"--amount", cResp.MinDeposit[0].Amount+""+cResp.MinDeposit[0].Denom,
		"--pubkey", pubkey,
		"--home", mynode,
		"--moniker", mynode,
		// "--chain-id","os_9000-1",
		"--commission-rate", commissionRate,
		"--commission-max-rate", commissionMaxRate,
		"--commission-max-change-rate", commissionMaxChangeRate,
		"--min-self-delegation=1",
		"--from", mynode,
		"--keyring-backend=test",
		"--home", mynode, // double-check this format
		"--node", "tcp://localhost:"+rpcPort,
		"--gas-prices", "7aphoton",
		"--gas", "auto",
		"--gas-adjustment", "1.1",
		"--yes",
	)
	if err != nil {
		log.Errorf("Stake command failed : %s", output)
		return err
	}
	log.Infof("Stake Output : %s", output)
	fmt.Println()
	log.Infof("You can copy staking tx hash and check details on explorer or use hash detail commands")
	return err
}

func getValidatorStatusCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "validator-info",
		Short: "Validator info for Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getValidatorStatusCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

type ValidatorDevKey []struct {
	Address string `yaml:"address"`
}
type ValidatorDevInfo struct {
	Status string `yaml:"status"`
}

func getValidatorStatusCmdLogic(mynode string) error {

	// Load the .env file

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}
	rpcPort := getEnvOrFail("RPC_PORT")
	// bootRpc := getEnvOrFail("BOOT_NODE_RPC")

	getAddr := exec.Command("ethermintd", "keys", "show", mynode, "--bech", "val", "--home", mynode, "--keyring-backend", "test")
	output, err := getAddr.Output()
	if err != nil {
		log.Errorf("Key show command failed : %s", string(output))
	}
	var cResp ValidatorDevKey

	err = yaml.Unmarshal(output, &cResp)
	if err != nil {
		log.Errorf("Get balance command failed: %s", err)
	}

	outputInfo, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, "ethermintd", "query", "staking", "validator", cResp[0].Address, "--node", "http://localhost:"+rpcPort, "--output", "json")
	if err != nil {
		log.Fatalf("Failed to get validator info : %s", outputInfo)
		return err
	}

	var cRespp ValidatorDevInfo
	err = yaml.Unmarshal([]byte(outputInfo), &cRespp)
	if err != nil {
		log.Errorf("Get balance command failed: %s", err)
	}
	log.Infof("Validator details in JSON : %s", outputInfo)
	fmt.Println()
	if cRespp.Status == "BOND_STATUS_BONDED" {
		log.Info("\xE2\x9C\x94 Validator is active!")
	}
	if cRespp.Status == "BOND_STATUS_UNBONDED" {
		log.Info("Validator is de-active!")
	}
	return err
}
