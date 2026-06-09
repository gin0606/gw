package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gin0606/gw/internal/cmd"
	"github.com/gin0606/gw/internal/help"
	"github.com/urfave/cli/v3"
)

var version = "dev"

func init() {
	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

const rootDescription = `gw wraps "git worktree" with lifecycle hooks and calculates each worktree's
path from the branch name.

In-depth reference:
  gw help hooks       Lifecycle hooks (.gw/hooks/) and environment variables.
  gw help config      .gw/config keys and defaults.
  gw help completion  Shell completion and dynamic completion candidates.
  gw help path        How worktree paths are calculated.
  gw help recipes     Shell snippets that compose with gw output.

For a single dump that an agent can consume in one shot:
  gw help all

Shell completion (the completion subcommand is hidden from the COMMANDS
list below):
  gw completion bash | zsh | fish | pwsh

Output contract:
  Commands write machine-readable primary results to stdout. Diagnostics,
  git output, and hook output are written to stderr.`

func main() {
	root := newApp()
	err := root.Run(context.Background(), os.Args)
	if err == nil {
		return
	}
	// By the time control reaches here, the root ExitErrHandler set in newApp
	// has suppressed urfave's default HandleExitCoder, so a cli.ExitCoder
	// (cli.Exit, OnUsageError result) propagates back unprinted. Print it
	// ourselves, skipping the empty-string case so a silent cli.Exit("", N)
	// (e.g. the one handleUsageError returns) does not produce a stray blank
	// line. We use a direct type assertion rather than errors.As to avoid
	// matching `*exec.ExitError` and other unrelated types that happen to
	// satisfy the ExitCoder shape.
	if coder, ok := err.(cli.ExitCoder); ok {
		if msg := coder.Error(); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(coder.ExitCode())
	}
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func newApp() *cli.Command {
	root := &cli.Command{
		Name:                  "gw",
		Usage:                 "A thin wrapper around git worktree with lifecycle hooks",
		Description:           rootDescription,
		Version:               version,
		EnableShellCompletion: true,
		HideHelpCommand:       true,
		OnUsageError:          handleUsageError,
		// Disable urfave's default ExitCoder handling so cli.ExitCoder values
		// (including the silent cli.Exit("", 1) returned by handleUsageError)
		// propagate back to main() instead of being printed by HandleExitCoder.
		// Setting this on the root is sufficient: Command.handleExitCoder
		// recurses to the parent until it reaches the root and only the root's
		// ExitErrHandler is invoked, so per-subcommand handlers are ignored.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Commands: []*cli.Command{
			cmdInit(),
			cmdAdd(),
			cmdRemove(),
			cmdList(),
			cmdHelp(),
		},
	}
	// Propagate the usage-error handler to every user-facing subcommand so
	// flag-parsing failures inside them are reported through the same path.
	for _, sub := range root.Commands {
		sub.OnUsageError = handleUsageError
	}
	return root
}

// handleUsageError formats urfave/cli's flag-parsing failures itself and
// returns a silent cli.Exit so main does not re-print the underlying error.
func handleUsageError(ctx context.Context, cmd *cli.Command, err error, isSubcommand bool) error {
	fmt.Fprintf(cmd.Root().ErrWriter, "Incorrect Usage: %s\n", err.Error())
	return cli.Exit("", 1)
}

const initDescription = `Create .gw/config and .gw/hooks/ with default templates in the repository
root. Errors if .gw/ already exists.

The hook templates are short local references; full hook semantics live in
` + "`gw help hooks`" + `.

Output:
  On success, prints nothing to stdout. The initialization message and
  diagnostics are written to stderr.`

func cmdInit() *cli.Command {
	return &cli.Command{
		Name:        "init",
		Usage:       "Initialize .gw/ configuration and hook templates",
		UsageText:   "gw init",
		Description: initDescription,
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() > 0 {
				return fmt.Errorf("unexpected argument: %s", c.Args().First())
			}
			return cmd.Init()
		},
	}
}

