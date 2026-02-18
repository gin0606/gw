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
	// No completion outside a git repository
	repoRoot, err := git.RepoRoot(".")
	if err != nil {
		return
	}

	// Check the last argument before --generate-shell-completion
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

	if strings.HasPrefix(prev, "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return
	}

	// Positional argument already provided; no further completion needed
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
	// No completion outside a git repository
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
	if prev != "--force" && strings.HasPrefix(prev, "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return
	}

	// Positional argument already provided; no further completion needed
	if cmd.NArg() > 0 {
		return
	}

	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil || len(worktrees) <= 1 {
		return
	}

	// Skip the first worktree (main worktree)
	for _, wt := range worktrees[1:] {
		fmt.Fprintln(cmd.Root().Writer, wt.Path)
	}
}
