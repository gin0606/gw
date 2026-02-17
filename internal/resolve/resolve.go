package resolve

import (
	"fmt"
	"path/filepath"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/pathutil"
)

// Resolve resolves an identifier to a worktree absolute path and branch name.
// It finds a worktree whose path equals <baseDir>/<sanitize(identifier)>.
func Resolve(repoRoot, baseDir, identifier string) (path string, branch string, err error) {
	sanitized, err := pathutil.Sanitize(identifier)
	if err != nil {
		return "", "", fmt.Errorf("invalid identifier %q: %w", identifier, err)
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return "", "", err
	}

	target := filepath.Join(baseDir, sanitized)
	for _, wt := range worktrees {
		if wt.Path == target {
			return wt.Path, wt.Branch, nil
		}
	}

	return "", "", fmt.Errorf("worktree not found for identifier %q (expected path: %s)", identifier, target)
}
