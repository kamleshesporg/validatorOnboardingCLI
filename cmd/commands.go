package main

import (
	"bufio"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	// For terminal display

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
	BootNodeRpc     string `jonsL"bootNodeRpc"`
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
	mynode = "/app/" + mynode

	genesisPath := mynode + "/config/genesis.json"

	if exists(genesisPath) {
		fmt.Println("⚠️  genesis.json already exists: " + genesisPath)
		if !yesNo("Delete and proceed?") {
			fmt.Println("Cancelled.")
			return nil
		}
		if err := os.RemoveAll(mynode); err != nil {
			return fmt.Errorf("failed to remove node folder: %w", err)
		}
	}

	if err := runCmd("ethermintd", "init", validatorName, "--chain-id", configCliParams.ChindId, "--home", mynode); err != nil {
		return fmt.Errorf("init command failed: %w", err)
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
	// permission := yesNo("Are you want to generate wallet ?")
	// if !permission {
	// 	log.Fatalf("Key Generation process stop.")
	// }

	validatorName := mynode
	mynode = "/app/" + mynode

	output, err := runCmdCaptureOutput("ethermintd", "keys", "add", validatorName, "--algo", "eth_secp256k1", "--keyring-backend", "test", "--home", mynode)
	if err != nil {
		return fmt.Errorf("keys add command failed: %w\nOutput: %s", err, output)
	}

	fmt.Println("Key generation output:\n", output)
	fmt.Println("\n🔑 Please copy your key output above. Press Enter to continue...")
	// fmt.Scanln()
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
	mynode = "/app/" + mynode

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
	fmt.Printf("Ethereum address: %s", ethAddress)

	fmt.Println("\n📲 QR Code (scan it securely): Please send %d MNT coin to your validator wallet for validator staking.", configCliParams.MinStakeFund)
	fmt.Println("Its your validator wallet : ")
	fmt.Println("Default ethm1 format : ", ethm1Address)
	fmt.Println("Converted into 0x format :", ethAddress)
	qrterminal.GenerateHalfBlock(ethAddress, qrterminal.L, os.Stdout)

	getConfirmationForPayment("Have you send MNT?", ethm1Address)
	fmt.Println("Now stake cmd run....")
	// fmt.Scanln()

	return nil
}
func getConfirmationForPayment(s string, ethm1Address string) bool {

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (yes/no): ", s)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	if input == "yes" || input == "y" {
		if getBalanceCmdLogic(ethm1Address) {
			fmt.Println("\n ✅ Your fund deposited Now")
			return true
		} else {
			fmt.Println("\n ❌ Balance not deposited yet, Please try again.")
			getConfirmationForPayment(s, ethm1Address)
		}
		// Perform actions for "yes"
	} else if input == "no" || input == "n" {
		fmt.Println("Please deposit mnt first then you can proceed")
		getConfirmationForPayment(s, ethm1Address)
		// Perform actions for "no" or exit
	} else {
		fmt.Println("Invalid input. Please enter 'yes' or 'no'.")
		// Handle invalid input, possibly re-prompting the user
		getConfirmationForPayment(s, ethm1Address)
	}

	return true
}

// Define a struct for items within the 'balances' array
type BalanceItem struct {
	Amount string `yaml:"amount"` // Ensure the field name matches your struct (e.g., Amount)
	Denom  string `yaml:"denom"`
}

// Updated struct for the overall YAML output
type CmdOutput struct {
	Balances   []BalanceItem `yaml:"balances"`
	Pagination struct {
		NextKey interface{} `yaml:"next_key"`
		Total   string      `yaml:"total"`
	} `yaml:"pagination"`
}

