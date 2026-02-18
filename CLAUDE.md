# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gw` is a git worktree wrapper with lifecycle hooks. It runs user scripts before and after worktree creation/removal, and automatically calculates worktree paths from branch names. Go 1.25+, sole external dependency is `github.com/BurntSushi/toml`.

Design philosophy: features achievable through hooks are not built into the tool itself.

## Common Commands

```sh
# Run all tests
go test ./...

# Run tests for a single package
go test ./internal/pathutil/

# Run a single test
go test ./internal/pathutil/ -run TestSanitize

# Vet
go vet ./...

# Build
go build ./cmd/gw/
```

CI runs `go vet ./...` and `go test ./...`.

## Architecture

```
cmd/gw/main.go          Entry point. Parses args and dispatches to subcommands
cmd/gw/completion.go     Shell completion logic (custom completers for add, rm)
internal/
  cmd/                   Subcommand implementations (init, add, rm, list)
  git/                   Git command wrappers (RepoRoot, BranchExists, ListWorktrees, ListLocalBranches, ListRefs, etc.)
  config/                Loads .gw/config (TOML)
  hook/                  Hook execution engine for .gw/hooks/
  pathutil/              Branch name sanitization and worktree path calculation
  testutil/              Test helper for creating temporary git repositories
```

### Subcommand Common Flow

Each subcommand follows the same pattern:
1. Detect repository root (`git.RepoRoot`)
2. Load config (`config.Load`) → compute base directory (`pathutil.BaseDir`)
3. Execute command-specific logic

### Test Pattern

Tests use `testutil.NewTestRepo(t)` to create a temporary environment with a bare repository + clone. They are integration-style tests that execute real git commands.
