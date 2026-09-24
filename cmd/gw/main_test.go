package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/pathutil"
	"github.com/gin0606/gw/internal/testutil"
)

var gwBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gw-test-bin")
	if err != nil {
		panic(err)
	}

	gwBinary = filepath.Join(dir, "gw")
	buildCmd := exec.Command("go", "build", "-o", gwBinary, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic(fmt.Sprintf("build failed: %v\n%s", err, out))
	}

	// Isolate every git the tests spawn (directly, via the gw binary, and
	// via gw's hooks) from the developer's global and system git config.
	// Child processes inherit os.Environ(), so setting it once here covers
	// all of them. Mirrors internal/testutil. Set after the build above so
	// the build itself still sees the developer's config (e.g. safe.directory).
	os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runGw(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(gwBinary, args...)
	cmd.Dir = dir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run gw: %v", err)
		}
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func TestNoArgs(t *testing.T) {
	stdout, _, exitCode := runGw(t, t.TempDir())

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "COMMANDS") {
		t.Errorf("expected help output in stdout, got: %q", stdout)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, _, exitCode := runGw(t, t.TempDir(), "foo")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
}

func TestVersion(t *testing.T) {
	stdout, _, exitCode := runGw(t, t.TempDir(), "--version")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "gw version") {
		t.Errorf("expected version output, got: %q", stdout)
	}
}

func TestInit_Basic(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	stdout, stderr, exitCode := runGw(t, repo.Root, "init")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}

	if stdout != "" {
		t.Errorf("expected empty stdout, got: %q", stdout)
	}

	if !strings.Contains(stderr, "Initialized .gw/ in") {
		t.Errorf("expected initialization message in stderr, got: %q", stderr)
	}

	configPath := filepath.Join(repo.Root, ".gw", "config")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .gw/config: %v", err)
	}
	configContent := string(configBytes)

	repoName := filepath.Base(repo.Root)
	expectedEntry := fmt.Sprintf(`worktrees_dir = "../%s-worktrees"`, repoName)
	if !strings.Contains(configContent, expectedEntry) {
		t.Errorf("config should contain %q, got: %q", expectedEntry, configContent)
	}

	hookPath := filepath.Join(repo.Root, ".gw", "hooks", "post-add")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("failed to stat .gw/hooks/post-add: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("post-add hook should be executable, got mode: %v", info.Mode())
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	if err := os.MkdirAll(filepath.Join(repo.Root, ".gw"), 0755); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode := runGw(t, repo.Root, "init")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, ".gw/ already exists") {
		t.Errorf("expected already exists error, got: %q", stderr)
	}
}

func TestInit_OutsideGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, exitCode := runGw(t, tmpDir, "init")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestInit_ExtraArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "init", "extra")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("expected 'unexpected argument' in stderr, got: %q", stderr)
	}
}

func TestInit_FromWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktree("wt-for-init", "init-branch")

	_, stderr, exitCode := runGw(t, wtPath, "init")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}

	// .gw/ should be created in the main repo root, not in the worktree
	if _, err := os.Stat(filepath.Join(repo.Root, ".gw", "config")); err != nil {
		t.Errorf(".gw/config should exist in repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".gw")); err == nil {
		t.Error(".gw/ should not be created in worktree directory")
	}
}

func TestAdd_NewBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	// Local main is ahead of origin/main; the default start point is origin/main.
	repo.Commit("local only")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature/new")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	if got, want := repo.RevParse("refs/heads/feature/new"), repo.RevParse("origin/main"); got != want {
		t.Errorf("new branch = %s, want origin/main (%s)", got, want)
	}

	outputPath := strings.TrimSpace(stdout)
	if !strings.HasSuffix(outputPath, "feature-new") {
		t.Errorf("expected path ending with 'feature-new', got: %q", outputPath)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", outputPath)
	}
}

func TestAdd_NewBranch_WithFrom(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature/from-tag", "--from", "main")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	outputPath := strings.TrimSpace(stdout)
	if !strings.HasSuffix(outputPath, "feature-from-tag") {
		t.Errorf("expected path ending with 'feature-from-tag', got: %q", outputPath)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", outputPath)
	}
}

