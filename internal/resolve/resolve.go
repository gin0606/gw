package resolve

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/pathutil"
)

// Resolve resolves an identifier to a worktree absolute path and branch name.
// Resolution order:
//  1. Sanitized name path match: find a worktree whose path equals <baseDir>/<sanitize(identifier)>
//  2. Branch name scan: find a worktree within baseDir whose branch matches identifier
//  3. Error if no match
func Resolve(repoRoot, baseDir, identifier string) (path string, branch string, err error) {
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return "", "", err
	}

	// 1. Sanitized name path match
	sanitized, sanitizeErr := pathutil.Sanitize(identifier)
	if sanitizeErr == nil {
		target := filepath.Join(baseDir, sanitized)
		for _, wt := range worktrees {
			if wt.Path == target {
				return wt.Path, wt.Branch, nil
			}
		}
	}

	// 2. Branch name scan (within base directory)
	baseDirPrefix := baseDir + string(filepath.Separator)
	for _, wt := range worktrees {
		if !strings.HasPrefix(wt.Path, baseDirPrefix) {
			continue
		}
		if wt.Branch != "" && wt.Branch == identifier {
			return wt.Path, wt.Branch, nil
		}
	}

	return "", "", fmt.Errorf("worktree not found for identifier %q", identifier)
}
