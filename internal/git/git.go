package git

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns the root directory of the main repository.
// When called from a worktree, it returns the main repository root, not the worktree root.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	gitCommonDir := strings.TrimSpace(string(out))

	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(dir, gitCommonDir)
	}

	gitCommonDir = filepath.Clean(gitCommonDir)

	// Resolve symlinks for consistent path representation
	gitCommonDir, err = filepath.EvalSymlinks(gitCommonDir)
	if err != nil {
		return "", err
	}

	return filepath.Dir(gitCommonDir), nil
}

// DefaultBranch returns the default branch name from origin/HEAD.
func DefaultBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("origin/HEAD is not set; run 'git remote set-head origin --auto'")
	}

	ref := strings.TrimSpace(string(out))
	const prefix = "refs/remotes/origin/"
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("unexpected origin/HEAD format: %s", ref)
	}

	return strings.TrimPrefix(ref, prefix), nil
}

// BranchExists checks if a local branch exists.
func BranchExists(repoRoot, branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoRoot
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, err
}

// RemoteRefExists checks if a remote ref exists (e.g., "origin/main").
func RemoteRefExists(repoRoot, ref string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/"+ref)
	cmd.Dir = repoRoot
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, err
}

// RepoName returns the repository name (basename of the repo root).
func RepoName(repoRoot string) string {
	return filepath.Base(repoRoot)
}

// Worktree represents a git worktree entry.
// Branch is empty for detached HEAD worktrees.
type Worktree struct {
	Path   string
	Branch string
}

// ListWorktrees parses `git worktree list --porcelain` and returns all worktrees.
func ListWorktrees(repoRoot string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}