func TestAdd_NewBranch_NoRemoteDefaultBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.Commit("local only")
	// Delete remote tracking branch but keep origin/HEAD
	repo.DeleteRemoteRef("origin/main")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature/fallback")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	if got, want := repo.RevParse("refs/heads/feature/fallback"), repo.RevParse("refs/heads/main"); got != want {
		t.Errorf("new branch = %s, want local main (%s)", got, want)
	}

	outputPath := strings.TrimSpace(stdout)
	if !strings.HasSuffix(outputPath, "feature-fallback") {
		t.Errorf("expected path ending with 'feature-fallback', got: %q", outputPath)
	}
}

func TestAdd_ExistingBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("existing-branch")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "existing-branch")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	outputPath := strings.TrimSpace(stdout)
	if !strings.HasSuffix(outputPath, "existing-branch") {
		t.Errorf("expected path ending with 'existing-branch', got: %q", outputPath)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", outputPath)
	}
}

func TestAdd_ExistingBranch_WithFrom_Error(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("existing-branch")

	_, stderr, exitCode := runGw(t, repo.Root, "add", "existing-branch", "--from", "main")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("expected 'already exists' in stderr, got: %q", stderr)
	}
}

// A precondition failure must not leave the base directory behind.
func TestAdd_PreconditionFailure_NoBaseDirCreated(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("existing-no-basedir")

	_, _, exitCode := runGw(t, repo.Root, "add", "existing-no-basedir", "--from", "main")
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	baseDir := filepath.Join(filepath.Dir(repo.Root), filepath.Base(repo.Root)+"-worktrees")
	if _, err := os.Stat(baseDir); !os.IsNotExist(err) {
		t.Errorf("base directory should not have been created: %s (stat err: %v)", baseDir, err)
	}
}

func TestAdd_NoArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "add")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "branch name required") {
		t.Errorf("expected 'branch name required' in stderr, got: %q", stderr)
	}
}

func TestAdd_ExtraArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "add", "feature/foo", "extra")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("expected 'unexpected argument' in stderr, got: %q", stderr)
	}
}

func TestAdd_OutsideGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, exitCode := runGw(t, tmpDir, "add", "feature/test")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestAdd_FromWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktree("existing-wt", "wt-branch")

	stdout, _, exitCode := runGw(t, wtPath, "add", "feature/from-wt")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	outputPath := strings.TrimSpace(stdout)
	if !strings.HasSuffix(outputPath, "feature-from-wt") {
		t.Errorf("expected path ending with 'feature-from-wt', got: %q", outputPath)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", outputPath)
	}
}

func TestAdd_PreAddHook_Failure(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("pre-add", "#!/bin/sh\nexit 1\n")

	_, _, exitCode := runGw(t, repo.Root, "add", "feature/hook-fail")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	// Worktree should NOT have been created
	repoName := filepath.Base(repo.Root)
	wtPath := filepath.Join(filepath.Dir(repo.Root), repoName+"-worktrees", "feature-hook-fail")
	if _, err := os.Stat(wtPath); err == nil {
		t.Error("worktree should not have been created when pre-add hook fails")
	}
}

func TestAdd_PostAddHook_Failure(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("post-add", "#!/bin/sh\nexit 1\n")

	stdout, stderr, exitCode := runGw(t, repo.Root, "add", "feature/post-fail")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stderr, "gw: warning:") {
		t.Errorf("expected warning in stderr, got: %q", stderr)
	}

	// Worktree should still have been created
	outputPath := strings.TrimSpace(stdout)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("worktree should have been created even when post-add hook fails")
	}
}

func TestAdd_DirectoryCollision(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create the directory that would be used by the worktree
	repoName := filepath.Base(repo.Root)
	baseDir := filepath.Join(filepath.Dir(repo.Root), repoName+"-worktrees")
	collisionDir := filepath.Join(baseDir, "feature-collision")
	if err := os.MkdirAll(collisionDir, 0755); err != nil {
		t.Fatal(err)
	}

	_, _, exitCode := runGw(t, repo.Root, "add", "feature/collision")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestAdd_BrokenSymlinkCollision(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	markerFile := filepath.Join(t.TempDir(), "hook-ran.txt")
	repo.WriteHook("pre-add", "#!/bin/sh\ntouch "+markerFile+"\n")

	repoName := filepath.Base(repo.Root)
	baseDir := filepath.Join(filepath.Dir(repo.Root), repoName+"-worktrees")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(baseDir, "feature-dangling")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), link); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode := runGw(t, repo.Root, "add", "feature/dangling")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "path already exists") {
		t.Errorf("expected 'path already exists' in stderr, got: %q", stderr)
	}
	if _, err := os.Stat(markerFile); err == nil {
		t.Error("pre-add hook should not have been executed when the path is a broken symlink")
	}
	gitCmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/feature/dangling")
	gitCmd.Dir = repo.Root
	if err := gitCmd.Run(); err == nil {
		t.Error("branch should not have been created")
	}
}

