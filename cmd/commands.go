package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	// For terminal display
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
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

func addKeyCmd(mynode string) *cobra.Command {
	return &cobra.Command{
		Use:   "add-key",
		Short: "Add key to keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			return addKeyCmdLogic(mynode)
		},
	}
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

	getConfirmationForPayment("Have you send MNT?")
	fmt.Scanln()

	return nil
}
func getConfirmationForPayment(s string) bool {

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (yes/no): ", s)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	if input == "yes" || input == "y" {
		fmt.Println("Continuing...")
		// Perform actions for "yes"
	} else if input == "no" || input == "n" {
		fmt.Println("Exiting...")
		// Perform actions for "no" or exit
	} else {
		fmt.Println("Invalid input. Please enter 'yes' or 'no'.")
		// Handle invalid input, possibly re-prompting the user
	}

	return true
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

func startNodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start-node",
		Short: "Start the Ethermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmd("ethermintd start --home ./myMrmintchainNode --p2p.laddr tcp://0.0.0.0:26666 --rpc.laddr tcp://0.0.0.0:26667 --grpc.address 0.0.0.0:9092 --grpc-web.address 0.0.0.0:9093 --json-rpc.address 0.0.0.0:8547")
		},
	}
}
