package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin0606/gw/internal/git"
	"github.com/urfave/cli/v3"
)

func completeAdd(ctx context.Context, cmd *cli.Command) {
	repoRoot, err := git.RepoRoot(".")
	if err != nil {
		return
	}

	args := os.Args
	prev := ""
	for i, a := range args {
		if a == "--generate-shell-completion" && i > 0 {
			prev = args[i-1]
			break
		}
	}

	if prev == "--from" {
		refs, err := git.ListRefs(repoRoot)
		if err != nil {
			return
		}
		for _, r := range refs {
			fmt.Fprintln(cmd.Root().Writer, r)
		}
		return
	}

	if prev != "--no-hooks" && strings.HasPrefix(prev, "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return
	}

	if cmd.NArg() > 0 {
		return
	}

	branches, err := git.ListLocalBranches(repoRoot)
	if err != nil {
		return
	}
	for _, b := range branches {
		fmt.Fprintln(cmd.Root().Writer, b)
	}
}

func completeRemove(ctx context.Context, cmd *cli.Command) {
	repoRoot, err := git.RepoRoot(".")
	if err != nil {
		return
	}

	args := os.Args
	prev := ""
	for i, a := range args {
		if a == "--generate-shell-completion" && i > 0 {
			prev = args[i-1]
			break
		}
	}

	// --force is a bool flag; the next argument is a positional arg, not a flag value
	if prev != "--force" && prev != "--no-hooks" && strings.HasPrefix(prev, "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return
	}

	if cmd.NArg() > 0 {
		return
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil || len(worktrees) <= 1 {
		return
	}

	// The main worktree (index 0) cannot be removed via gw rm, so exclude it from completions.
	for _, wt := range worktrees[1:] {
		fmt.Fprintln(cmd.Root().Writer, wt.Path)
	}
}
