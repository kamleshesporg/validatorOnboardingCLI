package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {

	var rootCmd = &cobra.Command{
		Use:   "mrmintchain",
		Short: "Full mrmint validator setup CLI tool",
	}

	rootCmd.AddCommand(
		initNodeCmd(),
		addKeyCmd(),
		addGenesisAccountCmd(),
		startNodeCmd(),
		autoRunCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// 🆕 Auto-run command that runs everything in order
func autoRunCmd() *cobra.Command {
	var mynode string

	cmd := &cobra.Command{
		Use:   "auto-setup",
		Short: "Automatically run the full validator setup process",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🚀 Starting full validator setup...")

			if err := initNodeLogic(mynode); err != nil {
				return fmt.Errorf("❌ init node failed: %w", err)
			}

			fmt.Println("🚀 Starting validator key generation...")
			if err := addKeyCmdLogic(mynode); err != nil {
				return fmt.Errorf("❌ add key failed: %w", err)
			}
			if err := addGenesisAccountLogic(mynode); err != nil {
				return fmt.Errorf("❌ add genesis account failed: %w", err)
			}
			// if err := gentxCmd(); err != nil {
			// 	return fmt.Errorf("❌ gentx failed: %w", err)
			// }
			if err := portsAndEnvGeneration(mynode); err != nil {
				return fmt.Errorf("❌ collect gentxs failed: %w", err)
			}
			// if err := startNodeCmdLogic(mynode); err != nil {
			// 	return fmt.Errorf("❌ start node failed: %w", err)
			// }

			fmt.Println("✅ Validator setup completed successfully.")
			return nil
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}

// ethermintd start --home kamleshnode001 --p2p.laddr tcp://0.0.0.0:46666 --rpc.laddr tcp://0.0.0.0:46667 --grpc.address 0.0.0.0:4092 --grpc-web.address 0.0.0.0:4093 --json-rpc.address 0.0.0.0:4545 --p2p.persistent_peers 29996f0c7cc853d551e280a8162480fcd684f0b8@127.0.0.1:26656
