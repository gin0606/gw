package pathutil_test

import (
	"path/filepath"
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

func TestValidatePath_NotExists(t *testing.T) {
	err := pathutil.ValidatePath("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
