package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin0606/gw/internal/config"
	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/hook"
	"github.com/gin0606/gw/internal/pathutil"
)

// Init implements the "gw init" command.
func Init() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		return err
	}

	gwDir := filepath.Join(repoRoot, ".gw")
	if _, err := os.Stat(gwDir); err == nil {
		return fmt.Errorf(".gw/ already exists")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	repoName := git.RepoName(repoRoot)
	configContent := renderConfigTemplate(repoName)

	hooksDir := filepath.Join(gwDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return cleanupAndReturn(gwDir, fmt.Errorf("create .gw/hooks: %w", err))
	}

	if err := os.WriteFile(filepath.Join(gwDir, "config"), []byte(configContent), 0644); err != nil {
		return cleanupAndReturn(gwDir, fmt.Errorf("write .gw/config: %w", err))
	}

	for _, name := range hook.Names() {
		content := renderHookTemplate(name)
		if err := os.WriteFile(filepath.Join(hooksDir, string(name)), []byte(content), 0755); err != nil {
			return cleanupAndReturn(gwDir, fmt.Errorf("write .gw/hooks/%s: %w", name, err))
		}
	}

	fmt.Fprintf(os.Stderr, "Initialized .gw/ in %s\n", repoRoot)
	return nil
}

// cleanupAndReturn returns original after removing gwDir. If the cleanup
// itself fails, the cleanup error is joined to original so the user can tell
// why a follow-up `gw init` reports ".gw/ already exists" despite the
// previous error message. Passing a nil original would silently return nil
// on cleanup success and lose the original signal, so the function panics
// to make the misuse loud at the call site.
func cleanupAndReturn(gwDir string, original error) error {
	if original == nil {
		panic("cleanupAndReturn: original must be non-nil")
	}
	if rmErr := os.RemoveAll(gwDir); rmErr != nil {
		return errors.Join(original, fmt.Errorf("clean up partial %s: %w", gwDir, rmErr))
	}
	return original
}

func renderConfigTemplate(repoName string) string {
	return fmt.Sprintf("# See: gw help config\n%s = \"../%s%s\"\n", config.KeyWorktreesDir, repoName, pathutil.DefaultBaseDirSuffix)
}

func renderHookTemplate(name hook.Name) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "# gw hook: %s\n", name)
	b.WriteString("# Full hook semantics: gw help hooks\n")
	b.WriteString("#\n")
	b.WriteString("# Available environment variables:\n")
	for _, env := range hook.EnvVars() {
		fmt.Fprintf(&b, "#   %s\n", env)
	}
	b.WriteString("#\n")
	b.WriteString("# Example:\n")
	for _, line := range exampleLines(name) {
		fmt.Fprintf(&b, "# %s\n", line)
	}
	return b.String()
}

func exampleLines(name hook.Name) []string {
	switch name {
	case hook.PreAdd:
		return []string{
			"git fetch origin",
		}
	case hook.PostAdd:
		return []string{
			"npm install",
			fmt.Sprintf(`cp "$%s/.env" "$%s/.env"`, hook.EnvRepoRoot, hook.EnvWorktreePath),
		}
	case hook.PreRemove:
		return []string{
			"docker compose down",
		}
	case hook.PostRemove:
		return []string{
			"git fetch --prune origin",
			fmt.Sprintf(`if git merge-base --is-ancestor "$%s" origin/main 2>/dev/null; then`, hook.EnvBranch),
			fmt.Sprintf(`  git branch -D "$%s"`, hook.EnvBranch),
			"fi",
		}
	}
	return nil
}