func TestAdd_OriginHeadNotSet_NewBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.DeleteOriginHead()

	markerFile := filepath.Join(t.TempDir(), "hook-ran.txt")
	repo.WriteHook("pre-add", "#!/bin/sh\ntouch "+markerFile+"\n")

	_, _, exitCode := runGw(t, repo.Root, "add", "feature/no-origin")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	if _, err := os.Stat(markerFile); err == nil {
		t.Error("pre-add hook should not have been executed when origin/HEAD is not set")
	}
}

func TestAdd_OriginHeadNotSet_ExistingBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.DeleteOriginHead()
	repo.CreateBranch("existing-no-origin")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "existing-no-origin")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	outputPath := strings.TrimSpace(stdout)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", outputPath)
	}
}

func TestAdd_BranchAlreadyCheckedOut(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	// "main" is already checked out in the main repo
	_, _, exitCode := runGw(t, repo.Root, "add", "main")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestAdd_ExistingBranch_WithFrom_PreAddNotExecuted(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("existing-hook-test")

	markerFile := filepath.Join(t.TempDir(), "hook-ran.txt")
	repo.WriteHook("pre-add", "#!/bin/sh\ntouch "+markerFile+"\n")

	_, _, exitCode := runGw(t, repo.Root, "add", "existing-hook-test", "--from", "main")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	if _, err := os.Stat(markerFile); err == nil {
		t.Error("pre-add hook should not have been executed")
	}
}

// symlinkedWorktreesDir configures worktrees_dir as an absolute path through
// a symlink (link -> real) and returns the configured path and the path git
// is expected to register for it.
func symlinkedWorktreesDir(t *testing.T, repo *testutil.TestRepo) (configured, resolved string) {
	t.Helper()
	// t.TempDir() is deliberately left unresolved: on macOS it is itself
	// under a symlink (/var -> /private/var).
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	realResolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	configured = filepath.Join(dir, "link", "trees")
	repo.WriteConfig(fmt.Sprintf("worktrees_dir = %q\n", configured))
	return configured, filepath.Join(realResolved, "trees")
}

// writePathRecordingHooks installs hooks that write GW_WORKTREE_PATH to
// <dir>/<hook name> and returns a reader for those recordings.
func writePathRecordingHooks(t *testing.T, repo *testutil.TestRepo, names ...string) func(name string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		repo.WriteHook(name, "#!/bin/sh\nprintf '%s' \"$GW_WORKTREE_PATH\" > '"+filepath.Join(dir, name)+"'\n")
	}
	return func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s recording: %v", name, err)
		}
		return string(b)
	}
}

// listedWorktrees returns the non-main worktree paths printed by `gw list`.
func listedWorktrees(t *testing.T, repo *testutil.TestRepo) []string {
	t.Helper()
	stdout, _, exitCode := runGw(t, repo.Root, "list")
	if exitCode != 0 {
		t.Fatalf("gw list exit code = %d, want 0", exitCode)
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line != repo.Root {
			paths = append(paths, line)
		}
	}
	return paths
}

func TestAdd_WorktreesDirThroughSymlink_PathMatchesGit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	_, resolvedBase := symlinkedWorktreesDir(t, repo)
	recorded := writePathRecordingHooks(t, repo, "pre-add", "post-add")

	stdout, stderr, exitCode := runGw(t, repo.Root, "add", "feature/symlinked")
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}

	listed := listedWorktrees(t, repo)
	if len(listed) != 1 {
		t.Fatalf("expected one worktree in gw list, got: %q", listed)
	}
	want := listed[0]
	if want != filepath.Join(resolvedBase, "feature-symlinked") {
		t.Errorf("gw list path = %q, want symlink-resolved %q", want, filepath.Join(resolvedBase, "feature-symlinked"))
	}
	if got := strings.TrimSpace(stdout); got != want {
		t.Errorf("gw add stdout = %q, want %q", got, want)
	}
	if got := recorded("pre-add"); got != want {
		t.Errorf("pre-add GW_WORKTREE_PATH = %q, want %q", got, want)
	}
	if got := recorded("post-add"); got != want {
		t.Errorf("post-add GW_WORKTREE_PATH = %q, want %q", got, want)
	}
}

