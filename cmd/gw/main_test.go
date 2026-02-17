package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// --- Phase 1: CLI skeleton ---

func TestNoArgs(t *testing.T) {
	_, stderr, exitCode := runGw(t, t.TempDir())

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage in stderr, got: %q", stderr)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, stderr, exitCode := runGw(t, t.TempDir(), "foo")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "gw: error:") {
		t.Errorf("expected 'gw: error:' format in stderr, got: %q", stderr)
	}
}

func TestVersion(t *testing.T) {
	stdout, _, exitCode := runGw(t, t.TempDir(), "version")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "gw version 0.1.0") {
		t.Errorf("expected version output, got: %q", stdout)
	}
}

// --- Phase 5: gw add ---

func TestAdd_NewBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature/new")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
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
	// Delete remote tracking branch but keep origin/HEAD
	repo.DeleteRemoteRef("origin/main")

	stdout, _, exitCode := runGw(t, repo.Root, "add", "feature/fallback")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
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
	if !strings.Contains(stderr, "gw: error:") {
		t.Errorf("expected 'gw: error:' format, got: %q", stderr)
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
