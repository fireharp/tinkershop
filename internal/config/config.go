package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fireharp/tinkershop/internal/policy"
)

type Config struct {
	DBPath          string        `json:"db_path"`
	BlobDir         string        `json:"blob_dir"`
	Roots           []string      `json:"roots"`
	Since           string        `json:"since,omitempty"`
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

func (c Config) SinceTime(now time.Time) (*time.Time, error) {
	if c.Since == "" {
		return nil, nil
	}

	value := strings.TrimSpace(c.Since)
	if days, ok, err := parseDays(value); ok || err != nil {
		if err != nil {
			return nil, err
		}
		cutoff := now.AddDate(0, 0, -days).UTC()
		return &cutoff, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			cutoff := parsed.UTC()
			return &cutoff, nil
		}
	}

	return nil, errors.New("since must be YYYY-MM-DD, RFC3339, or a day window like 14d")
}

func parseDays(value string) (int, bool, error) {
	if !strings.HasSuffix(value, "d") {
		return 0, false, nil
	}

	rawDays := strings.TrimSuffix(value, "d")
	days, err := strconv.Atoi(rawDays)
	if err != nil || days < 0 {
		return 0, true, errors.New("day window must look like 14d")
	}
	return days, true, nil
}
