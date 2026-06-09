package hook_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/hook"
	"github.com/gin0606/gw/internal/testutil"
)

func TestRun_HookNotExists(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err != nil {
		t.Errorf("expected no error for missing hook, got: %v", err)
	}
}

func TestRun_HookExistsExitZero(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("pre-add", "#!/bin/sh\nexit 0\n")

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRun_HookNotExecutable(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHookNoExec("pre-add", "#!/bin/sh\nexit 0\n")

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for non-executable hook")
	}
	// Pin the actionable parts of the error message: the absolute hook path
	// (so the user knows which file to fix) and the chmod hint.
	hookPath := filepath.Join(repo.Root, ".gw", "hooks", "pre-add")
	msg := err.Error()
	if !strings.Contains(msg, hookPath) {
		t.Errorf("error %q should mention hook path %q", msg, hookPath)
	}
	if !strings.Contains(msg, "chmod +x") {
		t.Errorf("error %q should hint at chmod +x", msg)
	}
}

func TestRun_HookNonZeroExit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("pre-add", "#!/bin/sh\nexit 1\n")

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err == nil {
		t.Error("expected error for non-zero exit")
	}
}

func TestRun_EnvironmentVariables(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	outFile := filepath.Join(t.TempDir(), "env.txt")

	repo.WriteHook("pre-add", "#!/bin/sh\n"+
		"echo \"REPO_ROOT=$GW_REPO_ROOT\" >> "+outFile+"\n"+
		"echo \"WORKTREE_PATH=$GW_WORKTREE_PATH\" >> "+outFile+"\n"+
		"echo \"BRANCH=$GW_BRANCH\" >> "+outFile+"\n")

	wtPath := "/expected/worktree/path"
	branch := "feature/test"
	err := hook.Run(repo.Root, hook.PreAdd, wtPath, branch, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "REPO_ROOT="+repo.Root) {
		t.Errorf("GW_REPO_ROOT not set correctly:\n%s", content)
	}
	if !strings.Contains(content, "WORKTREE_PATH="+wtPath) {
		t.Errorf("GW_WORKTREE_PATH not set correctly:\n%s", content)
	}
	if !strings.Contains(content, "BRANCH="+branch) {
		t.Errorf("GW_BRANCH not set correctly:\n%s", content)
	}
}

func TestRun_Cwd(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	outFile := filepath.Join(t.TempDir(), "pwd.txt")

	repo.WriteHook("pre-add", "#!/bin/sh\npwd -P > "+outFile+"\n")

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if got != repo.Root {
		t.Errorf("cwd = %q, want %q", got, repo.Root)
	}
}

func TestRun_StdoutToOutput(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("pre-add", "#!/bin/sh\necho 'hook stdout'\n")

	var buf bytes.Buffer
	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &buf)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "hook stdout") {
		t.Errorf("expected hook stdout in output, got: %q", buf.String())
	}
}

// TestRun_AllSpecEnvVarsArePassed asserts that the set of GW_* environment
// variables the hook actually sees matches hook.EnvVars(). This catches drift
// in either direction: if Run starts exporting a new GW_FOO without listing
// it in EnvVars (so docs and init templates omit it), or if EnvVars adds an
// entry that Run never exports.
func TestRun_AllSpecEnvVarsArePassed(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	outFile := filepath.Join(t.TempDir(), "gwenv.txt")

	repo.WriteHook("pre-add", "#!/bin/sh\nenv | grep '^GW_' | sort > "+outFile+"\n")

	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			t.Errorf("malformed env line: %q", line)
			continue
		}
		seen[line[:eq]] = true
	}

	want := map[string]bool{}
	for _, n := range hook.EnvVars() {
		want[n] = true
	}
	for n := range want {
		if !seen[n] {
			t.Errorf("EnvVars lists %q but Run did not export it", n)
		}
	}
	for n := range seen {
		if !want[n] {
			t.Errorf("Run exported %q but EnvVars does not list it (docs/init template will be stale)", n)
		}
	}
}

func TestRun_StderrToOutput(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.WriteHook("pre-add", "#!/bin/sh\necho 'hook stderr' >&2\n")

	var buf bytes.Buffer
	err := hook.Run(repo.Root, hook.PreAdd, "/some/path", "main", &buf)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "hook stderr") {
		t.Errorf("expected hook stderr in output, got: %q", buf.String())
	}
}
