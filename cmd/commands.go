package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	ChaindId        string `json:"chindId"`
	MinStakeFund    int64  `json:"minStakeFund"`
	BootNodeRpc     string `json:"bootNodeRpc"`
}

var Mrmintd = "./ethermintd"

var configCliParams ConfigCliParams

type ParamChange struct {
	Subspace string          `json:"subspace"`
	Key      string          `json:"key"`
	Value    json.RawMessage `json:"value"`
}

type ParameterChangeProposalContent struct {
	Type        string        `json:"@type"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Changes     []ParamChange `json:"changes"`
}

type TextProposalContent struct {
	Type        string `json:"@type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type MsgExecLegacyContentWrapper struct {
	Type      string          `json:"@type"`   // "/cosmos.gov.v1.MsgExecLegacyContent"
	Content   json.RawMessage `json:"content"` // This holds the marshaled ParameterChangeProposalContent or TextProposalContent
	Authority string          `json:"authority"`
}

type ProposalFile struct {
	Messages []json.RawMessage `json:"messages"`
	Deposit  string            `json:"deposit"`
	Proposer string            `json:"proposer,omitempty"` // Optional proposer address
	Metadata string            `json:"metadata,omitempty"` // Optional metadata
}

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

	if output, err := runCmdCaptureOutput(Mrmintd, "init", validatorName, "--chain-id", configCliParams.ChaindId, "--home", mynode); err != nil {
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

	output, err := runCmdCaptureOutput(Mrmintd, "keys", "add", validatorName, "--algo", "eth_secp256k1", "--keyring-backend", "test", "--home", mynode)
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
	configCliParams = getConfigCliParams()

	validatorName := mynode
	mynode = "" + mynode

	getAddr := exec.Command(Mrmintd, "keys", "show", validatorName, "-a", "--home", mynode, "--keyring-backend", "test")
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
	configCliParams = getConfigCliParams()
	bootRpc := configCliParams.BootNodeRpc
	if bootRpc == "" {
		bootRpc = getEnvOrFail("BOOT_NODE_RPC")
	}
	if bootRpc == "" {
		log.Errorf("Boot node rpc not provided")
		return false, 0
	}
	output, err := runCmdCaptureOutput(Mrmintd, "query", "bank", "balances", walletEthmAddress, "--node", bootRpc)
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
	configCliParams = getConfigCliParams()

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
	mynode = "" + mynode // Ensure mynode path is correctly formatted

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}

	err = godotenv.Load(filepath.Join(".env"))
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

	imageName := getEnvOrFail("IMAGE_NAME")
	if imageName == "" {
		log.Fatal("❌ IMAGE_NAME is not set in environment")
	}
	log.Infof("Docker image found : %s in %s File", imageName, filepath.Join(".env"))

	// --- FIX IS HERE ---
	// Get the absolute path of the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Errorf("❌ Failed to get current working directory: %v", err)
		return err
	}

	// Construct the absolute path for the volume mount
	hostNodePath := filepath.Join(cwd, mynode)
	containerNodePath := filepath.Join("/app", mynode) // Assuming /app is where you want to mount inside Docker

	// Run the command with ports from ENV and the absolute path for volume mount
	err = runCmd("docker", "run", "-d", "-it", "--name", mynode,
		"-v", fmt.Sprintf("%s:%s", hostNodePath, containerNodePath), // Use the absolute paths here
		"-p", p2pPort+":"+p2pPort, // P2P port
		"-p", rpcPort+":"+rpcPort, // RPC port
		"-p", grpcPort+":"+grpcPort, // Ethereum JSON-RPC
		"-p", grpcWebPort+":"+grpcWebPort, // gRPC
		"-p", jsonRpcPort+":"+jsonRpcPort, // gRPC-Web
		imageName, Mrmintd, "start",
		"--home", mynode, // This refers to the path *inside* the container
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
	getAddr := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
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

	outputLocal, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, Mrmintd, "query", "block", "--node", "http://localhost:"+rpcPort)
	if err != nil {
		log.Errorf("Query block command error : %s \n", outputLocal)
		return err
	}

	outputBootNode, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, Mrmintd, "query", "block", "--node", bootRpc)
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
	output, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, Mrmintd, "query", "gov", "param", "deposit", "--node", "tcp://localhost:"+rpcPort)
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

	getAddr := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	addrOut, err := getAddr.Output()
	if err != nil {
		return err
	}
	ethm1Address := strings.TrimSpace(string(addrOut))
	_, balance := getBalanceCmdLogic(ethm1Address)
	ethAddress, _ := Bech32ToEthAddress(ethm1Address)
	log.Printf("balance %d Of wallet : %s", balance, ethAddress)

	pubkey, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, Mrmintd, "tendermint", "show-validator", "--home", mynode)
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
		Mrmintd,
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

func unjailCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "unjail",
		Short: "Unjail a jailed validator",
		Long: `Sends an unjail transaction to bring a jailed validator back online.
The validator must have sufficient funds to cover the transaction fees.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return unjailCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name for the jailed validator account)")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func unjailCmdLogic(mynode string) error {
	configCliParams = getConfigCliParams()

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT")

	getDelegatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	delegatorAddrOut, err := getDelegatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get delegator address for '%s': %s\nOutput: %s", mynode, err, string(delegatorAddrOut))
		return err
	}
	delegatorAddress := strings.TrimSpace(string(delegatorAddrOut))

	log.Infof("Attempting to unjail validator using key '%s' (delegator address: %s)", mynode, delegatorAddress)
	log.Infof("Sending unjail transaction to local node RPC: tcp://localhost:%s", rpcPort)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "slashing", "unjail",
		"--from", delegatorAddress,
		"--home", mynode, // Home path for keyring and node data
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId, // Use the chain ID from your config
		"--gas", "auto", // Automatically estimate gas required
		"--gas-adjustment", "1.1", // Add a buffer to gas estimate
		"--node", "tcp://localhost:"+rpcPort, // Target your local node's RPC
		"--yes", // Automatically confirm the transaction
	)

	if err != nil {
		log.Errorf("❌ Failed to unjail validator '%s': %s\nOutput: %s", mynode, err, output)
		log.Warnf("Please ensure your validator is actually jailed and has sufficient funds for transaction fees.")
		return err
	}

	log.Infof("✅ Validator '%s' unjail transaction sent successfully! Transaction output:\n%s", mynode, output)
	log.Info("Please monitor the chain and verify your validator's status using 'mrmintchain validator-info --mynode %s' after a few blocks.", mynode)

	return nil
}

type ValidatorDevKey []struct {
	Address string `yaml:"address"`
}

type ValidatorDevInfo struct {
	Status string `yaml:"status"`
}

func getValidatorStatusCmdLogic(mynode string) error {

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}
	rpcPort := getEnvOrFail("RPC_PORT")

	getAddr := exec.Command(Mrmintd, "keys", "show", mynode, "--bech", "val", "--home", mynode, "--keyring-backend", "test")
	output, err := getAddr.Output()
	if err != nil {
		log.Errorf("Key show command failed : %s", string(output))
	}
	var cResp ValidatorDevKey

	err = yaml.Unmarshal(output, &cResp)
	if err != nil {
		log.Errorf("Get balance command failed: %s", err)
	}

	outputInfo, err := runCmdCaptureOutput("docker", "exec", "-i", mynode, Mrmintd, "query", "staking", "validator", cResp[0].Address, "--node", "http://localhost:"+rpcPort, "--output", "json")
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

func setWithdrawAddress() *cobra.Command {
	var mynode string
	var address string

	cmd := &cobra.Command{
		Use:   "withdraw-address",
		Short: "Set withdraw address to withdraw your fund",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setWithdrawAddressLogic(mynode, address)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")

	cmd.Flags().StringVar(&address, "address", "", "Please enter your withdraw wallet address")
	cmd.MarkFlagRequired("address")

	return cmd
}

func setWithdrawAddressLogic(mynode string, address string) error {
	configCliParams = getConfigCliParams()

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT") // Get RPC port from loaded .env

	getAddr := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	addrOut, err := getAddr.Output()
	if err != nil {
		log.Errorf("Failed to get validator address for '%s': %s\nOutput: %s", mynode, err, string(addrOut))
		return err
	}
	validatorDelegatorAddress := strings.TrimSpace(string(addrOut))

	log.Infof("Attempting to set withdraw address for validator '%s' (delegator address: %s) to '%s'", mynode, validatorDelegatorAddress, address)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "distribution", "set-withdraw-addr", address,
		"--from", mynode,
		"--home", mynode,
		"--chain-id", configCliParams.ChaindId,
		"--gas-prices", "7aphoton",
		"--keyring-backend", "test",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort,
		"--yes",
	)

	if err != nil {
		log.Errorf("❌ Failed to set withdraw address for '%s': %s\nOutput: %s", mynode, err, output)
		return err
	}

	log.Infof("✅ Withdraw address set successfully! Transaction output:\n%s", output)
	log.Info("Please monitor the chain to confirm the transaction.")
	return nil
}

func delegateSelfStakeCmd() *cobra.Command {
	var mynode string
	var amount string

	cmd := &cobra.Command{
		Use:   "self-delegate",
		Short: "Delegate more tokens to your validator (increase self-delegation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return delegateSelfStakeLogic(mynode, amount)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name for your validator account)")
	cmd.MarkFlagRequired("mynode")

	cmd.Flags().StringVar(&amount, "amount", "", "Amount of tokens to delegate (e.g., 1000000000000000000aphoton)")
	cmd.MarkFlagRequired("amount")

	return cmd
}

func delegateSelfStakeLogic(mynode string, amount string) error {
	configCliParams = getConfigCliParams() // Ensure config is loaded

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load .env: %v", err)
	}
	rpcPort := getEnvOrFail("RPC_PORT")

	getDelegatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	delegatorAddrOut, err := getDelegatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get delegator address: %s\nOutput: %s", err, string(delegatorAddrOut))
		return err
	}
	validatorDelegatorAddress := strings.TrimSpace(string(delegatorAddrOut))

	getValidatorOperatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "--bech", "val", "--home", mynode, "--keyring-backend", "test")
	validatorOperatorAddrOut, err := getValidatorOperatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get validator operator address: %s\nOutput: %s", err, string(validatorOperatorAddrOut))
		return err
	}

	var keyInfo []struct {
		Address string `yaml:"address"`
	}

	err = yaml.Unmarshal(validatorOperatorAddrOut, &keyInfo)
	if err != nil {
		log.Errorf("Failed to parse validator operator address output: %s\nOutput: %s", err, string(validatorOperatorAddrOut))
		return err
	}

	if len(keyInfo) == 0 || keyInfo[0].Address == "" {
		log.Errorf("Could not find validator operator address in output: %s", string(validatorOperatorAddrOut))
		return fmt.Errorf("validator operator address not found")
	}

	validatorOperatorAddress := keyInfo[0].Address

	validatorStatusOutput, err := runCmdCaptureOutput(Mrmintd, "query", "staking", "validator", validatorOperatorAddress, "--node", "tcp://localhost:"+rpcPort, "--output", "json")
	if err != nil {

		if strings.Contains(validatorStatusOutput, "not found") || strings.Contains(validatorStatusOutput, "no such validator") {
			log.Errorf("❌ Validator '%s' not found on chain. Please create your validator first using the 'stake' command.", mynode)
			return fmt.Errorf("validator not found on chain")
		} else {
			log.Errorf("❌ Failed to query validator status: %s\nOutput: %s", err, validatorStatusOutput)
			return err
		}
	}

	var validatorInfo struct {
		OperatorAddress string `json:"operator_address"`
		Status          string `json:"status"`
	}
	err = json.Unmarshal([]byte(validatorStatusOutput), &validatorInfo)
	if err != nil {
		log.Errorf("Failed to parse validator status JSON: %s\nOutput: %s", err, validatorStatusOutput)
		return err
	}

	if validatorInfo.Status != "BOND_STATUS_BONDED" && validatorInfo.Status != "BOND_STATUS_UNBONDING" {
		log.Warnf("⚠️ Validator '%s' is not in a bonded or unbonding state (%s). Proceeding but verify this is intended.", mynode, validatorInfo.Status)
	}

	log.Infof("Attempting to self-delegate '%s' from '%s' to validator '%s'", amount, validatorDelegatorAddress, validatorOperatorAddress)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "staking", "delegate", validatorOperatorAddress, amount,
		"--from", validatorDelegatorAddress,
		"--home", mynode,
		"--keyring-backend", "test", // Match the backend
		"--chain-id", configCliParams.ChaindId,
		"--gas-prices", "7aphoton",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort,
		"--yes",
	)

	if err != nil {
		log.Errorf("❌ Failed to self-delegate tokens: %s\nOutput: %s", err, output)
		return err
	}

	log.Infof("✅ Tokens self-delegated successfully! Transaction output:\n%s", output)
	return nil
}

func unstakeCmd() *cobra.Command {
	var mynode string
	var amount string

	cmd := &cobra.Command{
		Use:   "unstake",
		Short: "Unstake (undelegate) tokens from your validator",
		Long: `Initiates the unbonding process for a specified amount of tokens from your validator.
