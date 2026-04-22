package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ServerAddr   string   `json:"server_addr"`
	ScanDirs     []string `json:"scan_dirs"`
	IndexDir     string   `json:"index_dir"`
	ExtractEXIF  bool     `json:"extract_exif"`
	MLServiceURL string   `json:"ml_service_url"`
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerAddr:   ":8080",
		ScanDirs:     []string{},
		IndexDir:     "data/index",
		ExtractEXIF:  true,
		MLServiceURL: "http://localhost:8000",
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Ensure index directory exists
	if err := os.MkdirAll(cfg.IndexDir, 0755); err != nil {
		return nil, err
	}

	// Default scan directory: ~/Pictures
	if len(cfg.ScanDirs) == 0 {
		home, _ := os.UserHomeDir()
		cfg.ScanDirs = []string{filepath.Join(home, "Pictures")}
	}

	return cfg, nil
}