// assertAddRejectedBeforeHook runs "gw add" with a pre-add hook installed and
// checks that it fails without running the hook, creating the worktree, or
// creating the branch, and that stderr mentions wantInStderr.
func assertAddRejectedBeforeHook(t *testing.T, repo *testutil.TestRepo, branch, wantInStderr string, args ...string) {
	t.Helper()

	markerFile := filepath.Join(t.TempDir(), "hook-ran.txt")
	repo.WriteHook("pre-add", "#!/bin/sh\ntouch "+markerFile+"\n")

	_, stderr, exitCode := runGw(t, repo.Root, append([]string{"add", branch}, args...)...)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, wantInStderr) {
		t.Errorf("expected %q in stderr, got: %q", wantInStderr, stderr)
	}
	if _, err := os.Stat(markerFile); err == nil {
		t.Error("pre-add hook should not have been executed")
	}

	baseDir := pathutil.BaseDir(repo.Root, filepath.Base(repo.Root), "")
	wtPath, err := pathutil.ComputePath(baseDir, branch)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtPath); err == nil {
		t.Errorf("worktree directory should not have been created: %s", wtPath)
	}
	if _, err := os.Stat(baseDir); err == nil {
		t.Errorf("base directory should not have been created: %s", baseDir)
	}

	exists, err := git.BranchExists(repo.Root, branch)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Errorf("branch %q should not have been created", branch)
	}
}

func TestAdd_NewBranch_FromUnresolvable_PreAddNotExecuted(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	assertAddRejectedBeforeHook(t, repo, "feature/bad-from", "start point 'nonexistent-ref' is not a valid commit", "--from", "nonexistent-ref")
}

func TestAdd_NewBranch_DefaultBranchUnresolvable_PreAddNotExecuted(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	// origin/HEAD is set, but neither origin/develop nor a local develop exists.
	repo.SetOriginHead("develop")

	assertAddRejectedBeforeHook(t, repo, "feature/no-default", "default start point 'develop' (from origin/HEAD) is not a valid commit; pass --from <ref> to choose one")
}

func TestAdd_NewBranch_FromRemoteRef(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.Commit("remote-only commit")
	repo.CreateBranch("remote-only")
	repo.PushBranch("remote-only")
	repo.DeleteBranch("remote-only")
	want := repo.RevParse("origin/remote-only")
	if want == repo.RevParse("origin/main") {
		t.Fatal("test setup: origin/remote-only must differ from origin/main")
	}

	stdout, stderr, exitCode := runGw(t, repo.Root, "add", "feature/from-remote", "--from", "origin/remote-only")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if _, err := os.Stat(strings.TrimSpace(stdout)); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", stdout)
	}
	if got := repo.RevParse("refs/heads/feature/from-remote"); got != want {
		t.Errorf("new branch = %s, want %s", got, want)
	}
	// Upstream tracking is set only when git receives the remote ref name.
	if got := repo.ConfigValue("branch.feature/from-remote.remote"); got != "origin" {
		t.Errorf("branch.<new>.remote = %q, want %q", got, "origin")
	}
	if got := repo.ConfigValue("branch.feature/from-remote.merge"); got != "refs/heads/remote-only" {
		t.Errorf("branch.<new>.merge = %q, want %q", got, "refs/heads/remote-only")
	}
}

func TestAdd_NewBranch_FromCommitHash(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.Commit("local only")
	head := repo.RevParse("HEAD")
	if head == repo.RevParse("origin/main") {
		t.Fatal("test setup: HEAD must differ from origin/main")
	}

	stdout, stderr, exitCode := runGw(t, repo.Root, "add", "feature/from-hash", "--from", head)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if _, err := os.Stat(strings.TrimSpace(stdout)); os.IsNotExist(err) {
		t.Errorf("worktree directory was not created: %s", stdout)
	}
	if got := repo.RevParse("refs/heads/feature/from-hash"); got != head {
		t.Errorf("new branch = %s, want %s", got, head)
	}
}

