package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// git registers the symlink-resolved path; resolve the base so the
	// pre-add hook sees the path git will record.
	baseDir, err = pathutil.ResolveExistingPrefix(baseDir)
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
		start := from
		if start == "" {
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
				start = remoteRef
			} else {
				start = defaultBranch
			}
		}

		checkRef := start
		if start == "-" {
			// git worktree add treats a "-" start point as @{-1}.
			checkRef = "@{-1}"
		}
		resolves, err := git.ResolvesToCommit(repoRoot, checkRef)
		if err != nil {
			return err
		}
		if !resolves {
			return invalidStartPoint(start, from)
		}

		// git worktree add forwards the start point to "git branch" without a
		// terminator, so a "-"-prefixed ref is passed by its full name. The
		// --end-of-options below only guards git worktree add's own parser.
		if start != "-" && strings.HasPrefix(start, "-") {
			fullName, ok, err := git.SymbolicFullName(repoRoot, start)
			if err != nil {
				return err
			}
			if !ok {
				return invalidStartPoint(start, from)
			}
			start = fullName
		}

		gitArgs = []string{"worktree", "add", wtPath, "-b", branch, "--end-of-options", start}
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

	// Report git's registered path so output and hooks agree with `gw list`.
	wtPath = registeredPath(repoRoot, wtPath, branch)

	if !noHooks {
		if err := hook.Run(repoRoot, hook.PostAdd, wtPath, branch, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "gw: warning: post-add hook failed: %v\n", err)
		}
	}

	fmt.Println(wtPath)

	return nil
}

// registeredPath falls back to wtPath: the worktree already exists, so a
// failed lookup must not turn the command into a failure.
func registeredPath(repoRoot, wtPath, branch string) string {
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return wtPath
	}
	for _, wt := range worktrees {
		if wt.Path == wtPath {
			return wt.Path
		}
	}
	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path
		}
	}
	return wtPath
}

func invalidStartPoint(start, from string) error {
	if from != "" {
		return fmt.Errorf("start point '%s' is not a valid commit", start)
	}
	return fmt.Errorf("default start point '%s' (from origin/HEAD) is not a valid commit; pass --from <ref> to choose one", start)
}
