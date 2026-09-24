package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/hook"
	"github.com/gin0606/gw/internal/pathutil"
)

// Remove deletes the worktree at path via `git worktree remove`, running
// pre/post-remove hooks unless noHooks is set. force passes --force to git
// and downgrades pre-remove hook failures to warnings.
func Remove(path string, force, noHooks bool) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// Resolution failures are deferred: a registered worktree whose on-disk
	// path is broken (parent replaced by a file, etc.) must still match
	// git's metadata via the unresolved path so `gw rm --force` can clean it
	// up. The unresolved path is also compared because git may have
	// registered it before a component was replaced by a symlink.
	resolved, resolveErr := pathutil.ResolveExistingPrefix(absPath)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return err
	}

	var wtPath, branch string
	found := false
	for _, wt := range worktrees {
		if (resolveErr == nil && wt.Path == resolved) || wt.Path == absPath {
			wtPath = wt.Path
			branch = wt.Branch
			found = true
			break
		}
	}
	if !found {
		// A resolution failure (e.g. ENOTDIR on an intermediate component) on
		// an unregistered path is a real I/O problem, not a typo: surface it
		// instead of masking it as "not a git worktree".
		if resolveErr != nil {
			return fmt.Errorf("resolve symlinks for %q: %w", absPath, resolveErr)
		}
		return fmt.Errorf("path %q is not a git worktree", absPath)
	}
	if wtPath == repoRoot {
		return fmt.Errorf("cannot remove the main worktree")
	}

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PreRemove, wtPath, branch, os.Stderr); err != nil {
			if !force {
				return fmt.Errorf("pre-remove hook failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "gw: warning: pre-remove hook failed: %v\n", err)
		}
	}

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

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PostRemove, wtPath, branch, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "gw: warning: post-remove hook failed: %v\n", err)
		}
	}

	return nil
}