const addDescription = `Create a new worktree for <branch>.

Branch resolution:
  If <branch> already exists, gw uses it. Otherwise gw creates a new branch
  from origin/<default-branch>; if that remote-tracking ref does not exist,
  gw falls back to the local <default-branch>. The default branch is the
  one origin/HEAD points at, so creating a new branch without --from
  requires origin/HEAD to be configured
  ('git remote set-head origin --auto'); otherwise pass --from <ref> to
  choose the start point yourself. Passing --from together with an
  existing branch is an error.

Worktree path:
  The path is computed from the branch name and the configured base
  directory. See ` + "`gw help path`" + `.

Hooks:
  pre-add runs first; if it exits non-zero, gw aborts and the worktree is
  not created. After git creates the worktree, post-add runs; if it exits
  non-zero, gw prints a warning to stderr and the operation still succeeds.
  --no-hooks skips both. See ` + "`gw help hooks`" + `.

Output:
  On success, prints the new worktree absolute path to stdout.
  Git output and hook output are written to stderr.`

func cmdAdd() *cli.Command {
	return &cli.Command{
		Name:          "add",
		Usage:         "Create a new worktree",
		UsageText:     "gw add [--from <ref>] [--no-hooks] <branch>",
		Description:   addDescription,
		ShellComplete: completeAdd,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "from", Usage: "Create new branch from specified ref"},
			&cli.BoolFlag{Name: "no-hooks", Usage: "Skip pre-add and post-add hooks"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() < 1 {
				return fmt.Errorf("branch name required (try 'gw add --help' or 'gw help add')")
			}
			if c.Args().Len() > 1 {
				return fmt.Errorf("unexpected argument: %s", c.Args().Get(1))
			}
			return cmd.Add(c.Args().First(), c.String("from"), c.Bool("no-hooks"))
		},
	}
}

const removeDescription = `Remove the worktree at <path>. <path> may be absolute or relative.
Removing the main worktree is rejected; the main worktree is the one whose
path equals the repository root (the first entry printed by ` + "`gw list`" + `).

Hooks:
  pre-remove runs first. If it exits non-zero, gw aborts by default; with
  --force, gw prints a warning to stderr and continues. After git removes
  the worktree, post-remove runs; if it exits non-zero, gw prints a warning
  and the operation still succeeds. --no-hooks skips both.
  See ` + "`gw help hooks`" + `.

--force is also forwarded to ` + "`git worktree remove`" + `, allowing removal of a
worktree with uncommitted or untracked changes.

Output:
  On success, prints nothing to stdout. Git output, hook output, and
  warnings are written to stderr.`

func cmdRemove() *cli.Command {
	return &cli.Command{
		Name:          "rm",
		Usage:         "Remove a worktree",
		UsageText:     "gw rm [--force] [--no-hooks] <path>",
		Description:   removeDescription,
		ShellComplete: completeRemove,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Force removal even if worktree is dirty or hook fails"},
			&cli.BoolFlag{Name: "no-hooks", Usage: "Skip pre-remove and post-remove hooks"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() < 1 {
				return fmt.Errorf("path required (try 'gw rm --help' or 'gw help rm')")
			}
			if c.Args().Len() > 1 {
				return fmt.Errorf("unexpected argument: %s", c.Args().Get(1))
			}
			return cmd.Remove(c.Args().First(), c.Bool("force"), c.Bool("no-hooks"))
		},
	}
}

const listDescription = `List all worktrees of the current repository, one absolute path per line.
The list includes the main worktree.

Output:
  Prints worktree absolute paths to stdout, one per line.
  Diagnostics are written to stderr.`

func cmdList() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all worktrees",
		UsageText:   "gw list",
		Description: listDescription,
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() > 0 {
				return fmt.Errorf("unexpected argument: %s", c.Args().First())
			}
			return cmd.List()
		},
	}
}

const helpDescription = `Show help for a command or topic.

  gw help              Root overview (same as gw --help).
  gw help <command>    Same as gw <command> --help.
  gw help <topic>      One of: hooks, config, completion, path, recipes.
  gw help all          Root overview, every user-facing command (except
                       help itself), and every topic.`