func TestAdd_NewBranch_FromDash(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("prev")
	repo.Checkout("prev")
	repo.Commit("prev commit")
	repo.Checkout("main")
	want := repo.RevParse("refs/heads/prev")
	if want == repo.RevParse("refs/heads/main") {
		t.Fatal("test setup: prev must differ from main")
	}

	_, stderr, exitCode := runGw(t, repo.Root, "add", "feature/from-dash", "--from", "-")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if got := repo.RevParse("refs/heads/feature/from-dash"); got != want {
		t.Errorf("new branch = %s, want prev (%s)", got, want)
	}
}

func TestList_Basic(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/list-test")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "list")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, wtPath) {
		t.Errorf("expected list to contain %q, got: %q", wtPath, stdout)
	}

	if !strings.Contains(stdout, repo.Root) {
		t.Errorf("expected list to contain repo root %q, got: %q", repo.Root, stdout)
	}
}

func TestList_NoWorktrees(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	stdout, _, exitCode := runGw(t, repo.Root, "list")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Should contain at least the main repo
	if !strings.Contains(stdout, repo.Root) {
		t.Errorf("expected list to contain repo root %q, got: %q", repo.Root, stdout)
	}
}

func TestList_MultipleWorktrees(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout1, _, exitCode := runGw(t, repo.Root, "add", "feature/list-one")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wt1 := strings.TrimSpace(addStdout1)

	addStdout2, _, exitCode := runGw(t, repo.Root, "add", "feature/list-two")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wt2 := strings.TrimSpace(addStdout2)

	stdout, _, exitCode := runGw(t, repo.Root, "list")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, wt1) {
		t.Errorf("expected list to contain %q, got: %q", wt1, stdout)
	}
	if !strings.Contains(stdout, wt2) {
		t.Errorf("expected list to contain %q, got: %q", wt2, stdout)
	}
}

func TestList_FromWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/list-from-wt")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Run list from inside the worktree
	stdout, _, exitCode := runGw(t, wtPath, "list")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, repo.Root) {
		t.Errorf("expected list to contain repo root %q, got: %q", repo.Root, stdout)
	}
	if !strings.Contains(stdout, wtPath) {
		t.Errorf("expected list to contain worktree %q, got: %q", wtPath, stdout)
	}
}

func TestList_ExtraArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "list", "extra")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("expected 'unexpected argument' in stderr, got: %q", stderr)
	}
}

func TestList_OutsideGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, exitCode := runGw(t, tmpDir, "list")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestRm_Basic(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/rm-test")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "rm", wtPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got: %q", stdout)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

// The worktree is created with git directly because a branch name cannot
// carry a newline into the path.
func TestRm_PathWithNewline(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktree("wt\nnewline", "feature/newline")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", wtPath)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}

	gitList := exec.Command("git", "worktree", "list", "--porcelain", "-z")
	gitList.Dir = repo.Root
	out, err := gitList.Output()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("worktree "+wtPath+"\x00")) {
		t.Errorf("worktree %q still listed by git: %q", wtPath, out)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %q", wtPath)
	}
}

func TestRm_UncommittedChanges_WithoutForce(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/dirty-no-force")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Create uncommitted changes in the worktree
	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", "dirty.txt")
	gitAdd.Dir = wtPath
	if err := gitAdd.Run(); err != nil {
		t.Fatal(err)
	}

	// Remove without --force should fail
	_, _, exitCode = runGw(t, repo.Root, "rm", wtPath)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	// Worktree should still exist
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree should not have been removed without --force")
	}
}

func TestRm_Force_UncommittedChanges(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/force-rm")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Create uncommitted changes in the worktree
	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", "dirty.txt")
	gitAdd.Dir = wtPath
	if err := gitAdd.Run(); err != nil {
		t.Fatal(err)
	}

	_, _, exitCode = runGw(t, repo.Root, "rm", wtPath, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

func TestRm_PreRemoveHook_Failure_NoForce(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/hook-rm")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Set up failing pre-remove hook
	repo.WriteHook("pre-remove", "#!/bin/sh\nexit 1\n")

	_, _, exitCode = runGw(t, repo.Root, "rm", wtPath)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}

	// Worktree should still exist
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree should not have been removed when pre-remove hook fails without --force")
	}
}

