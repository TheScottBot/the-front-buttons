package systemadapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The names inside the product folder (TD1).
const (
	configFileName     = "config.yaml"
	stateDirectoryName = "state"
)

// UserConfigLocation is the ConfigLocation adapter for both operating systems:
// Go's os.UserConfigDir already gives ~/Library/Application Support on macOS and
// %AppData% on Windows, so one implementation serves both (TD1).
type UserConfigLocation struct{}

// ConfigFile is the per machine configuration file.
func (UserConfigLocation) ConfigFile() (string, error) {
	productDirectory, directoryError := productDirectory()
	if directoryError != nil {
		return "", directoryError
	}

	return filepath.Join(productDirectory, configFileName), nil
}

// StateDirectory holds logs and other state the daemon writes.
func (UserConfigLocation) StateDirectory() (string, error) {
	productDirectory, directoryError := productDirectory()
	if directoryError != nil {
		return "", directoryError
	}

	return filepath.Join(productDirectory, stateDirectoryName), nil
}

func productDirectory() (string, error) {
	userConfigDirectory, directoryError := os.UserConfigDir()
	if directoryError != nil {
		return "", fmt.Errorf("finding the user configuration directory: %w", directoryError)
	}

	return filepath.Join(userConfigDirectory, platform.ProductIdentifier), nil
}
