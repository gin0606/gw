package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gin0606/gw/internal/cmd"
	"github.com/urfave/cli/v3"
)

var version = "0.1.0"

func main() {
	root := &cli.Command{
		Name:    "gw",
		Usage:   "A thin wrapper around git worktree",
		Version: version,
		Commands: []*cli.Command{
			cmdInit(),
			cmdAdd(),
			cmdRemove(),
			cmdList(),
		},
	}
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func cmdInit() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "Initialize .gw/ configuration",
		UsageText: "gw init",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() > 0 {
				return fmt.Errorf("unexpected argument: %s", c.Args().First())
			}
			return cmd.Init()
		},
	}
}

func cmdAdd() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Create a new worktree",
		UsageText: "gw add [--from <ref>] <branch>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "from", Usage: "Create new branch from specified ref"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() < 1 {
				return fmt.Errorf("branch name required")
			}
			if c.Args().Len() > 1 {
				return fmt.Errorf("unexpected argument: %s", c.Args().Get(1))
			}
			return cmd.Add(c.Args().First(), c.String("from"))
		},
	}
}

func cmdRemove() *cli.Command {
	return &cli.Command{
		Name:      "rm",
		Usage:     "Remove a worktree",
		UsageText: "gw rm [--force] <path>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Force removal even if worktree is dirty or hook fails"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() < 1 {
				return fmt.Errorf("path required")
			}
			if c.Args().Len() > 1 {
				return fmt.Errorf("unexpected argument: %s", c.Args().Get(1))
			}
			return cmd.Remove(c.Args().First(), c.Bool("force"))
		},
	}
}

func cmdList() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List all worktrees",
		UsageText: "gw list",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() > 0 {
				return fmt.Errorf("unexpected argument: %s", c.Args().First())
			}
			return cmd.List()
		},
	}
}
