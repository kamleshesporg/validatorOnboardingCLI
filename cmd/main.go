package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Formatter:       log.TextFormatter, // This is a constant, not a struct
	})
	log.SetDefault(logger)

	var rootCmd = &cobra.Command{
		Use:   "mrmintchain",
		Short: "Full mrmint validator setup CLI tool",
	}

	rootCmd.AddCommand(
		initNodeCmd(),
		addKeyCmd(),
		addGenesisAccountCmd(),
		startNodeCmd(),
		portsAndEnvGenerationCmd(),
		stopNodeCmd(),
		restartNodeCmd(),
		stakeFundCmd(),
		getValidatorStatusCmd(),
		getValidatorBalanceCmd(),
		autoRunCmd(),
		setWithdrawAddress(),
		delegateSelfStakeCmd(),
		unjailCmd(),
		unstakeCmd(),
		withdrawRewardsCmd(),
		editCommissionCmd(),
		queryProposalsCmd(),
		voteProposalCmd(),
		submitParamChangeProposalCmd(),
		queryTxCmd(),
		createValidatorCmd(),
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

			if err := portsAndEnvGenerationLogic(mynode); err != nil {
				return fmt.Errorf("❌ collect gentxs failed: %w", err)
			}

			fmt.Println("✅ Validator setup completed successfully. Please run start-node command to start validator node.")
			return nil
		},
	}
	cmd.Flags().StringVar(&mynode, "mynode", "", "Please enter your node name")
	cmd.MarkFlagRequired("mynode")
	return cmd
}