func getBalanceCmdLogic(walletEthmAddress string) bool {
	bootRpc := configCliParams.BootNodeRpc
	if bootRpc == "" {
		fmt.Errorf("Boot node rpc not provided")
		return false
	}
	output, err := runCmdCaptureOutput("ethermintd", "query", "bank", "balances", walletEthmAddress, "--node", bootRpc)
	if err != nil {
		fmt.Errorf("Get balance command failed: %w\nOutput: %s", err, output)
		return false
	}
	var cResp CmdOutput

	err = yaml.Unmarshal([]byte(output), &cResp)
	if err != nil {
		fmt.Errorf("Get balance command failed: %w", err)
		return false
	}

	if len(cResp.Balances) == 0 {
		fmt.Println("The balances array is indeed empty, as expected. Please deposit fund then proceed")
		return false
	} else {

		bigAmount := new(big.Int)
		bigAmount, ok := bigAmount.SetString(cResp.Balances[0].Amount, 10)
		if !ok {
			fmt.Println("Invalid number")
			return false
		}

		// Divide by Wei (as big.Int)
		wei := big.NewInt(1e18)
		exactBalance := new(big.Int).Div(bigAmount, wei)

		if err != nil {
			fmt.Println("Error:", err)
			return false
		}

		// This block would only execute if balances was not empty.
		fmt.Println(" 💸 The balances is :", exactBalance, cResp.Balances[0].Denom)
		if exactBalance.Int64() < configCliParams.MinStakeFund {
			fmt.Println(" 😧 The balances is less then mininmum deposit amount %d mnt, Please deposit more", configCliParams.MinStakeFund)
			return false
		}

	}
	return true
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

// func startNodeCmdLogic(mynode string) error {
// 	portsArray := []string{}

// 	port1 := getPortInputAndCheck("\n 👉 Please enter 5 digit port for p2p-laddr:", "26666", portsArray)
// 	portsArray = append(portsArray, port1)
// 	p2pladdr := "tcp://0.0.0.0:" + port1
// 	fmt.Println("    ✅ p2p-laddr:", p2pladdr)

// 	port2 := getPortInputAndCheck("\n 👉 Please enter 5 digit port for rpc-laddr:", "26667", portsArray)
// 	portsArray = append(portsArray, port2)
// 	rpcladdr := "tcp://0.0.0.0:" + port2
// 	fmt.Println("    ✅ rpc-laddr:", rpcladdr)

// 	port3 := getPortInputAndCheck("\n 👉 Please enter 4 digit port for grpc-address:", "9092", portsArray)
// 	portsArray = append(portsArray, port3)
// 	grpcAddress := "0.0.0.0:" + port3
// 	fmt.Println("    ✅ grpc-address:", grpcAddress)

// 	port4 := getPortInputAndCheck("\n 👉 Please enter 4 digit port for grpc-web-address:", "9093", portsArray)
// 	portsArray = append(portsArray, port4)
// 	grpcwebaddress := "0.0.0.0:" + port4
// 	fmt.Println("    ✅ grpc-web-address:", grpcwebaddress)

// 	port5 := getPortInputAndCheck("\n 👉 Please enter 4 digit port for json-rpc-address:", "8547", portsArray)
// 	jsonrpcaddress := "0.0.0.0:" + port5
// 	fmt.Println("    ✅ json-rpc-address:", jsonrpcaddress)

// 	err := runCmd("ethermintd", "start",
// 		"--home", mynode,
// 		"--p2p.laddr", p2pladdr,
// 		"--rpc.laddr", rpcladdr,
// 		"--grpc.address", grpcAddress,
// 		"--grpc-web.address", grpcwebaddress,
// 		"--json-rpc.address", jsonrpcaddress,
// 		"--p2p.persistent_peers", configCliParams.PersistentPeers)
// 	if err != nil {
// 		return fmt.Errorf("node start command failed: %w", err)
// 	}
// 	fmt.Printf("Node Started!!!!!!!!!")
// 	return err
// }

func portsAndEnvGeneration(mynode string) error {
	portsArray := []string{}
	fmt.Println(configCliParams)
	p2p := getPortInputAndCheck("P2P_PORT", "26666", portsArray)
	portsArray = append(portsArray, p2p)

	rpc := getPortInputAndCheck("RPC_PORT", "26667", portsArray)
	portsArray = append(portsArray, rpc)

	grpc := getPortInputAndCheck("GRPC_PORT", "9092", portsArray)
	portsArray = append(portsArray, grpc)

	grpcWeb := getPortInputAndCheck("GRPC_WEB_PORT", "9093", portsArray)
	portsArray = append(portsArray, grpcWeb)

	jsonRpc := getPortInputAndCheck("JSON_RPC_PORT", "8547", portsArray)
	// portsArray = append(portsArray, jsonRpc)

	// ports := map[string]string{
	// 	"P2P_PORT":         p2p,
	// 	"RPC_PORT":         rpc,
	// 	"GRPC_PORT":        grpc,
	// 	"GRPC_WEB_PORT":    grpcWeb,
	// 	"JSON_RPC_PORT":    jsonRpc,
	// 	"PERSISTENT_PEERS": configCliParams.PersistentPeers,
	// }

	// err := savePortsToEnvFile("/app/.env", ports)
	// if err != nil {
	// 	log.Fatal("Failed to save ports:", err)
	// }
	// return err

	// Construct .env content
	envContent := fmt.Sprintf(`P2P_PORT=%s
RPC_PORT=%s
GRPC_PORT=%s
GRPC_WEB_PORT=%s
JSON_RPC_PORT=%s
PERSISTENT_PEERS=%s`, p2p,
		rpc,
		grpc,
		grpcWeb,
		jsonRpc,
		configCliParams.PersistentPeers)

	// Path to write .env
	envPath := filepath.Join(mynode, ".env")

	// Ensure the directory exists
	if err := os.MkdirAll(mynode, os.ModePerm); err != nil {
		fmt.Printf("❌ Failed to create node directory: %v\n", err)
		os.Exit(1)
	}

	// Write to .env
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write .env file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ .env file generated at %s\n", envPath)
	return err
}

// func startNodeCmd() *cobra.Command {
// 	var mynode string
// 	var p2pPort, rpcPort, grpcPort, grpcWebPort, jsonRpcPort string

// 	cmd := &cobra.Command{
// 		Use:   "start-node",
// 		Short: "Start the Ethermint node",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			return startNodeCmdLogic(mynode, p2pPort, rpcPort, grpcPort, grpcWebPort, jsonRpcPort, cmd)
// 		},
// 	}

// 	cmd.Flags().StringVar(&mynode, "mynode", "", "Node name (required)")
// 	cmd.Flags().StringVar(&p2pPort, "p2p-port", "", "Port for p2p.laddr (5 digits)")
// 	cmd.Flags().StringVar(&rpcPort, "rpc-port", "", "Port for rpc.laddr (5 digits)")
// 	cmd.Flags().StringVar(&grpcPort, "grpc-port", "", "Port for grpc.address (4 digits)")
// 	cmd.Flags().StringVar(&grpcWebPort, "grpc-web-port", "", "Port for grpc-web.address (4 digits)")
// 	cmd.Flags().StringVar(&jsonRpcPort, "json-rpc-port", "", "Port for json-rpc.address (4 digits)")
// 	cmd.MarkFlagRequired("mynode")

// 	return cmd
// }

func startNodeCmdLogic(mynode string) error {
	// Load the .env file

	mynode = "/app/" + mynode

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

	fmt.Println("✅ Using Ports from ENV:")
	fmt.Println("  - p2p-laddr:", p2pLaddr)
	fmt.Println("  - rpc-laddr:", rpcLaddr)
	fmt.Println("  - grpc-address:", grpcAddress)
	fmt.Println("  - grpc-web-address:", grpcWebAddress)
	fmt.Println("  - json-rpc-address:", jsonRpcAddress)
	fmt.Println("  - persistent-peers:", PersistentPeers)

	// Run the command with ports from ENV
	err = runCmd("ethermintd", "start",
		"--home", mynode,
		"--p2p.laddr", p2pLaddr,
		"--rpc.laddr", rpcLaddr,
		"--grpc.address", grpcAddress,
		"--grpc-web.address", grpcWebAddress,
		"--json-rpc.address", jsonRpcAddress,
		"--p2p.persistent_peers", PersistentPeers)
	if err != nil {
		return fmt.Errorf("❌ node start command failed: %w", err)
	}

	fmt.Println("🚀 Node started successfully!")
	return nil
}

func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("❌ Missing required environment variable: %s", key)
	}
	return value
}

