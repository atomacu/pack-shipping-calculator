package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
)

const DefaultHTTPPort = "8080"
const DefaultDatabasePath = "data/pack_calculator.db"

var DefaultPackSizes = []int{250, 500, 1000, 2000, 5000}

type Config struct {
	HTTPPort     string `json:"http_port"`
	DatabasePath string `json:"database_path"`
	PackSizes    []int  `json:"pack_sizes"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		HTTPPort:     DefaultHTTPPort,
		DatabasePath: DefaultDatabasePath,
		PackSizes:    slices.Clone(DefaultPackSizes),
	}

	if path == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	defer file.Close()

	var loaded Config
	if err := json.NewDecoder(file).Decode(&loaded); err != nil {
		return Config{}, err
	}

	if loaded.HTTPPort != "" {
		cfg.HTTPPort = loaded.HTTPPort
	}

	if loaded.DatabasePath != "" {
		cfg.DatabasePath = loaded.DatabasePath
	}

	if len(loaded.PackSizes) > 0 {
		cfg.PackSizes = slices.Clone(loaded.PackSizes)
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	slices.Sort(cfg.PackSizes)
	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.HTTPPort == "" {
		return fmt.Errorf("http_port is required")
	}
	if cfg.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}
	for _, size := range cfg.PackSizes {
		if size <= 0 {
			return fmt.Errorf("pack sizes must be positive: %d", size)
		}
	}
	return nil
}
