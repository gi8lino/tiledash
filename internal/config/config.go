package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads the dashboard configuration from the given path.
func LoadConfig(path string) (DashboardConfig, error) {
	cfg := DashboardConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	return cfg, nil
}
