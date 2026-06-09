package pathutil

// separatorReplacement is the character that "/" in branch names is replaced
// with when computing the worktree directory name. It must remain a single
// character: Sanitize passes it to strings.Trim as a cutset, which treats
// each rune independently, so a multi-character value would silently change
// the trim semantics. It is unexported because changing this constant alone
// is not enough: `internal/help/topics/path.txt` and the README examples
// hard-code "-" and would need to be updated in tandem. The constant exists
// for readability inside this package only.
const separatorReplacement = "-"

// DefaultBaseDirSuffix is appended to the repository name to form the default
// worktree base directory (a sibling of the repository). Exported so the
// `gw init` config template can render the same default the runtime uses.
const DefaultBaseDirSuffix = "-worktrees"
