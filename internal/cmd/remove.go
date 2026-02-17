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
	"github.com/gin0606/gw/internal/resolve"
)

// Remove implements the "gw rm" command.
func Remove(args []string) int {
	identifier, force, err := parseRemoveArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
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

	// Load config and resolve base directory
	cfg, err := config.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	repoName := git.RepoName(repoRoot)
	baseDir := pathutil.BaseDir(repoRoot, repoName, cfg.WorktreesDir)

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// Resolve identifier to worktree path and branch
	wtPath, branch, err := resolve.Resolve(repoRoot, baseDir, identifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 2. Run pre-remove hook (in worktree directory)
	if err := hook.Run(repoRoot, "pre-remove", wtPath, wtPath, branch, os.Stderr); err != nil {
		if !force {
			fmt.Fprintf(os.Stderr, "gw: error: pre-remove hook failed: %v\n", err)
			return 1
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
		return 1
	}

	// 4. Run post-remove hook (at repo root)
	if err := hook.Run(repoRoot, "post-remove", repoRoot, wtPath, branch, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gw: warning: post-remove hook failed: %v\n", err)
	}

	return 0
}

func parseRemoveArgs(args []string) (identifier string, force bool, err error) {
	if len(args) == 0 {
		return "", false, fmt.Errorf("identifier required")
	}

	for _, arg := range args {
		switch {
		case arg == "--force":
			force = true
		case identifier == "":
			identifier = arg
		default:
			return "", false, fmt.Errorf("unknown argument: %s", arg)
		}
	}

	if identifier == "" {
		return "", false, fmt.Errorf("identifier required")
	}

	return identifier, force, nil
}
