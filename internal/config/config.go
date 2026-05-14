package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/fireharp/tinkershop/internal/policy"
)

type Config struct {
	DBPath          string        `json:"db_path"`
	BlobDir         string        `json:"blob_dir"`
	Roots           []string      `json:"roots"`
	ScanWindowHours int           `json:"scan_window_hours"`
	Compression     string        `json:"compression"`
	Policies        []policy.Rule `json:"policies"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".local", "share", "tinkershop")
	return Config{
		DBPath:          filepath.Join(dataDir, "tinkershop.sqlite"),
		BlobDir:         filepath.Join(dataDir, "blobs"),
		Roots:           []string{filepath.Join(home, "Prog")},
		ScanWindowHours: 168,
		Compression:     "gzip",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DBPath == "" {
		return errors.New("db_path is required")
	}
	if c.BlobDir == "" {
		return errors.New("blob_dir is required")
	}
	if len(c.Roots) == 0 {
		return errors.New("at least one root is required")
	}
	if c.Compression == "" {
		return errors.New("compression is required")
	}
	if c.Compression != "gzip" && c.Compression != "none" {
		return errors.New("compression must be gzip or none")
	}
	return nil
}
