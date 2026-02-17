package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gin0606/gw/internal/config"
)

func TestLoad_NoConfigFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreesDir != "" {
		t.Errorf("expected empty WorktreesDir, got %q", cfg.WorktreesDir)
	}
}

func TestLoad_WithRelativeWorktreesDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `worktrees_dir = "../my-trees"`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreesDir != "../my-trees" {
		t.Errorf("got %q, want %q", cfg.WorktreesDir, "../my-trees")
	}
}

func TestLoad_WithAbsoluteWorktreesDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `worktrees_dir = "/tmp/trees"`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreesDir != "/tmp/trees" {
		t.Errorf("got %q, want %q", cfg.WorktreesDir, "/tmp/trees")
	}
}

func TestLoad_UnknownKeysIgnored(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
worktrees_dir = "../my-trees"
unknown_key = "value"
another_unknown = 42
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error for unknown keys: %v", err)
	}
	if cfg.WorktreesDir != "../my-trees" {
		t.Errorf("got %q, want %q", cfg.WorktreesDir, "../my-trees")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `this is not valid toml = [`)

	_, err := config.Load(dir)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func writeConfig(t *testing.T, repoRoot, content string) {
	t.Helper()
	configDir := filepath.Join(repoRoot, ".gw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
