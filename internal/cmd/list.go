package cmd

import (
	"fmt"
	"os"

	"github.com/gin0606/gw/internal/git"
)

// List prints the path of every worktree registered in the current repository, one per line.
func List() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return err
	}

	for _, wt := range worktrees {
		fmt.Println(wt.Path)
	}

	return nil
}
