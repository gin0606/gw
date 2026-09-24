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

// ResolvesToCommit checks if ref resolves to a commit.
func ResolvesToCommit(repoRoot, ref string) (bool, error) {
	// Peel in a separate step: appending ^{commit} to ref itself would
	// change the meaning of revisions such as :/<regex>.
	oid, ok, err := verifyRev(repoRoot, "--end-of-options", ref)
	if err != nil || !ok {
		return false, err
	}
	_, ok, err = verifyRev(repoRoot, oid+"^{commit}")
	return ok, err
}

// verifyRev runs "git rev-parse --verify --quiet" and returns the object id.
func verifyRev(repoRoot string, args ...string) (string, bool, error) {
	cmd := exec.Command("git", append([]string{"rev-parse", "--verify", "--quiet"}, args...)...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", false, nil
	}
	return "", false, err
}

// RepoName returns the basename of repoRoot, used as the repository
// identifier for worktree path computation.
func RepoName(repoRoot string) string {
	return filepath.Base(repoRoot)
}

// Worktree represents a git worktree entry.
// Branch is empty for detached HEAD worktrees.
type Worktree struct {
	Path   string
	Branch string
}

// ListLocalBranches returns the short names of all local branches.
func ListLocalBranches(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}
	return splitLines(string(out)), nil
}

// ListRefs returns the short names of all local branches, remote branches, and tags.
func ListRefs(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads/", "refs/remotes/", "refs/tags/")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list refs: %w", err)
	}
	return splitLines(string(out)), nil
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// ListWorktrees parses `git worktree list --porcelain -z` and returns all worktrees.
// With -z, each attribute is NUL-terminated and each entry ends with an extra
// NUL, so paths containing newlines survive intact. Requires git 2.36+.
func ListWorktrees(repoRoot string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain", "-z")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []Worktree
	var current Worktree

	for _, field := range strings.Split(string(out), "\x00") {
		switch {
		case field == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{}
		case strings.HasPrefix(field, "worktree "):
			current = Worktree{Path: strings.TrimPrefix(field, "worktree ")}
		case strings.HasPrefix(field, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(field, "branch refs/heads/")
		}
	}

	return worktrees, nil
}
