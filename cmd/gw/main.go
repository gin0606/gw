package main

import (
	"fmt"
	"os"

	"github.com/gin0606/gw/internal/cmd"
)

var version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage())
		return 1
	}

	switch args[0] {
	case "version":
		cmd.Version(version)
		return 0
	case "add":
		return cmd.Add(args[1:])
	case "go":
		return cmd.Go(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "gw: error: unknown command '%s'\n", args[0])
		fmt.Fprint(os.Stderr, usage())
		return 1
	}
}

func usage() string {
	return `usage: gw <command> [<args>]

Commands:
   add       Create a new worktree
   go        Print worktree path
   version   Print version information
`
}
