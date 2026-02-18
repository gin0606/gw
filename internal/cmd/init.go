package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
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
	} else if !os.IsNotExist(err) {
		return err
	}

	repoName := git.RepoName(repoRoot)
	configContent := fmt.Sprintf("# See https://github.com/gin0606/gw\nworktrees_dir = \"../%s-worktrees\"\n", repoName)

	hooksDir := filepath.Join(gwDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(gwDir, "config"), []byte(configContent), 0644); err != nil {
		os.RemoveAll(gwDir)
		return err
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
		return err
	}

	fmt.Fprintf(os.Stderr, "Initialized .gw/ in %s\n", repoRoot)
	return nil
}
