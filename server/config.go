package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ServerConfig struct {
	AuthToken string `json:"auth_token"`
	Port      int    `json:"port"`
	DBPath    string `json:"db_path"`
}

// Global server config
var config ServerConfig

func LoadServerConfig() error {
	// Default config
	config = ServerConfig{
		AuthToken: "itam-secret-token-default", // Fallback only
		Port:      8080,
		DBPath:    "data/itam.db",
	}

	// Look for config.json in the same directory as the executable
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.json")

	file, err := os.ReadFile(configPath)
	if err != nil {
		// Fallback to current working directory if not found next to exe
		file, err = os.ReadFile("config.json")
		if err != nil {
			absPath, _ := filepath.Abs("config.json")
			fmt.Printf("Config file NOT found next to exe or at: %s\nUsing default settings...\n", absPath)
			return nil
		}
		configPath, _ = filepath.Abs("config.json")
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	fmt.Printf("Successfully loaded config from: %s\n", configPath)
	return nil
}