The tokens will be locked for the unbonding period (e.g., 21 days) before becoming liquid again.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return unstakeCmdLogic(mynode, amount)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name of the validator)")
	cmd.MarkFlagRequired("mynode")

	cmd.Flags().StringVar(&amount, "amount", "", "Amount of tokens to unstake (e.g., 50aphoton)")
	cmd.MarkFlagRequired("amount")

	return cmd
}

func unstakeCmdLogic(mynode string, amount string) error {
	configCliParams = getConfigCliParams()

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT")

	getDelegatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	delegatorAddrOut, err := getDelegatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get delegator address for '%s': %s\nOutput: %s", mynode, err, string(delegatorAddrOut))
		return err
	}

	getValidatorOperatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "--bech", "val", "--home", mynode, "--keyring-backend", "test")
	validatorOperatorAddrOut, err := getValidatorOperatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get validator operator address for '%s': %s\nOutput: %s", mynode, err, string(validatorOperatorAddrOut))
		return err
	}

	var keyInfo []struct {
		Address string `yaml:"address"`
	}

	err = yaml.Unmarshal(validatorOperatorAddrOut, &keyInfo)
	if err != nil {
		log.Errorf("Failed to parse validator operator address output for '%s': %s\nOutput: %s", mynode, err, string(validatorOperatorAddrOut))
		return err
	}

	if len(keyInfo) == 0 || keyInfo[0].Address == "" {
		log.Errorf("Could not find validator operator address in output for '%s': %s", mynode, string(validatorOperatorAddrOut))
		return fmt.Errorf("validator operator address not found")
	}

	validatorOperatorAddress := keyInfo[0].Address // This is the clean ethmvaloper1... address

	log.Infof("Attempting to unstake '%s' from validator '%s' (%s)", amount, mynode, validatorOperatorAddress)
	log.Infof("Sending undelegation transaction to local node RPC: tcp://localhost:%s", rpcPort)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "staking", "unbond", validatorOperatorAddress, amount,
		"--from", mynode, // Use the key name for --from flag
		"--home", mynode, // Pass --home for keyring access
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId,
		"--gas-prices", "7aphoton",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort, // Target your local node's RPC
		"--yes", // Automatically confirm transaction
	)

	if err != nil {
		log.Errorf("❌ Failed to unstake tokens from '%s': %s\nOutput: %s", mynode, err, output)
		return err
	}

	log.Infof("✅ Unstake (undelegate) transaction sent successfully for '%s'! Transaction output:\n%s", mynode, output)
	log.Info("Tokens will be liquid after the unbonding period (typically 21 days). Please monitor your balance.")

	return nil
}

func withdrawRewardsCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "withdraw-rewards",
		Short: "Withdraw all accumulated staking rewards and validator commission",
		Long: `Sends a transaction to withdraw all accumulated staking rewards from your self-delegation
and any commission earned as a validator to your primary wallet address (which is also the
delegator address in this context).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withdrawRewardsCmdLogic(mynode)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name of the validator)")
	cmd.MarkFlagRequired("mynode")

	return cmd
}

func withdrawRewardsCmdLogic(mynode string) error {
	configCliParams = getConfigCliParams()

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT") // Get RPC port from loaded .env

	log.Infof("Attempting to withdraw all rewards for validator '%s'", mynode)
	log.Infof("Sending withdraw transaction to local node RPC: tcp://localhost:%s", rpcPort)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "distribution", "withdraw-all-rewards",
		"--from", mynode,
		"--home", mynode,
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId,
		"--gas", "auto",
		"--gas-prices", "7aphoton",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort,
		"--yes",
	)

	if err != nil {
		log.Errorf("❌ Failed to withdraw rewards for '%s': %s\nOutput: %s", mynode, err, output)
		log.Warnf("Please ensure your node is running and synced, and you have accumulated rewards to withdraw.")
		return err
	}

	log.Infof("✅ Withdraw rewards transaction sent successfully for '%s'! Transaction output:\n%s", mynode, output)
	log.Info("Please check your account balance to confirm the rewards have been received.")

	return nil
}

func editCommissionCmd() *cobra.Command {
	var mynode string
	var commissionRate string // Use string to pass directly to ethermintd

	cmd := &cobra.Command{
		Use:   "edit-commission",
		Short: "Edit your validator's commission rate",
		Long: `Sends a transaction to update your validator's commission rate.
