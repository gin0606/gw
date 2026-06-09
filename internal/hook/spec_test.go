package hook

import "testing"

// TestName_Cwd_PanicsOnUnknown pins the panic contract documented on
// Name.cwd: an undeclared Name reaching cwd is a programming error, so the
// switch must not silently fall back to repoRoot or worktreePath. Without
// this test, a "fix" that turns the panic into a default return would slip
// through unnoticed and silently mis-route a hook's working directory.
func TestName_Cwd_PanicsOnUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown Name, got none")
		}
	}()
	Name("bogus").cwd("/repo", "/worktree")
}