func savePortsToEnvFile(envPath string, ports map[string]string) error {
	file, err := os.Create(envPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for key, value := range ports {
		line := fmt.Sprintf("%s=%s\n", key, value)
		_, err := file.WriteString(line)
		if err != nil {
			return err
		}
	}
	fmt.Println("✅ Ports saved to", envPath)
	return nil
}

func getPortInputAndCheck(prompt string, defaultPort string, existing []string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [%s]: ", prompt, defaultPort)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			input = defaultPort
		}

		// Check numeric
		if _, err := strconv.Atoi(input); err != nil {
			fmt.Println("❌ Invalid input. Please enter numeric port.")
			continue
		}

		// Check port length (4 or 5 digits)
		if len(input) != 4 && len(input) != 5 {
			fmt.Println("❌ Port must be 4 or 5 digits.")
			continue
		}

		// Check for duplicates
		if checkArrayAlreadyExists(existing, input) {
			fmt.Printf("❌ Port %s already used.\n", input)
			continue
		}

		// Check availability
		if err := checkPort(input); err != nil {
			fmt.Printf("❌ Port %s not available: %s\n", input, err)
			continue
		}

		return input
	}
}

// func getPortInputAndCheck(s string, defualtPort string, portsArray []string) string {

// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Printf("%s (default: [%s]): ", s, defualtPort)
// 	input, _ := reader.ReadString('\n')
// 	input = strings.TrimSpace(input)
// 	input = strings.ToLower(input)

// 	if input == "" {
// 		input = defualtPort
// 		// return getPortInputAndCheck(s, defualtPort, portsArray)
// 	}
// 	if strings.Count(defualtPort, "") != strings.Count(input, "") {
// 		fmt.Printf("❌ Invalid port length")
// 		return getPortInputAndCheck(s, defualtPort, portsArray)
// 	}
// 	if checkArrayAlreadyExists(portsArray, input) {
// 		fmt.Printf("❌ Port %s is arleady used, try another one", input)
// 		return getPortInputAndCheck(s, defualtPort, portsArray)
// 	}
// 	if err := checkPort(input); err != nil {
// 		fmt.Printf("❌ Port %s is : %s\n", input, err)
// 		return getPortInputAndCheck(s, defualtPort, portsArray)
// 	}
// 	return input
// }

func checkPort(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

//	func checkPort(port string) error {
//		ln, err := net.Listen("tcp", ":"+port)
//		if err != nil {
//			fmt.Println("PORT CHECK : ", err)
//			return err
//		}
//		defer ln.Close()
//		return nil
//	}
func checkArrayAlreadyExists(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