Note: The new rate must adhere to the 'max-rate' and 'max-change-rate' defined
during your validator's creation. Changes are typically limited to once per 24 hours.
Example: --commission-rate "0.10" for 10% commission.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return editCommissionCmdLogic(mynode, commissionRate)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name of the validator)")
	cmd.MarkFlagRequired("mynode")

	cmd.Flags().StringVar(&commissionRate, "commission-rate", "", "New commission rate (e.g., \"0.10\" for 10%)")
	cmd.MarkFlagRequired("commission-rate")

	return cmd
}

func editCommissionCmdLogic(mynode string, commissionRate string) error {
	configCliParams = getConfigCliParams() // Ensure config is loaded

	// Load node-specific .env for RPC port
	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	// Load global .env for consistency and general configs (like chain-id if dynamic)
	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT") // Get RPC port from loaded .env

	// Input validation for commission rate
	rateFloat, err := strconv.ParseFloat(commissionRate, 64)
	if err != nil {
		log.Errorf("❌ Invalid commission rate format: %s. Must be a decimal (e.g., 0.10).", commissionRate)
		return fmt.Errorf("invalid commission rate")
	}
	if rateFloat < 0 || rateFloat > 1 {
		log.Errorf("❌ Commission rate must be between 0 and 1 (e.g., 0.05 for 5%%, 0.10 for 10%%). Got: %s", commissionRate)
		return fmt.Errorf("commission rate out of valid range")
	}

	// Get the delegator's address (ethm1...) -- This is the --from address for the transaction
	// This command uses the node's home directory for keyring access.
	getDelegatorAddrCmd := exec.Command(Mrmintd, "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
	delegatorAddrOut, err := getDelegatorAddrCmd.Output()
	if err != nil {
		log.Errorf("Failed to get delegator address for '%s': %s\nOutput: %s", mynode, err, string(delegatorAddrOut))
		return err
	}
	delegatorAddress := strings.TrimSpace(string(delegatorAddrOut))

	log.Infof("Attempting to set commission rate for validator '%s' (%s) to '%s'", mynode, delegatorAddress, commissionRate)
	log.Infof("Sending edit-validator transaction to local node RPC: tcp://localhost:%s", rpcPort)

	// Construct and run the edit-validator command
	// `ethermintd tx staking edit-validator --commission-rate [new-rate] --from [key-name] ...`
	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "staking", "edit-validator",
		"--commission-rate", commissionRate, // Pass the new rate
		"--from", mynode, // Use the key name for --from flag
		"--home", mynode, // Pass --home for keyring access
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId,
		"--gas", "auto",
		"--gas-prices", "7aphoton", // Explicitly setting gas prices
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort, // Target your local node's RPC
		"--yes", // Automatically confirm transaction
	)

	if err != nil {
		log.Errorf("❌ Failed to edit validator commission for '%s': %s\nOutput: %s", mynode, err, output)
		log.Warnf("Please ensure your validator is bonded and that the new commission rate adheres to 'max-rate' and 'max-change-rate' rules.")
		return err
	}

	log.Infof("✅ Validator commission edit transaction sent successfully for '%s'! Transaction output:\n%s", mynode, output)
	log.Info("Please monitor the chain and verify the new commission rate using 'mrmintchain validator-info --mynode %s'.", mynode)

	return nil
}

func queryProposalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-proposals",
		Short: "Query all active and past governance proposals",
		Long:  `Retrieves a list of all governance proposals on the chain, including their status (e.g., voting_period, passed, rejected).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return queryProposalsCmdLogic()
		},
	}
	return cmd
}

func queryProposalsCmdLogic() error {
	configCliParams = getConfigCliParams() // Ensure config is loaded

	err := godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Warnf("Could not load global .env file. Assuming default RPC port.")
	}

	rpcPort := getEnvOrFail("RPC_PORT")

	log.Infof("Querying all governance proposals from RPC: tcp://localhost:%s", rpcPort)

	output, cmdErr := runCmdCaptureOutput(Mrmintd,
		"query", "gov", "proposals",
		"--node", "tcp://localhost:"+rpcPort,
		"-o", "json",
	)

	if cmdErr != nil {
		if strings.Contains(output, "no proposals found") {
			log.Warnf("ℹ️ No governance proposals found on the chain.")
			return nil
		}
		log.Errorf("❌ Failed to query proposals: %s\nOutput: %s", cmdErr, output)
		log.Warnf("Please ensure your node is running and synced.")
		return cmdErr
	}

	var prettyJSON bytes.Buffer
	if err = json.Indent(&prettyJSON, []byte(output), "", "  "); err != nil {
		log.Errorf("Failed to pretty-print JSON output: %v", err)
		fmt.Println(output)
		return nil
	}

	log.Infof("✅ Successfully retrieved governance proposals:")
	fmt.Println(prettyJSON.String())

	return nil
}

func voteProposalCmd() *cobra.Command {
	var mynode string
	var proposalID uint64
	var voteOption string

	cmd := &cobra.Command{
		Use:   "vote-proposal",
		Short: "Cast a vote on a governance proposal",
		Long: `Cast your vote on a specific governance proposal.
Valid vote options are: "yes", "no", "abstain", "no_with_veto".
Example: --proposal-id 1 --option "yes"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return voteProposalCmdLogic(mynode, proposalID, voteOption)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (key name of the voter)")
	cmd.MarkFlagRequired("mynode")

	cmd.Flags().Uint64Var(&proposalID, "proposal-id", 0, "The ID of the proposal to vote on")
	cmd.MarkFlagRequired("proposal-id")

	cmd.Flags().StringVar(&voteOption, "option", "", "Your vote option: yes, no, abstain, no_with_veto")
	cmd.MarkFlagRequired("option")

	return cmd
}

func voteProposalCmdLogic(mynode string, proposalID uint64, voteOption string) error {
	configCliParams = getConfigCliParams() // Ensure config is loaded

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}

	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT")

	validOptions := map[string]bool{
		"yes":          true,
		"no":           true,
		"abstain":      true,
		"no_with_veto": true,
	}
	if !validOptions[strings.ToLower(voteOption)] {
		log.Errorf("❌ Invalid vote option: %s. Must be one of: yes, no, abstain, no_with_veto.", voteOption)
		return fmt.Errorf("invalid vote option")
	}

	log.Infof("Attempting to cast '%s' vote on proposal ID %d for voter '%s'", voteOption, proposalID, mynode)
	log.Infof("Sending vote transaction to local node RPC: tcp://localhost:%s", rpcPort)

	output, err := runCmdCaptureOutput(Mrmintd,
		"tx", "gov", "vote",
		fmt.Sprintf("%d", proposalID), // Proposal ID
		strings.ToLower(voteOption),   // Vote option
		"--from", mynode,
		"--home", mynode,
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId,
		"--gas", "auto",
		"--gas-prices", "7aphoton",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort, // Target your local node's RPC
		"--yes",
	)

	if err != nil {
		log.Errorf("❌ Failed to cast vote on proposal %d for '%s': %s\nOutput: %s", proposalID, mynode, err, output)
		log.Warnf("Please ensure your node is running and synced, the proposal is in the 'voting_period', and your key has funds for fees.")
		return err
	}

	log.Infof("✅ Vote transaction sent successfully for proposal %d! Transaction output:\n%s", proposalID, output)
	log.Info("You can verify your vote using 'ethermintd query gov vote %d %s --node tcp://localhost:%s'.", proposalID, mynode, rpcPort)

	return nil
}

