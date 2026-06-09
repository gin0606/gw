package help

import (
	"sort"
	"strings"
	"testing"
)

// TestTopicFS_MatchesTopicNames asserts that the set of embedded *.txt files
// exactly matches the set of names returned by TopicNames. This catches both
// directions of drift: adding a topics/foo.txt without registering "foo" in
// TopicNames, and registering a name without adding the corresponding file.
func TestTopicFS_MatchesTopicNames(t *testing.T) {
	entries, err := topicFS.ReadDir("topics")
	if err != nil {
		t.Fatal(err)
	}

	var fileNames []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".txt") {
			t.Errorf("unexpected non-.txt file under topics/: %s", name)
			continue
		}
		fileNames = append(fileNames, strings.TrimSuffix(name, ".txt"))
	}

	registered := TopicNames()
	sort.Strings(fileNames)
	sort.Strings(registered)

	if strings.Join(fileNames, ",") != strings.Join(registered, ",") {
		t.Errorf("topics/*.txt vs TopicNames mismatch:\n  files:      %v\n  registered: %v", fileNames, registered)
	}
}
