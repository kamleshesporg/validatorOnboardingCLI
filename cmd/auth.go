package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const (
	configDirName = ".mrmintchain"
	authFileName  = "auth.json"
)

// AuthConfig holds the authentication token.
type AuthConfig struct {
	Token string `json:"token"`
}

// getAuthFilePath returns the full path to the authentication file.
func getAuthFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	return filepath.Join(homeDir, configDirName, authFileName), nil
}

// saveAuthToken saves the authentication token to a file in the user's home directory.
func saveAuthToken(token string) error {
	authFilePath, err := getAuthFilePath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(authFilePath)
	if err := os.MkdirAll(configDir, 0700); err != nil { // rwx------
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	authConfig := AuthConfig{Token: token}
	configData, err := json.MarshalIndent(authConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}

	// Write with permissions that only the current user can read/write.
	if err := os.WriteFile(authFilePath, configData, 0600); err != nil { // rw-------
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	log.Infof("✅ Authentication token saved to %s", authFilePath)
	return nil
}

// loadAuthToken loads the authentication token from the config file.
func loadAuthToken() (string, error) {
	authFilePath, err := getAuthFilePath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(authFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("not logged in. Please run 'mrmintchain login' first")
	}

	configData, err := os.ReadFile(authFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read auth file: %w", err)
	}

	var authConfig AuthConfig
	if err := json.Unmarshal(configData, &authConfig); err != nil {
		return "", fmt.Errorf("failed to parse auth file: %w", err)
	}

	if authConfig.Token == "" {
		return "", fmt.Errorf("auth token is empty in config file. Please log in again")
	}

	return authConfig.Token, nil
}

// logoutCmd defines the command to delete the authentication token file.
func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from the platform",
		Long:  `Removes the stored authentication token, requiring you to log in again for authenticated commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutCmdLogic()
		},
	}
	return cmd
}

// logoutCmdLogic handles the deletion of the auth file.
func logoutCmdLogic() error {
	authFilePath, err := getAuthFilePath()
	if err != nil {
		return err // The error from getAuthFilePath is already descriptive
	}

	// Check if the file exists before trying to remove it.
	if _, err := os.Stat(authFilePath); os.IsNotExist(err) {
		log.Info("ℹ️ You are already logged out.")
		return nil
	}

	// Remove the authentication file.
	if err := os.Remove(authFilePath); err != nil {
		return fmt.Errorf("failed to remove authentication file: %w", err)
	}

	log.Info("✅ You have been successfully logged out.")
	return nil
}