func submitParamChangeProposalCmd() *cobra.Command {
	var mynode string
	var title string
	var description string
	var deposit string // This will be a separate flag for ethermintd tx
	var module string
	var paramKey string
	var paramValue string

	cmd := &cobra.Command{
		Use:   "submit-param-change-proposal",
		Short: "Submit a parameter change governance proposal (file-based)",
		Long: `Submits a proposal to change a specific parameter within a blockchain module.
This command generates a temporary JSON file for the proposal content.
It's used for parameters related to 'mint', 'rewards' (distribution), 'gov', 'staking', etc.
The proposal requires an initial deposit to enter the voting period.
Example: --module "mint" --param-key "MintDenom" --param-value "\"aphoton\""`, // Escaped quotes for JSON string
		RunE: func(cmd *cobra.Command, args []string) error {
			return submitParamChangeProposalCmdLogic(mynode, title, description, deposit, module, paramKey, paramValue)
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Name of the key (from your validator) to submit the proposal")
	cmd.MarkFlagRequired("mynode")
	cmd.Flags().StringVar(&title, "title", "", "Title of the proposal")
	cmd.MarkFlagRequired("title")
	cmd.Flags().StringVar(&description, "description", "", "Description of the proposal")
	cmd.MarkFlagRequired("description")
	cmd.Flags().StringVar(&deposit, "deposit", "", "Initial deposit amount (e.g., 1000000000000000000aphoton)")
	cmd.MarkFlagRequired("deposit")
	cmd.Flags().StringVar(&module, "module", "", "Name of the module to change parameter in (e.g., mint, distribution, gov, staking)")
	cmd.MarkFlagRequired("module")
	cmd.Flags().StringVar(&paramKey, "param-key", "", "Key of the parameter to change within the module (e.g., MintDenom, CommunityTax)")
	cmd.MarkFlagRequired("param-key")
	cmd.Flags().StringVar(&paramValue, "param-value", "", "New value for the parameter (must be correctly formatted JSON string if complex, e.g., '\"aphoton\"' for a string value, or '\"1000\"' for a number, or '{\"key\":\"value\"}' for an object)")
	cmd.MarkFlagRequired("param-value")

	return cmd
}

func submitParamChangeProposalCmdLogic(
	mynode, title, description, deposit, module, paramKey, paramValue string,
) error {
	configCliParams = getConfigCliParams()

	err := godotenv.Load(filepath.Join(mynode, ".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load node-specific .env for '%s': %v", mynode, err)
	}
	err = godotenv.Load(filepath.Join(".env"))
	if err != nil {
		log.Fatalf("❌ Failed to load global .env: %v", err)
	}

	rpcPort := getEnvOrFail("RPC_PORT")

	parsedParamValue := json.RawMessage(fmt.Sprintf("%q", paramValue))

	paramChangeContent := ParameterChangeProposalContent{
		Type:        "/cosmos.params.v1beta1.ParameterChangeProposal",
		Title:       title,
		Description: description,
		Changes: []ParamChange{
			{
				Subspace: module,
				Key:      paramKey,
				Value:    parsedParamValue,
			},
		},
	}

	paramChangeContentBytes, err := json.Marshal(paramChangeContent)
	if err != nil {
		log.Fatalf("❌ Failed to marshal param change content: %v", err)
	}

	govAuthorityAddress := "ethm10d07y265gmmuvt4z0w9aw880jnsr700jpva843" // **IMPORTANT: Make this dynamic if it changes!**

	legacyContentWrapper := MsgExecLegacyContentWrapper{
		Type:      "/cosmos.gov.v1.MsgExecLegacyContent",
		Content:   paramChangeContentBytes,
		Authority: govAuthorityAddress,
	}

	legacyContentWrapperBytes, err := json.Marshal(legacyContentWrapper)
	if err != nil {
		log.Fatalf("❌ Failed to marshal legacy content wrapper: %v", err)
	}

	proposalFile := ProposalFile{
		Messages: []json.RawMessage{legacyContentWrapperBytes},
		Deposit:  deposit,
	}

	proposalJSON, err := json.MarshalIndent(proposalFile, "", "  ")
	if err != nil {
		log.Fatalf("❌ Failed to marshal full proposal file to JSON: %v", err)
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "param-change-proposal-*.json")
	if err != nil {
		log.Fatalf("❌ Failed to create temporary file for proposal: %v", err)
	}

	if _, err := tmpFile.Write(proposalJSON); err != nil {
		log.Fatalf("❌ Failed to write proposal JSON to temporary file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatalf("❌ Failed to close temporary file: %v", err)
	}

	log.Infof("Submitting parameter change proposal for '%s' module, key '%s' to value '%s'", module, paramKey, paramValue)
	log.Infof("Proposal Title: '%s', Description: '%s', Deposit: '%s'", title, description, deposit)
	log.Infof("Using temporary proposal file: %s", tmpFile.Name())
	log.Infof("Sending proposal transaction to local node RPC: tcp://localhost:%s", rpcPort)
	log.Debugf("Generated Proposal JSON:\n%s", string(proposalJSON))

	output, cmdErr := runCmdCaptureOutput(Mrmintd,
		"tx", "gov", "submit-proposal",
		tmpFile.Name(),
		"--from", mynode,
		"--home", mynode,
		"--keyring-backend", "test",
		"--chain-id", configCliParams.ChaindId,
		"--gas", "auto",
		"--gas-prices", "7aphoton",
		"--gas-adjustment", "1.1",
		"--node", "tcp://localhost:"+rpcPort,
		"--yes",
	)

	if cmdErr != nil {
		log.Errorf("❌ Failed to submit parameter change proposal: %s\nOutput: %s", cmdErr, output)
		log.Warnf("Please ensure your node is running and synced, your key has sufficient funds for the deposit, and the parameter values are correctly formatted within the JSON structure.")
		return cmdErr
	}

	log.Infof("✅ Parameter change proposal submitted successfully! Transaction output:\n%s", output)
	log.Info("The proposal will enter the 'deposit_period'. If sufficient deposit is reached, it will move to 'voting_period'.")
	log.Info("You can track its status using 'mrmintchain query-proposals'.")

	return nil
}

// queryTxCmd defines the 'query-tx' command
func queryTxCmd() *cobra.Command {
	var mynode string // To specify the node directory to load .env
	// txHash will be taken as an argument, so no flag needed for it

	cmd := &cobra.Command{
		Use:   "query-tx [transaction_hash]",
		Short: "Query transaction details by hash",
		Long: `Queries the details of a specific blockchain transaction using its hash.
Requires the transaction hash as an argument and the node name (--mynode)
to load RPC connection details from the node's .env file.`,
		Args: cobra.ExactArgs(1), // Ensures exactly one argument (the transaction hash) is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			txHash := args[0] // Get the transaction hash from the command arguments
			return queryTxCmdLogic(mynode, txHash)
		},
	}

	// Add the --mynode flag to specify which node's .env to load
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name (directory where .env is located)")
	cmd.MarkFlagRequired("mynode") // Make the --mynode flag mandatory
	return cmd
}

// queryTxCmdLogic contains the core logic for querying transaction details
func queryTxCmdLogic(mynode, txHash string) error {
	// Load the .env file from the specified node directory
	// This is crucial to get the RPC_PORT for connecting to your node
	envPath := filepath.Join(mynode, ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		log.Fatalf("❌ Failed to load .env file from '%s' for node '%s': %v", envPath, mynode, err)
	}

	// Get the RPC port from the loaded environment variables
	rpcPort := getEnvOrFail("RPC_PORT")
	rpcLaddr := "tcp://localhost:" + rpcPort // Assuming RPC is exposed on localhost

	log.Infof("🔍 Attempting to query transaction %s using RPC endpoint: %s", txHash, rpcLaddr)

	// Execute the 'ethermintd query tx' command
	// We request JSON output for easier parsing if needed (even though we'll print raw here)
	output, err := runCmdCaptureOutput(Mrmintd, "query", "tx", txHash, "--node", rpcLaddr, "--output", "json")
	if err != nil {
		log.Errorf("❌ Failed to query transaction %s: %s", txHash, err)
		// Print the command's combined output for debugging
		fmt.Fprintln(os.Stderr, "Command Output (Error):", output)
		return err
	}

	// Print the raw JSON output of the transaction details to standard output
	fmt.Println(output)
	log.Info("✅ Transaction query complete.")
	return nil
}
