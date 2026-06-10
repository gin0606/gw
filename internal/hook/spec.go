package hook

// Name identifies a worktree lifecycle hook. Only the constants declared
// below are valid; passing any other value to Run is a programming error.
type Name string

const (
	PreAdd     Name = "pre-add"
	PostAdd    Name = "post-add"
	PreRemove  Name = "pre-remove"
	PostRemove Name = "post-remove"
)

// Environment variable names exposed to hook scripts. These form the
// user-facing contract; renaming any of them is a breaking change.
const (
	EnvRepoRoot     = "GW_REPO_ROOT"
	EnvWorktreePath = "GW_WORKTREE_PATH"
	EnvBranch       = "GW_BRANCH"
)

// cwd returns the directory in which a hook of this name should run.
// pre-add and post-remove execute at repoRoot because the worktree does
// not exist on disk at those moments; post-add and pre-remove execute
// inside the worktree itself. Panics on an unknown Name.
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
// under .gw/hooks/.
func Names() []Name {
	return []Name{PreAdd, PostAdd, PreRemove, PostRemove}
}

// EnvVars returns the environment variable names passed to every hook, in
// the order they appear in `gw help hooks` and in `gw init` templates. The
// returned slice is freshly allocated and callers may mutate it freely.
func EnvVars() []string {
	return []string{EnvRepoRoot, EnvWorktreePath, EnvBranch}
}
