package config

import (
	"reflect"
	"testing"
)

// TestKeyWorktreesDir_MatchesStructTag pins the bidirectional link between
// KeyWorktreesDir (used for `gw init` rendering and help drift checks) and
// the toml tag that actually decides decoding. Without this, renaming the
// struct tag while leaving the constant alone would silently break loaded
// configs without any test catching the divergence.
func TestKeyWorktreesDir_MatchesStructTag(t *testing.T) {
	field, ok := reflect.TypeOf(Config{}).FieldByName("WorktreesDir")
	if !ok {
		t.Fatal("Config.WorktreesDir field missing")
	}
	if got := field.Tag.Get("toml"); got != KeyWorktreesDir {
		t.Errorf("Config.WorktreesDir toml tag = %q, KeyWorktreesDir = %q", got, KeyWorktreesDir)
	}
}