func TestRm_PreRemoveHook_Failure_WithForce(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/hook-force-rm")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Set up failing pre-remove hook
	repo.WriteHook("pre-remove", "#!/bin/sh\nexit 1\n")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", wtPath, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stderr, "gw: warning:") {
		t.Errorf("expected warning in stderr, got: %q", stderr)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

func TestRm_PostRemoveHook_Failure(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/post-rm")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Set up failing post-remove hook
	repo.WriteHook("post-remove", "#!/bin/sh\nexit 1\n")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", wtPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stderr, "gw: warning:") {
		t.Errorf("expected warning in stderr, got: %q", stderr)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

func TestRm_NotFound(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a subdirectory that is inside the repo but is not a worktree
	subdir := filepath.Join(repo.Root, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode := runGw(t, repo.Root, "rm", subdir)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "not a git worktree") {
		t.Errorf("expected 'not a git worktree' in stderr, got: %q", stderr)
	}
}

// A non-existent path under an existing directory resolves via its existing
// ancestor, so a typo in the path argument is reported as a plain
// missing-worktree error rather than a symlink-resolution failure.
func TestRm_NotFound_MissingPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "rm", filepath.Join(repo.Root, "ghost"))

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "is not a git worktree") {
		t.Errorf("expected 'is not a git worktree' in stderr, got: %q", stderr)
	}
	if strings.Contains(stderr, "resolve symlinks") {
		t.Errorf("missing path should not be reported as symlink-resolution failure: %q", stderr)
	}
}

// filepath.EvalSymlinks walks each path component, so a child under a regular
// file fails with ENOTDIR (not fs.ErrNotExist). For a path that is not
// registered as a worktree, that failure must surface as a symlink-resolution
// error rather than be conflated with "not a git worktree".
func TestRm_SymlinkResolutionError_NotMistakenForMissing(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	notDir := filepath.Join(repo.Root, "not-a-dir")
	if err := os.WriteFile(notDir, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode := runGw(t, repo.Root, "rm", filepath.Join(notDir, "child"))

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if strings.Contains(stderr, "is not a git worktree") {
		t.Errorf("symlink-resolution failure should not be reported as 'is not a git worktree': %q", stderr)
	}
	if !strings.Contains(stderr, "resolve symlinks") {
		t.Errorf("expected 'resolve symlinks' in stderr, got: %q", stderr)
	}
	// The offending path must propagate through the wrapped fs.PathError so
	// users can see which component failed.
	if !strings.Contains(stderr, notDir) {
		t.Errorf("expected wrapped path %q in stderr, got: %q", notDir, stderr)
	}
}

func TestRm_BranchSurvives(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/branch-survives")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	_, _, exitCode = runGw(t, repo.Root, "rm", wtPath)
	if exitCode != 0 {
		t.Fatalf("gw rm exit code = %d, want 0", exitCode)
	}

	// Branch should still exist
	gitCmd := exec.Command("git", "rev-parse", "--verify", "feature/branch-survives")
	gitCmd.Dir = repo.Root
	if err := gitCmd.Run(); err != nil {
		t.Error("branch should still exist after worktree removal")
	}
}

func TestRm_EmptyStdout(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/empty-stdout")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "rm", wtPath)
	if exitCode != 0 {
		t.Fatalf("gw rm exit code = %d, want 0", exitCode)
	}

	if stdout != "" {
		t.Errorf("expected empty stdout, got: %q", stdout)
	}
}

func TestRm_NoArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "rm")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "path required") {
		t.Errorf("expected 'path required' in stderr, got: %q", stderr)
	}
}

func TestRm_MainWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "rm", repo.Root)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "cannot remove the main worktree") {
		t.Errorf("expected 'cannot remove the main worktree' in stderr, got: %q", stderr)
	}
}

func TestRm_OutsideGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, exitCode := runGw(t, tmpDir, "rm", "/nonexistent/path")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}

func TestRm_ExtraArgs(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "rm", "feature/foo", "extra")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("expected 'unexpected argument' in stderr, got: %q", stderr)
	}
}