// helpAllExcluded names the subcommands omitted from `gw help all`:
// `completion` is auto-injected by urfave/cli for shell-script generation and
// is documented under `gw help completion`, and `help` itself would be
// self-referential.
var helpAllExcluded = map[string]bool{"help": true, "completion": true}

// helpAllCommandsOf returns the user-facing subcommands of root in display
// order, deriving the list from root.Commands so it cannot drift from the
// commands actually registered. Hidden commands and the ones listed in
// helpAllExcluded are filtered out.
func helpAllCommandsOf(root *cli.Command) []*cli.Command {
	var out []*cli.Command
	for _, sub := range root.Commands {
		if sub.Hidden || helpAllExcluded[sub.Name] {
			continue
		}
		out = append(out, sub)
	}
	return out
}

func cmdHelp() *cli.Command {
	return &cli.Command{
		Name:        "help",
		Aliases:     []string{"h"},
		Usage:       "Show help for a command or topic",
		UsageText:   "gw help [<command>|<topic>|all]",
		Description: helpDescription,
		Action: func(ctx context.Context, c *cli.Command) error {
			// urfave/cli treats a command named "help" specially and skips
			// flag parsing for it (see command_run.go), so "--help" / "-h"
			// arrive as positional args rather than being intercepted as
			// the framework help flag. Translate any occurrence of those
			// flags to "show help for the help command", mirroring how the
			// framework would have handled them on any other subcommand.
			// (`gw help --help`, `gw help -h`, and combinations like
			// `gw help hooks --help` all end up here.)
			args := c.Args().Slice()
			root := c.Root()
			for _, a := range args {
				if a == "--help" || a == "-h" {
					return cli.ShowCommandHelp(ctx, root, "help")
				}
			}
			switch len(args) {
			case 0:
				return cli.ShowRootCommandHelp(root)
			case 1:
				return runHelp(ctx, root, args[0])
			default:
				return fmt.Errorf("unexpected argument: %s", args[1])
			}
		},
	}
}

// runHelp resolves target as a topic first, then as a subcommand. Topic and
// command namespaces overlap on "completion" (the framework registers a
// `completion` subcommand for shell script generation, and a `completion`
// topic covers both that and dynamic completion candidates); the topic wins
// because it is the more comprehensive document.
func runHelp(ctx context.Context, root *cli.Command, target string) error {
	if target == "all" {
		return printHelpAll(root)
	}
	body, err := help.Topic(target)
	if err == nil {
		fmt.Fprint(root.Writer, body)
		return nil
	}
	if !errors.Is(err, help.ErrUnknownTopic) {
		// Internal failure (corrupt embed FS, missing topic file); surface it.
		return err
	}
	for _, sub := range root.Commands {
		if sub.HasName(target) {
			return cli.ShowCommandHelp(ctx, root, target)
		}
	}
	// Match urfave/cli's wording and exit code for `gw <unknown>` so the two
	// entry points (`gw help <unknown>` vs `gw <unknown>`) behave the same.
	return cli.Exit(fmt.Sprintf("No help topic for '%s'", target), 3)
}

func printHelpAll(root *cli.Command) error {
	w := root.Writer

	fmt.Fprintln(w, help.SectionHeader("OVERVIEW", ""))
	cli.HelpPrinter(w, cli.RootCommandHelpTemplate, root)

	for _, sub := range helpAllCommandsOf(root) {
		fmt.Fprintln(w)
		fmt.Fprintln(w, help.SectionHeader("COMMAND", sub.Name))
		cli.HelpPrinter(w, cli.CommandHelpTemplate, sub)
	}

	for _, name := range help.TopicNames() {
		body, err := help.Topic(name)
		if err != nil {
			return fmt.Errorf("internal: render help topic %q: %w", name, err)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, help.SectionHeader("TOPIC", name))
		fmt.Fprint(w, body)
	}
	return nil
}
