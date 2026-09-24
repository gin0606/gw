package pathutil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/gin0606/gw/internal/pathutil"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/user-auth", "feature-user-auth"},
		{"feature/auth/login", "feature-auth-login"},
		{"simple", "simple"},
		{"a/b/c/d", "a-b-c-d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := pathutil.Sanitize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_TrimHyphens(t *testing.T) {
	got, err := pathutil.Sanitize("/feature/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature" {
		t.Errorf("got %q, want %q", got, "feature")
	}
}

func TestSanitize_EmptyResult(t *testing.T) {
	_, err := pathutil.Sanitize("/")
	if err == nil {
		t.Error("expected error for branch that sanitizes to empty")
	}
}

func TestSanitize_Dot(t *testing.T) {
	_, err := pathutil.Sanitize(".")
	if err == nil {
		t.Error("expected error for branch '.'")
	}
}

func TestSanitize_DotDot(t *testing.T) {
	_, err := pathutil.Sanitize("..")
	if err == nil {
		t.Error("expected error for branch '..'")
	}
}

func TestBaseDir_Default(t *testing.T) {
	got := pathutil.BaseDir("/home/user/repo", "repo", "")
	want := filepath.Join("/home/user/repo", "..", "repo-worktrees")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBaseDir_Relative(t *testing.T) {
	got := pathutil.BaseDir("/home/user/repo", "repo", "../my-trees")
	want := filepath.Join("/home/user/repo", "../my-trees")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBaseDir_Absolute(t *testing.T) {
	got := pathutil.BaseDir("/home/user/repo", "repo", "/tmp/trees")
	if got != "/tmp/trees" {
		t.Errorf("got %q, want %q", got, "/tmp/trees")
	}
}

func TestComputePath(t *testing.T) {
	got, err := pathutil.ComputePath("/base", "feature/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/base", "feature-foo")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidatePath_Exists(t *testing.T) {
	dir := t.TempDir()
	err := pathutil.ValidatePath(dir)
	if err == nil {
		t.Error("expected error for existing directory")
	}
}

func TestValidatePath_BrokenSymlink(t *testing.T) {
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), link); err != nil {
		t.Fatal(err)
	}
	err := pathutil.ValidatePath(link)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error for broken symlink, got %v", err)
	}
}

func TestValidatePath_SymlinkToDir(t *testing.T) {
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	err := pathutil.ValidatePath(link)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error for symlink to existing directory, got %v", err)
	}
}

func TestValidatePath_RegularFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	err := pathutil.ValidatePath(file)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error for regular file, got %v", err)
	}
}

func TestValidatePath_NotExists(t *testing.T) {
	err := pathutil.ValidatePath("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Lstat'ing a path under a regular file fails with ENOTDIR, which is the
// canonical non-fs.ErrNotExist failure we must not silently treat as "absent".
func TestValidatePath_LstatErrorNotMistakenForMissing(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	err := pathutil.ValidatePath(filepath.Join(file, "child"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error should not wrap fs.ErrNotExist, got %v", err)
	}
	if !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("error should wrap ENOTDIR, got %v", err)
	}
}

func TestResolveExistingPrefix(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	realResolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("existing dir through symlink", func(t *testing.T) {
		got, err := pathutil.ResolveExistingPrefix(link)
		if err != nil {
			t.Fatal(err)
		}
		if got != realResolved {
			t.Errorf("got %q, want %q", got, realResolved)
		}
	})

	t.Run("missing tail under symlinked ancestor", func(t *testing.T) {
		got, err := pathutil.ResolveExistingPrefix(filepath.Join(link, "a", "b"))
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(realResolved, "a", "b"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("child of a file", func(t *testing.T) {
		_, err := pathutil.ResolveExistingPrefix(filepath.Join(file, "child"))
		if !errors.Is(err, syscall.ENOTDIR) {
			t.Errorf("err = %v, want ENOTDIR", err)
		}
	})
}
