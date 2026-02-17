package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
)

// Init implements the "gw init" command.
func Init(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "gw: error: unknown argument: %s\n", args[0])
		return 1
	}

	// 1. Detect repo root
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 2. Check if .gw/ already exists
	gwDir := filepath.Join(repoRoot, ".gw")
	if _, err := os.Stat(gwDir); err == nil {
		fmt.Fprintf(os.Stderr, "gw: error: .gw/ already exists\n")
		return 1
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 3. Create .gw/config and .gw/hooks/post-add
	repoName := git.RepoName(repoRoot)
	configContent := fmt.Sprintf("# See https://github.com/gin0606/gw\nworktrees_dir = \"../%s-worktrees\"\n", repoName)

	hooksDir := filepath.Join(gwDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	if err := os.WriteFile(filepath.Join(gwDir, "config"), []byte(configContent), 0644); err != nil {
		os.RemoveAll(gwDir)
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	postAddContent := `#!/bin/sh
# This hook is called after a worktree is created.
# See https://github.com/gin0606/gw for other available hooks.
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path
#   GW_BRANCH          - Branch name
#
# Example: Install dependencies
# npm install
`
	if err := os.WriteFile(filepath.Join(hooksDir, "post-add"), []byte(postAddContent), 0755); err != nil {
		os.RemoveAll(gwDir)
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "Initialized .gw/ in %s\n", repoRoot)
	return 0
}
