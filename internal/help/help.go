// Package help provides topic-based help content for the gw CLI.
//
// Topics live as plain-text files under topics/ and are embedded at build time.
// The identifiers used at runtime (hook names, env var names, config keys) are
// declared in their owning packages (internal/hook, internal/config); tests in
// this package and in cmd/gw assert that those identifiers appear in the topic
// bodies, preventing identifier drift. The prose around the identifiers (hook
// triggers, working directories, failure modes, etc.) is hand-authored and is
// not protected by drift checks.
package help

import (
	"embed"
	"errors"
	"fmt"
	"slices"
	"strings"
)

//go:embed topics/*.txt
var topicFS embed.FS

// init verifies that every registered topic has a corresponding embedded
// .txt file. The same invariant is enforced at test time by
// TestTopicFS_MatchesTopicNames; this init is defense in depth so a binary
// built without that test passing still fails fast at any `gw` invocation
// rather than only when the missing topic is requested.
func init() {
	for _, name := range TopicNames() {
		if _, err := topicFS.ReadFile("topics/" + name + ".txt"); err != nil {
			panic(fmt.Sprintf("help: missing embedded topic %q: %v", name, err))
		}
	}
}

// Topic names. The display order in `gw help all` follows TopicNames.
const (
	TopicHooks      = "hooks"
	TopicConfig     = "config"
	TopicCompletion = "completion"
	TopicPath       = "path"
	TopicRecipes    = "recipes"
)

// ErrUnknownTopic is returned by Topic when the requested name is not a
// registered topic. Callers can use errors.Is(err, ErrUnknownTopic) to
// distinguish a genuinely unknown name from internal failures (e.g. a
// corrupt embed.FS). The init() in this package panics at process start
// if any registered topic is missing from the embedded FS, so a non-
// ErrUnknownTopic error reaching production means the binary itself is
// broken and should not be treated as user input error.
var ErrUnknownTopic = errors.New("unknown help topic")

// TopicNames returns all topic names in canonical display order. The returned
// slice is freshly allocated and callers may mutate it freely.
func TopicNames() []string {
	return []string{
		TopicHooks,
		TopicConfig,
		TopicCompletion,
		TopicPath,
		TopicRecipes,
	}
}

// Topic returns the plain-text body of the given topic.
// Returns ErrUnknownTopic wrapped with the offending name when the name is
// not registered. Any other error indicates an internal failure (e.g. the
// embed.FS is corrupt or the topic file is missing from the build) and must
// not be silently translated into "unknown topic".
func Topic(name string) (string, error) {
	known := TopicNames()
	if !slices.Contains(known, name) {
		return "", fmt.Errorf("%w %q (known topics: %s)", ErrUnknownTopic, name, strings.Join(known, ", "))
	}
	body, err := topicFS.ReadFile("topics/" + name + ".txt")
	if err != nil {
		return "", fmt.Errorf("read help topic %q: %w", name, err)
	}
	return string(body), nil
}

// SectionHeader formats a section heading used by `gw help all`.
// kind and name are rendered as-is; the cmd/gw CLI passes upper-case kinds
// ("OVERVIEW", "COMMAND", "TOPIC") and lower-case names. Callers should keep
// this format stable: tests and downstream consumers grep on these boundaries.
func SectionHeader(kind, name string) string {
	if name == "" {
		return fmt.Sprintf("===== %s =====", kind)
	}
	return fmt.Sprintf("===== %s: %s =====", kind, name)
}
