package hook

// Name identifies a worktree lifecycle hook. The constants below name every
// hook gw will execute; passing any other value to Run is a programming error.
// Using these constants instead of string literals lets the compiler catch
// renames and typos at the call site.
type Name string

const (
	PreAdd     Name = "pre-add"
	PostAdd    Name = "post-add"
	PreRemove  Name = "pre-remove"
	PostRemove Name = "post-remove"
)

// Environment variable names passed to hooks. These are documented in
// `gw help hooks` and listed in `gw init` hook templates.
const (
	EnvRepoRoot     = "GW_REPO_ROOT"
	EnvWorktreePath = "GW_WORKTREE_PATH"
	EnvBranch       = "GW_BRANCH"
)

// cwd returns the directory in which a hook of this name should be executed.
// pre-add and post-remove run at repoRoot because the worktree does not exist
// on disk at those moments. post-add and pre-remove run inside the worktree,
// which is the surface most hook scripts want to operate on. Encoding this
// rule on Name keeps every Run call site from having to remember it.
//
// Panics on a Name not declared in this file; that would indicate a
// programming error (a fabricated Name slipped past the type).
func (n Name) cwd(repoRoot, worktreePath string) string {
	switch n {
	case PreAdd, PostRemove:
		return repoRoot
	case PostAdd, PreRemove:
		return worktreePath
	}
	panic("hook: unknown Name " + string(n))
}

// Names returns the hook names in lifecycle order (pre-add, post-add,
// pre-remove, post-remove). Each value matches the corresponding file name
// under .gw/hooks/. The returned slice is freshly allocated and callers may
// mutate it freely.
func Names() []Name {
	return []Name{PreAdd, PostAdd, PreRemove, PostRemove}
}

// EnvVars returns the environment variable names passed to every hook, in
// the order they appear in `gw help hooks` and in `gw init` templates. The
// returned slice is freshly allocated and callers may mutate it freely.
func EnvVars() []string {
	return []string{EnvRepoRoot, EnvWorktreePath, EnvBranch}
}
