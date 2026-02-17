package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sanitize converts a branch name to a filesystem-safe directory name.
// Rules: replace "/" with "-", then trim leading/trailing hyphens.
func Sanitize(branch string) (string, error) {
	s := strings.ReplaceAll(branch, "/", "-")
	s = strings.Trim(s, "-")

	if s == "" || s == "." || s == ".." {
		return "", fmt.Errorf("invalid branch name %q: sanitized result is %q", branch, s)
	}

	return s, nil
}

// BaseDir resolves the worktree base directory from config or default.
func BaseDir(repoRoot, repoName, worktreesDir string) string {
	if worktreesDir == "" {
		return filepath.Join(repoRoot, "..", repoName+"-worktrees")
	}
	if filepath.IsAbs(worktreesDir) {
		return worktreesDir
	}
	return filepath.Join(repoRoot, worktreesDir)
}

// ComputePath returns the full worktree path for a branch.
func ComputePath(baseDir, branch string) (string, error) {
	sanitized, err := Sanitize(branch)
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, sanitized), nil
}

// ValidatePath checks that the target directory does not already exist.
func ValidatePath(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("directory already exists: %s", path)
	}
	return nil
}

// EnsureBaseDir creates the base directory if it doesn't exist.
func EnsureBaseDir(baseDir string) error {
	return os.MkdirAll(baseDir, 0755)
}
