package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin0606/gw/internal/config"
	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/hook"
	"github.com/gin0606/gw/internal/pathutil"
)

// Add implements the "gw add" command.
func Add(branch, from string, noHooks bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		return err
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		return err
	}

	repoName := git.RepoName(repoRoot)
	baseDir := pathutil.BaseDir(repoRoot, repoName, cfg.WorktreesDir)

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	wtPath, err := pathutil.ComputePath(baseDir, branch)
	if err != nil {
		return err
	}

	if err := pathutil.ValidatePath(wtPath); err != nil {
		return err
	}

	exists, err := git.BranchExists(repoRoot, branch)
	if err != nil {
		return err
	}

	if exists && from != "" {
		return fmt.Errorf("branch '%s' already exists; --from cannot be used", branch)
	}

	var gitArgs []string
	if !exists {
		gitArgs = []string{"worktree", "add", wtPath, "-b", branch}
		if from != "" {
			gitArgs = append(gitArgs, from)
		} else {
			defaultBranch, err := git.DefaultBranch(repoRoot)
			if err != nil {
				return err
			}

			remoteRef := "origin/" + defaultBranch
			remoteExists, err := git.RemoteRefExists(repoRoot, remoteRef)
			if err != nil {
				return err
			}

			if remoteExists {
				gitArgs = append(gitArgs, remoteRef)
			} else {
				gitArgs = append(gitArgs, defaultBranch)
			}
		}
	} else {
		gitArgs = []string{"worktree", "add", wtPath, branch}
	}

	if err := pathutil.EnsureBaseDir(baseDir); err != nil {
		return err
	}

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PreAdd, wtPath, branch, os.Stderr); err != nil {
			return fmt.Errorf("pre-add hook failed: %w", err)
		}
	}

	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = repoRoot
	gitCmd.Stdout = os.Stderr
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PostAdd, wtPath, branch, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "gw: warning: post-add hook failed: %v\n", err)
		}
	}

	fmt.Println(wtPath)

	return nil
}