func TestRm_ForceFlagBeforePath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/flag-order")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Remove with --force before path
	_, _, exitCode = runGw(t, repo.Root, "rm", "--force", wtPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

func TestRm_FromWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout1, _, exitCode := runGw(t, repo.Root, "add", "feature/rm-source")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	sourcePath := strings.TrimSpace(addStdout1)

	addStdout2, _, exitCode := runGw(t, repo.Root, "add", "feature/rm-target")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	targetPath := strings.TrimSpace(addStdout2)

	// Remove the target from inside the source worktree
	_, _, exitCode = runGw(t, sourcePath, "rm", targetPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", targetPath)
	}
}

func TestRm_RelativePath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/rel-path")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	relPath, err := filepath.Rel(repo.Root, wtPath)
	if err != nil {
		t.Fatal(err)
	}

	// Remove using relative path
	_, _, exitCode = runGw(t, repo.Root, "rm", relPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory should have been removed: %s", wtPath)
	}
}

func TestRm_StaleWorktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/stale")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	// Manually delete the worktree directory to simulate a stale entry
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatal(err)
	}

	// Remove the stale worktree with --force from repo root
	_, _, exitCode = runGw(t, repo.Root, "rm", "--force", wtPath)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

// When a registered worktree's parent directory has been replaced by a
// regular file, filepath.EvalSymlinks fails with ENOTDIR. Because the path
// still lives in git's worktree metadata, `gw rm --force` (force so the
// underlying `git worktree remove --force` tolerates the broken checkout)
// must reach the cleanup path instead of bailing out on the resolution error.
func TestRm_StaleWorktree_ParentReplacedWithFile(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/stale-parent")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	parent := filepath.Dir(wtPath)
	if err := os.RemoveAll(parent); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, exitCode = runGw(t, repo.Root, "rm", "--force", wtPath)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	listStdout, _, exitCode := runGw(t, repo.Root, "list")
	if exitCode != 0 {
		t.Fatalf("gw list exit code = %d, want 0", exitCode)
	}
	if strings.Contains(listStdout, wtPath) {
		t.Errorf("worktree %q should be cleaned from git metadata, got: %q", wtPath, listStdout)
	}
}

func TestRm_WorktreesDirThroughSymlink(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	configuredBase, _ := symlinkedWorktreesDir(t, repo)

	if _, stderr, exitCode := runGw(t, repo.Root, "add", "feature/symlinked-rm"); exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	listed := listedWorktrees(t, repo)
	if len(listed) != 1 {
		t.Fatalf("expected one worktree in gw list, got: %q", listed)
	}
	want := listed[0]
	recorded := writePathRecordingHooks(t, repo, "pre-remove", "post-remove")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", filepath.Join(configuredBase, "feature-symlinked-rm"))
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if listed := listedWorktrees(t, repo); len(listed) != 0 {
		t.Errorf("worktree should no longer be listed, got: %q", listed)
	}
	if got := recorded("pre-remove"); got != want {
		t.Errorf("pre-remove GW_WORKTREE_PATH = %q, want %q", got, want)
	}
	if got := recorded("post-remove"); got != want {
		t.Errorf("post-remove GW_WORKTREE_PATH = %q, want %q", got, want)
	}
}

// With the worktree directory gone, the path cannot be resolved as a whole;
// its existing ancestor (through the symlink) must still be resolved so the
// argument matches the path git registered.
func TestRm_StaleWorktree_WorktreesDirThroughSymlink(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	configuredBase, _ := symlinkedWorktreesDir(t, repo)

	if _, stderr, exitCode := runGw(t, repo.Root, "add", "feature/symlinked-stale"); exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	listed := listedWorktrees(t, repo)
	if len(listed) != 1 {
		t.Fatalf("expected one worktree in gw list, got: %q", listed)
	}
	want := listed[0]
	if err := os.RemoveAll(want); err != nil {
		t.Fatal(err)
	}
	// pre-remove runs inside the (now missing) worktree, so it cannot start;
	// --force downgrades that to a warning. Only post-remove is checked.
	recorded := writePathRecordingHooks(t, repo, "post-remove")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", "--force", filepath.Join(configuredBase, "feature-symlinked-stale"))
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if listed := listedWorktrees(t, repo); len(listed) != 0 {
		t.Errorf("worktree should no longer be listed, got: %q", listed)
	}
	if got := recorded("post-remove"); got != want {
		t.Errorf("post-remove GW_WORKTREE_PATH = %q, want %q", got, want)
	}
}

