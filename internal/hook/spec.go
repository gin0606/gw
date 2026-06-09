package hook

// Hook names. These match the file names under .gw/hooks/ and the identifiers
// surfaced by `gw help hooks`. Use these constants instead of string literals
// so that renames are caught at compile time.
const (
	PreAdd     = "pre-add"
	PostAdd    = "post-add"
	PreRemove  = "pre-remove"
	PostRemove = "post-remove"
)

// Environment variable names passed to hooks. These are documented in
// `gw help hooks` and listed in `gw init` hook templates.
const (
	EnvRepoRoot     = "GW_REPO_ROOT"
	EnvWorktreePath = "GW_WORKTREE_PATH"
	EnvBranch       = "GW_BRANCH"
)

// Names returns the hook names in lifecycle order (pre-add, post-add,
// pre-remove, post-remove). Each value matches the corresponding file name
// under .gw/hooks/. The returned slice is freshly allocated and callers may
// mutate it freely.
func Names() []string {
	return []string{PreAdd, PostAdd, PreRemove, PostRemove}
}

// EnvVars returns the environment variable names passed to every hook, in
// the order they appear in `gw help hooks` and in `gw init` templates. The
// returned slice is freshly allocated and callers may mutate it freely.
func EnvVars() []string {
	return []string{EnvRepoRoot, EnvWorktreePath, EnvBranch}
}
