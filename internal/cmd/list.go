package cmd

import (
	"fmt"
	"os"

	"github.com/gin0606/gw/internal/git"
)

// List implements the "gw list" command.
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
