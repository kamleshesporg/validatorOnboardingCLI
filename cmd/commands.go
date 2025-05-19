package main

import (
	"bufio"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"

	// For terminal display
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

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

	if err := runCmd("ethermintd", "init", mynode, "--chain-id", "os_9000-1", "--home", mynode); err != nil {
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
	permission := yesNo("Are you want to generate wallet ?")
	if !permission {
		log.Fatalf("Key Generation process stop.")
	}

	output, err := runCmdCaptureOutput("ethermintd", "keys", "add", mynode, "--algo", "eth_secp256k1", "--keyring-backend", "test", "--home", mynode)
	if err != nil {
		return fmt.Errorf("keys add command failed: %w\nOutput: %s", err, output)
	}

	fmt.Println("Key generation output:\n", output)
	fmt.Println("\n🔑 Please copy your key output above. Press Enter to continue...")
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
			// addr := string(addrOut)
			// return runCmd("ethermintd", "add-genesis-account", addr, "1000000000000000000aphoton")
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

func addGenesisAccountLogic(mynode string) error {
	getAddr := exec.Command("ethermintd", "keys", "show", mynode, "-a", "--home", mynode, "--keyring-backend", "test")
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

	fmt.Println("\n📲 QR Code (scan it securely): Please send 50 MNT coin to your validator wallet for validator staking.")
	fmt.Println("Its your validator wallet : ")
	fmt.Println("Default ethm1 format : ", ethm1Address)
	fmt.Println("Converted into 0x format :", ethAddress)
	qrterminal.GenerateHalfBlock(ethAddress, qrterminal.L, os.Stdout)

	getConfirmationForPayment("Have you send MNT?", ethm1Address)
	fmt.Println("Now stake cmd run....")
	fmt.Scanln()

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

	output, err := runCmdCaptureOutput("ethermintd", "query", "bank", "balances", walletEthmAddress, "--node", "tcp://localhost:26657")
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

		fmt.Println("The balance is:", exactBalance.String())

		if err != nil {
			fmt.Println("Error:", err)
			return false
		}

		// This block would only execute if balances was not empty.
		fmt.Println("The balances is :", exactBalance, cResp.Balances[0].Denom)
		if exactBalance.Int64() < 50 {
			fmt.Println("The balances is less then mininmum deposit amount 50mnt, Please deposit more")
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

func startNodeCmdLogic(mynode string) error {

	p2pladdr := "tcp://0.0.0.0:" + getPortInputAndCheck("Please enter 6digit port for p2p-laddr:", "26666")
	fmt.Println("p2pladdr:", p2pladdr)
	rpcladdr := "tcp://0.0.0.0:" + getPortInputAndCheck("Please enter 6digit port for rpc-laddr:", "26667")
	fmt.Println("rpcladdr:", rpcladdr)
	grpcAddress := "0.0.0.0:" + getPortInputAndCheck("Please enter 4digit port for rpc-laddr:", "9092")
	fmt.Println("grpcAddress:", grpcAddress)
	grpcwebaddress := "0.0.0.0:" + getPortInputAndCheck("Please enter 4digit port for grpc-web-address:", "9093")
	fmt.Println("grpcwebaddress:", grpcwebaddress)
	jsonrpcaddress := "0.0.0.0:" + getPortInputAndCheck("Please enter 4digit port for json-rpc-address:", "8547")
	fmt.Println("jsonrpcaddress:", jsonrpcaddress)

	output, err := runCmdCaptureOutput("ethermintd", "start",
		"--home", mynode,
		"--p2p.laddr", p2pladdr,
		"--rpc.laddr", rpcladdr,
		"--grpc.address", grpcAddress,
		"--grpc-web.address", grpcwebaddress,
		"--json-rpc.address", jsonrpcaddress)
	if err != nil {
		return fmt.Errorf("node start command failed: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Node Started:", output)
	return err
}

func getPortInputAndCheck(s string, defualtPort string) string {

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (Ex: %d): ", s, defualtPort)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)
	if input == "" {
		return getPortInputAndCheck(s, defualtPort)
	}
	if err := checkPort(input); err != nil {
		fmt.Printf("Port %s is already in use: %s\n", input, err)
		return getPortInputAndCheck(s, defualtPort)
	}
	return input
}

func checkPort(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("PORT CHECK : ", err)
		return err
	}
	defer ln.Close()
	return nil
}

func gentxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gentx",
		Short: "Generate genesis transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmd("ethermintd", "gentx", "validator", "1000000000000000000aphoton",
				"--chain-id", "os_9000-1",
				"--moniker", "mynode",
				"--keyring-backend", "test")
		},
	}
}

func collectGentxsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "collect-gentxs",
		Short: "Collect gentxs into genesis.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmd("ethermintd", "collect-gentxs")
		},
	}
}
