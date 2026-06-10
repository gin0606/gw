package pathutil

// separatorReplacement is what "/" in branch names becomes in the worktree
// directory name. Must stay a single character: Sanitize passes it to
// strings.Trim as a cutset, where each rune is treated independently, so a
// multi-character value would silently change the trim semantics. Changing
// it also requires updating internal/help/topics/path.txt and the README,
// which hard-code "-".
const separatorReplacement = "-"

// DefaultBaseDirSuffix is appended to the repository name to form the default
// worktree base directory (a sibling of the repository). Exported so the
// `gw init` config template can render the same default the runtime uses.
const DefaultBaseDirSuffix = "-worktrees"
