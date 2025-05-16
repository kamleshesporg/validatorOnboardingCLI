package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "mrmintchain",
		Short: "Full mrmintchain validator setup CLI tool",
	}

	rootCmd.AddCommand(
		initNodeCmd(),
		addKeyCmd("kamleshnode"),
		addGenesisAccountCmd(),
		gentxCmd(),
		collectGentxsCmd(),
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
	return &cobra.Command{
		Use:   "auto-run",
		Short: "Automatically run the full validator setup process",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🚀 Starting full validator setup...")

			if err := initNodeLogic("kamleshnode"); err != nil {
				return fmt.Errorf("❌ init node failed: %w", err)
			}
			if err := addKeyCmdLogic("kamleshnode"); err != nil {
				return fmt.Errorf("❌ add key failed: %w", err)
			}
			// if err := addGenesisAccountCmd(); err != nil {
			// 	return fmt.Errorf("❌ add genesis account failed: %w", err)
			// }
			// if err := gentxCmd(); err != nil {
			// 	return fmt.Errorf("❌ gentx failed: %w", err)
			// }
			// if err := collectGentxsCmd(); err != nil {
			// 	return fmt.Errorf("❌ collect gentxs failed: %w", err)
			// }
			// if err := startNodeCmd(); err != nil {
			// 	return fmt.Errorf("❌ start node failed: %w", err)
			// }

			fmt.Println("✅ Validator setup completed successfully.")
			return nil
		},
	}
}
