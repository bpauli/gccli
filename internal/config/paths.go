package config

import (
	"os"
	"path/filepath"
)

const appName = "gccli"

// ConfigDir returns the configuration directory for the application.
// It uses $XDG_CONFIG_HOME/gccli or ~/.config/gccli as fallback.
func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName), nil
}

// ConfigFilePath returns the path to the main configuration file.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// CredentialsDir returns the directory for credential-related files.
func CredentialsDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}
