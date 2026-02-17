package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin0606/gw/internal/config"
	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/pathutil"
	"github.com/gin0606/gw/internal/resolve"
)

// Go implements the "gw go" command.
func Go(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "gw: error: identifier required\n")
		return 1
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "gw: error: unknown argument: %s\n", args[1])
		return 1
	}

	identifier := args[0]

	// 1. Detect repo root
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	repoRoot, err := git.RepoRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 2. Load config and resolve base directory
	cfg, err := config.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	repoName := git.RepoName(repoRoot)
	baseDir := pathutil.BaseDir(repoRoot, repoName, cfg.WorktreesDir)

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 3. Resolve identifier
	wtPath, _, err := resolve.Resolve(repoRoot, baseDir, identifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw: error: %v\n", err)
		return 1
	}

	// 4. Output absolute path to stdout
	fmt.Println(wtPath)

	return 0
}
