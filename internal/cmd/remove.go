package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/hook"
)

// Remove implements the "gw rm" command.
func Remove(path string, force bool) error {
	// 1. Normalize path to absolute and resolve symlinks
	wtPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(wtPath); err == nil {
		wtPath = resolved
	}

	// 2. Detect repo root from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		return err
	}

	// 3. Look up the worktree to get its branch name (needed for hooks)
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return err
	}

	var branch string
	found := false
	for _, wt := range worktrees {
		if wt.Path == wtPath {
			branch = wt.Branch
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("path %q is not a git worktree", wtPath)
	}
	if wtPath == repoRoot {
		return fmt.Errorf("cannot remove the main worktree")
	}

	// Run pre-remove hook (in worktree directory)
	if err := hook.Run(repoRoot, "pre-remove", wtPath, wtPath, branch, os.Stderr); err != nil {
		if !force {
			return fmt.Errorf("pre-remove hook failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "gw: warning: pre-remove hook failed: %v\n", err)
	}

	// 3. Remove worktree
	gitArgs := []string{"worktree", "remove"}
	if force {
		gitArgs = append(gitArgs, "--force")
	}
	gitArgs = append(gitArgs, wtPath)

	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = repoRoot
	gitCmd.Stdout = os.Stderr
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}

	// 4. Run post-remove hook (at repo root)
	if err := hook.Run(repoRoot, "post-remove", repoRoot, wtPath, branch, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gw: warning: post-remove hook failed: %v\n", err)
	}

	return nil
}
