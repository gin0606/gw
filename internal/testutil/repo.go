package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/pathutil"
)

// TestRepo represents a temporary git repository for testing.
type TestRepo struct {
	Root     string // Main repository root (symlink-resolved)
	BareRoot string // Bare repository path (origin)
	t        *testing.T
}

// NewTestRepo creates a temporary git repository with a bare origin remote.
// The repository has an initial commit on "main" and origin/HEAD is set.
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	dir := t.TempDir()
	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	bareDir := filepath.Join(dir, "origin.git")
	gitCmd(t, "", "init", "--bare", "-b", "main", bareDir)

	repoDir := filepath.Join(dir, "repo")
	gitCmd(t, "", "init", "-b", "main", repoDir)
	gitCmd(t, repoDir, "remote", "add", "origin", bareDir)

	if err := os.WriteFile(filepath.Join(repoDir, ".gitkeep"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repoDir, "add", ".")
	gitCmd(t, repoDir, "commit", "-m", "initial")
	gitCmd(t, repoDir, "push", "-u", "origin", "main")
	gitCmd(t, repoDir, "remote", "set-head", "origin", "--auto")

	return &TestRepo{
		Root:     repoDir,
		BareRoot: bareDir,
		t:        t,
	}
}

// CreateBranch creates a local branch pointing at the current HEAD without checking it out.
func (r *TestRepo) CreateBranch(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "branch", name)
}

// PushBranch pushes a branch to origin.
func (r *TestRepo) PushBranch(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "push", "origin", name)
}

// CreateTag creates a lightweight tag at the current HEAD.
func (r *TestRepo) CreateTag(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "tag", name)
}

// UpdateRef points ref at commit, creating the ref if needed. Unlike
// CreateTag, it accepts ref names that git's porcelain would parse as options.
func (r *TestRepo) UpdateRef(ref, commit string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "update-ref", ref, commit)
}

// DeleteOriginHead removes origin/HEAD symbolic ref.
func (r *TestRepo) DeleteOriginHead() {
	r.t.Helper()
	gitCmd(r.t, r.Root, "remote", "set-head", "origin", "--delete")
}

// SetOriginHead points origin/HEAD at origin/<branch>, whether or not that ref exists.
func (r *TestRepo) SetOriginHead(branch string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/"+branch)
}

// RevParse returns the object name that rev resolves to.
func (r *TestRepo) RevParse(rev string) string {
	r.t.Helper()
	return gitCmd(r.t, r.Root, "rev-parse", "--verify", rev)
}

// Commit creates an empty commit on the currently checked-out branch.
func (r *TestRepo) Commit(msg string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "commit", "--allow-empty", "-m", msg)
}

// Checkout checks out a branch in the main worktree.
func (r *TestRepo) Checkout(branch string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "checkout", "-q", branch)
}

// DeleteBranch force-deletes a local branch.
func (r *TestRepo) DeleteBranch(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "branch", "-D", name)
}

// SymbolicHead returns the ref that HEAD of the main worktree points at.
func (r *TestRepo) SymbolicHead() string {
	r.t.Helper()
	return gitCmd(r.t, r.Root, "symbolic-ref", "HEAD")
}

// ConfigValue returns the value of a git config key; the key must be set.
func (r *TestRepo) ConfigValue(key string) string {
	r.t.Helper()
	return gitCmd(r.t, r.Root, "config", "--get", key)
}

// DeleteRemoteRef deletes a remote tracking ref (e.g., "origin/main").
func (r *TestRepo) DeleteRemoteRef(ref string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "update-ref", "-d", "refs/remotes/"+ref)
}

// CreateWorktree creates a git worktree with a new branch and returns its absolute path.
func (r *TestRepo) CreateWorktree(name, branch string) string {
	r.t.Helper()
	wtPath := filepath.Join(filepath.Dir(r.Root), name)
	gitCmd(r.t, r.Root, "worktree", "add", wtPath, "-b", branch)
	return wtPath
}

// CreateDetachedWorktree creates a git worktree with a detached HEAD and returns its absolute path.
func (r *TestRepo) CreateDetachedWorktree(name string) string {
	r.t.Helper()
	wtPath := filepath.Join(filepath.Dir(r.Root), name)
	gitCmd(r.t, r.Root, "worktree", "add", "--detach", wtPath)
	return wtPath
}

// WriteConfig writes .gw/config with the given TOML content.
func (r *TestRepo) WriteConfig(content string) {
	r.t.Helper()
	configDir := filepath.Join(r.Root, ".gw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

// WriteHook creates a hook script in .gw/hooks/ with execute permission.
func (r *TestRepo) WriteHook(name, content string) {
	r.t.Helper()
	hookDir := filepath.Join(r.Root, ".gw", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, name), []byte(content), 0755); err != nil {
		r.t.Fatal(err)
	}
}

// WriteHookNoExec creates a hook script without execute permission.
func (r *TestRepo) WriteHookNoExec(name, content string) {
	r.t.Helper()
	hookDir := filepath.Join(r.Root, ".gw", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, name), []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

// CreateWorktreeInBaseDir creates a worktree in the default base directory (<repo-name>-worktrees/<sanitized-branch>).
func (r *TestRepo) CreateWorktreeInBaseDir(branch string) string {
	r.t.Helper()
	repoName := filepath.Base(r.Root)
	baseDir := filepath.Join(filepath.Dir(r.Root), repoName+"-worktrees")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	wtPath, err := pathutil.ComputePath(baseDir, branch)
	if err != nil {
		r.t.Fatal(err)
	}
	gitCmd(r.t, r.Root, "worktree", "add", wtPath, "-b", branch)
	return wtPath
}

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		// Isolate from the developer's git config (e.g. tag.gpgSign breaks lightweight tags).
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (dir=%s): %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
