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

	hooks := []struct {
		name    string
		content string
	}{
		{"pre-add", `#!/bin/sh
# This hook is called before a worktree is created.
# Working directory: repository root
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path (to be created)
#   GW_BRANCH          - Branch name
#
# Exit non-zero to abort worktree creation.
#
# Example: Fetch latest remote so the new worktree starts from up-to-date origin/main
# git fetch origin
`},
		{"post-add", `#!/bin/sh
# This hook is called after a worktree is created.
# Working directory: the new worktree
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path
#   GW_BRANCH          - Branch name
#
# Example: Install dependencies and copy files not tracked by git
# npm install
# cp "$GW_REPO_ROOT/.env" "$GW_WORKTREE_PATH/.env"
`},
		{"pre-remove", `#!/bin/sh
# This hook is called before a worktree is removed.
# Working directory: the worktree being removed
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path (to be removed)
#   GW_BRANCH          - Branch name
#
# Exit non-zero to abort worktree removal (skipped with --force).
#
# Example: Stop development servers before removing the worktree
# docker compose down
`},
		{"post-remove", `#!/bin/sh
# This hook is called after a worktree is removed.
# Working directory: repository root
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path (already removed)
#   GW_BRANCH          - Branch name
#
# Example: Fetch and delete the branch if it has been merged
# git fetch --prune origin
# if ! git branch --list "$GW_BRANCH" | grep -q .; then
#   exit 0
# fi
# if git merge-base --is-ancestor "$GW_BRANCH" origin/main; then
#   git branch -D "$GW_BRANCH"
# fi
`},
	}

	for _, h := range hooks {
		if err := os.WriteFile(filepath.Join(hooksDir, h.name), []byte(h.content), 0755); err != nil {
			os.RemoveAll(gwDir)
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "Initialized .gw/ in %s\n", repoRoot)
	return nil
}
