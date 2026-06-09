package pathutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Sanitize converts a branch name to a filesystem-safe directory name.
// Rules: replace "/" with separatorReplacement, then trim leading/trailing replacement chars.
func Sanitize(branch string) (string, error) {
	s := strings.ReplaceAll(branch, "/", separatorReplacement)
	s = strings.Trim(s, separatorReplacement)

	if s == "" || s == "." || s == ".." {
		return "", fmt.Errorf("invalid branch name %q: sanitized result is %q", branch, s)
	}

	return s, nil
}

// BaseDir resolves the worktree base directory from config or default.
func BaseDir(repoRoot, repoName, worktreesDir string) string {
	if worktreesDir == "" {
		return filepath.Join(repoRoot, "..", repoName+DefaultBaseDirSuffix)
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

// ValidatePath returns nil if path does not exist. Non-fs.ErrNotExist Stat
// errors (permission denied, ENOTDIR on a parent, etc.) are propagated so
// callers don't mistake them for "path is available".
func ValidatePath(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("directory already exists: %s", path)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("validate path %s: %w", path, err)
	}
	return nil
}

// EnsureBaseDir creates the base directory if it doesn't exist.
func EnsureBaseDir(baseDir string) error {
	return os.MkdirAll(baseDir, 0755)
}
