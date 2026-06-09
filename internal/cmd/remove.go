package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/hook"
)

// Remove implements the "gw rm" command.
func Remove(path string, force, noHooks bool) error {
	// EvalSymlinks failures are deferred: a registered worktree whose on-disk
	// path is broken (parent gone, replaced by a file, etc.) must still match
	// git's metadata via the absolute-but-unresolved path so that
	// `gw rm --force` can clean it up.
	wtPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	resolved, evalSymlinksErr := filepath.EvalSymlinks(wtPath)
	if evalSymlinksErr == nil {
		wtPath = resolved
	}

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
		// A non-ENOENT EvalSymlinks failure (e.g. ENOTDIR on an intermediate
		// component) on an unregistered path is a real I/O problem, not a
		// typo: surface it instead of masking it as "not a git worktree".
		if evalSymlinksErr != nil && !errors.Is(evalSymlinksErr, fs.ErrNotExist) {
			return fmt.Errorf("resolve symlinks for %q: %w", wtPath, evalSymlinksErr)
		}
		return fmt.Errorf("path %q is not a git worktree", wtPath)
	}
	if wtPath == repoRoot {
		return fmt.Errorf("cannot remove the main worktree")
	}

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PreRemove, wtPath, wtPath, branch, os.Stderr); err != nil {
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
		if err := hook.Run(repoRoot, hook.PostRemove, repoRoot, wtPath, branch, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "gw: warning: post-remove hook failed: %v\n", err)
		}
	}

	return nil
}
