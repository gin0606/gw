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
func Add(args []string) int {
	branch, from, err := parseAddArgs(args)
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

	// 2. Calculate worktree path
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

	wtPath, err := pathutil.ComputePath(baseDir, branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	if err := pathutil.ValidatePath(wtPath); err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 3. Check branch existence, validate args, and resolve start-point ref
	exists, err := git.BranchExists(repoRoot, branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	if exists && from != "" {
		fmt.Fprintf(os.Stderr, "gw: error: branch '%s' already exists; --from cannot be used\n", branch)
		return 1
	}

	var gitArgs []string
	if !exists {
		gitArgs = []string{"worktree", "add", wtPath, "-b", branch}
		if from != "" {
			gitArgs = append(gitArgs, from)
		} else {
			defaultBranch, err := git.DefaultBranch(repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
				return 1
			}

			remoteRef := "origin/" + defaultBranch
			remoteExists, err := git.RemoteRefExists(repoRoot, remoteRef)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
				return 1
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

	// Ensure base directory exists
	if err := pathutil.EnsureBaseDir(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 4. Run pre-add hook (at repo root)
	if err := hook.Run(repoRoot, "pre-add", repoRoot, wtPath, branch, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: pre-add hook failed: %v\n", err)
		return 1
	}

	// 5. Create worktree

	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = repoRoot
	gitCmd.Stdout = os.Stderr
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return 1
	}

	// 6. Run post-add hook (in worktree directory)
	if err := hook.Run(repoRoot, "post-add", wtPath, wtPath, branch, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gw: warning: post-add hook failed: %v\n", err)
	}

	// 7. Output path to stdout
	fmt.Println(wtPath)

	return 0
}

func parseAddArgs(args []string) (branch, from string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("branch name required")
	}

	branch = args[0]

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--from requires a value")
			}
			from = args[i+1]
			i++
		default:
			return "", "", fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	return branch, from, nil
}
