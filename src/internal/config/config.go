package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Token        string `json:"token"`
	NotionVersion string `json:"notion_version"`
}

func Load() (*Config, string, error) {
	path, err := filePath()
	if err != nil {
		return &Config{}, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, path, nil
		}
		return &Config{}, path, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return &Config{}, path, err
	}

	return cfg, path, nil
}

func Save(cfg *Config) (string, error) {
	path, err := filePath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}

	return path, nil
}

func Clear() (string, error) {
	path, err := filePath()
	if err != nil {
		return "", err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return path, nil
}

func filePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "notion-cli", "config.json"), nil
}
