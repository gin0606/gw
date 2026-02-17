package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the .gw/config file.
type Config struct {
	WorktreesDir string `toml:"worktrees_dir"`
}

// Load reads and parses .gw/config from the repository root.
// Returns default config if the file does not exist.
func Load(repoRoot string) (*Config, error) {
	configPath := filepath.Join(repoRoot, ".gw", "config")

	cfg := &Config{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .gw/config: %w", err)
	}

	return cfg, nil
}
