package help_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/config"
	"github.com/gin0606/gw/internal/help"
	"github.com/gin0606/gw/internal/hook"
)

func TestTopic_KnownNames(t *testing.T) {
	for _, name := range help.TopicNames() {
		body, err := help.Topic(name)
		if err != nil {
			t.Errorf("Topic(%q) = err %v, want body", name, err)
			continue
		}
		if strings.TrimSpace(body) == "" {
			t.Errorf("Topic(%q) returned empty body", name)
		}
	}
}

func TestTopic_UnknownName(t *testing.T) {
	_, err := help.Topic("does-not-exist")
	if err == nil {
		t.Fatal("Topic with unknown name should return an error")
	}
	if !errors.Is(err, help.ErrUnknownTopic) {
		t.Errorf("unknown topic should return ErrUnknownTopic, got: %v", err)
	}
}

func TestHooksTopic_ContainsAllHookIdentifiers(t *testing.T) {
	body, err := help.Topic(help.TopicHooks)
	if err != nil {
		t.Fatal(err)
	}
	assertContainsAll(t, "topic hooks", body, hook.Names())
	assertContainsAll(t, "topic hooks", body, hook.EnvVars())
}

func TestConfigTopic_ContainsAllConfigKeys(t *testing.T) {
	body, err := help.Topic(help.TopicConfig)
	if err != nil {
		t.Fatal(err)
	}
	assertContainsAll(t, "topic config", body, []string{config.KeyWorktreesDir})
}

func TestReadmes_ContainAllIdentifiers(t *testing.T) {
	repoRoot := repoRoot(t)
	for _, name := range []string{"README.md", "README.ja.md"} {
		path := filepath.Join(repoRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		s := string(data)
		assertContainsAll(t, name, s, hook.Names())
		assertContainsAll(t, name, s, hook.EnvVars())
		assertContainsAll(t, name, s, []string{config.KeyWorktreesDir})
	}
}

func TestReadmes_ReferenceCLIHelpTopics(t *testing.T) {
	repoRoot := repoRoot(t)
	wantRefs := []string{"gw help all"}
	for _, topic := range help.TopicNames() {
		wantRefs = append(wantRefs, "gw help "+topic)
	}
	for _, name := range []string{"README.md", "README.ja.md"} {
		path := filepath.Join(repoRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		assertContainsAll(t, name, string(data), wantRefs)
	}
}

func assertContainsAll(t *testing.T, source, body string, idents []string) {
	t.Helper()
	for _, id := range idents {
		if !strings.Contains(body, id) {
			t.Errorf("%s: missing identifier %q", source, id)
		}
	}
}

// repoRoot returns the repository root by deriving it from this test file's
// own location, so it is independent of the caller's working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file is .../internal/help/help_test.go; go up three to repo root.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
