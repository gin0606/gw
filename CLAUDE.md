# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gw` is a thin wrapper CLI around git worktree with automatic path calculation and a hook system. Go 1.25+, sole external dependency is `github.com/BurntSushi/toml`.

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
internal/
  cmd/                   Subcommand implementations (add, rm, go, version)
  git/                   Git command wrappers (RepoRoot, BranchExists, ListWorktrees, etc.)
  config/                Loads .gw/config (TOML)
  hook/                  Hook execution engine for .gw/hooks/
  pathutil/              Branch name sanitization and worktree path calculation
  resolve/               Resolves identifier → worktree path (reverse lookup)
  testutil/              Test helper for creating temporary git repositories
```

### Subcommand Common Flow

Each subcommand (`cmd/add.go`, `cmd/remove.go`, `cmd/go.go`) follows the same pattern:
1. Detect repository root (`git.RepoRoot`)
2. Load config (`config.Load`) → compute base directory (`pathutil.BaseDir`)
3. Execute command-specific logic

### Output Convention

- **stdout**: Data only (paths, etc.) — used for shell integration like `cd "$(gw add ...)"`
- **stderr**: Logs, hook output, and errors. Git command output is also directed here
- Error message format: `gw: error: <message>` / `gw: warning: <message>`

### Test Pattern

Tests use `testutil.NewTestRepo(t)` to create a temporary environment with a bare repository + clone. They are integration-style tests that execute real git commands.

## Specification

See `SPECIFICATION.md` for detailed specs including path calculation rules, hook execution timing, and identifier resolution order.