// A worktree registered before its base directory was moved behind a symlink
// resolves to a different path than git registered; the unresolved argument
// must still match git's registration.
func TestRm_BaseDirReplacedWithSymlink_RegisteredPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	if _, stderr, exitCode := runGw(t, repo.Root, "add", "feature/moved-base"); exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	listed := listedWorktrees(t, repo)
	if len(listed) != 1 {
		t.Fatalf("expected one worktree in gw list, got: %q", listed)
	}
	registered := listed[0]

	baseDir := filepath.Dir(registered)
	moved := baseDir + "-moved"
	if err := os.Rename(baseDir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, baseDir); err != nil {
		t.Fatal(err)
	}
	recorded := writePathRecordingHooks(t, repo, "pre-remove", "post-remove")

	_, stderr, exitCode := runGw(t, repo.Root, "rm", registered)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", exitCode, stderr)
	}
	if listed := listedWorktrees(t, repo); len(listed) != 0 {
		t.Errorf("worktree should no longer be listed, got: %q", listed)
	}
	if got := recorded("pre-remove"); got != registered {
		t.Errorf("pre-remove GW_WORKTREE_PATH = %q, want %q", got, registered)
	}
	if got := recorded("post-remove"); got != registered {
		t.Errorf("post-remove GW_WORKTREE_PATH = %q, want %q", got, registered)
	}
}

func TestRm_ForceOnly_NoPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	_, stderr, exitCode := runGw(t, repo.Root, "rm", "--force")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "path required") {
		t.Errorf("expected 'path required' in stderr, got: %q", stderr)
	}
}

func TestCompletion_Add_Branches(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-complete")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "feature-complete") {
		t.Errorf("expected 'feature-complete' in completion output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "main") {
		t.Errorf("expected 'main' in completion output, got: %q", stdout)
	}
}

func TestCompletion_Add_FromRefs(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-ref")
	repo.PushBranch("feature-ref")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "--from", "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "origin/main") {
		t.Errorf("expected 'origin/main' in completion output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "feature-ref") {
		t.Errorf("expected 'feature-ref' in completion output, got: %q", stdout)
	}
}

func TestCompletion_Add_NoCompletionAfterBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-done")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature-done", "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	// Should not list branches since positional arg is already provided
	if strings.Contains(stdout, "main") {
		t.Errorf("expected no branch completions after positional arg, got: %q", stdout)
	}
}

func TestCompletion_Rm_Worktrees(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/rm-complete")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "rm", "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, wtPath) {
		t.Errorf("expected worktree path %q in completion output, got: %q", wtPath, stdout)
	}
	// Should not contain main worktree as its own line
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line == repo.Root {
			t.Errorf("expected main worktree NOT in completion output, got: %q", stdout)
		}
	}
}

func TestCompletion_Rm_NoCompletionAfterPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/rm-nocomp")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "rm", wtPath, "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	// Should not list worktree paths since positional arg is already provided
	if strings.Contains(stdout, "worktrees") {
		t.Errorf("expected no worktree completions after positional arg, got: %q", stdout)
	}
}

func TestCompletion_Rm_AfterForceFlag(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	addStdout, _, exitCode := runGw(t, repo.Root, "add", "feature/force-comp")
	if exitCode != 0 {
		t.Fatalf("gw add exit code = %d, want 0", exitCode)
	}
	wtPath := strings.TrimSpace(addStdout)

	stdout, _, exitCode := runGw(t, repo.Root, "rm", "--force", "--generate-shell-completion")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	// After --force, should complete worktree paths (not flags)
	if !strings.Contains(stdout, wtPath) {
		t.Errorf("expected worktree path %q in completion output after --force, got: %q", wtPath, stdout)
	}
}

func TestCompletion_CompletionSubcommand(t *testing.T) {
	stdout, _, exitCode := runGw(t, t.TempDir(), "completion", "bash")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "compgen") && !strings.Contains(stdout, "complete") {
		t.Errorf("expected bash completion script, got: %q", stdout)
	}
}
