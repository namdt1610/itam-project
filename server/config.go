package main

import (
	"encoding/json"
	"fmt"
	"os"
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

	file, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println("Config file not found, using defaults")
		return nil // Not an error to run with defaults, but warning printed
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	fmt.Printf("Loaded server config from config.json (Port: %d)\n", config.Port)
	return nil
}
