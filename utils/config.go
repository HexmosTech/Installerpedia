package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ReportBugs bool `toml:"report-bugs"`
	Debug      bool `toml:"debug"`
}

var GlobalConfig = Config{
	ReportBugs: true,  // Default
	Debug:      false, // Default: no debug logs unless explicitly enabled
}

func LoadConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return // Fallback to defaults if home dir is inaccessible
	}

	// Use .ipm.toml (hidden file convention for home directory)
	configPath := filepath.Join(home, ".ipm.toml")

	if _, err := os.Stat(configPath); err == nil {
		_, _ = toml.DecodeFile(configPath, &GlobalConfig)
	}
	// If no file or no "debug" key: GlobalConfig.Debug stays false (default)
}

func SaveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".ipm.toml")
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(GlobalConfig)
}

func DebugLog(format string, args ...interface{}) {
	if GlobalConfig.Debug {
		fmt.Printf(format, args...)
	}
}
