package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
)

func getConfirmationForPayment(s string, ethm1Address string) bool {

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (yes/no): ", s)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	if input == "yes" || input == "y" {
		_, exactBalance := getBalanceCmdLogic(ethm1Address)
		if exactBalance < configCliParams.MinStakeFund {
			log.Errorf("😧 The balances is less then mininmum deposit amount %d mnt, Please deposit more", configCliParams.MinStakeFund)

			log.Error("❌ Balance not deposited yet, Please try again.")
			getConfirmationForPayment(s, ethm1Address)
		} else {
			log.Info("✅ Your fund deposited!")
			return true
		}
		// Perform actions for "yes"
	} else if input == "no" || input == "n" {
		fmt.Println("Please deposit mnt first then you can proceed")
		getConfirmationForPayment(s, ethm1Address)
		// Perform actions for "no" or exit
	} else {
		log.Info("Invalid input. Please enter 'yes' or 'no'.")
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

func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("❌ Missing required environment variable: %s", key)
	}
	return value
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
			log.Error("❌ Invalid input. Please enter numeric port.")
			continue
		}

		// Check port length (4 or 5 digits)
		if len(input) != 4 && len(input) != 5 {
			log.Error("❌ Port must be 4 or 5 digits.")
			continue
		}

		// Check for duplicates
		if checkArrayAlreadyExists(existing, input) {
			log.Errorf("❌ Port %s already used.\n", input)
			continue
		}

		// Check availability
		if err := checkPort(input); err != nil {
			log.Errorf("❌ Port %s not available: %s\n", input, err)
			continue
		}

		return input
	}
}

func checkPort(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

func checkArrayAlreadyExists(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func getStakingInputs(prompt string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			input = defaultValue
		}

		// Check numeric
		if _, err := strconv.ParseFloat(input, 64); err != nil {
			log.Error("❌ Invalid input. Please enter numeric port.")
			continue
		}

		// // Check port length (4 or 5 digits)
		// if len(input) != 4 && len(input) != 5 {
		// 	log.Error("❌ Port must be 4 or 5 digits.")
		// 	continue
		// }
		return input
	}
}
