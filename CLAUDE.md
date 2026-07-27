# CLAUDE.md

## Project Overview

`gw` is a git worktree wrapper with lifecycle hooks. It runs user scripts before and after worktree creation/removal, and automatically calculates worktree paths from branch names.

## Design Principles

- Features achievable through hooks are NOT built into the tool itself. Always prefer hooks over new built-in functionality.
- Do not reimplement git behavior. Delegate to git commands via the `internal/git` package.

## Verification

```sh
goimports -l .        # CI runs this (should produce no output)
go vet ./...          # CI runs this
staticcheck ./...     # CI runs this
go test ./...         # CI runs this
go build ./cmd/gw/
```

## Architecture

### Config & Hooks

Config (`.gw/config`) and hooks (`.gw/hooks/`) are documented in README.md. Refer to it for available keys, hook names, and environment variables.

### Subcommand Common Flow

Each subcommand follows the same pattern:
1. Detect repository root (`git.RepoRoot`)
2. Load config (`config.Load`) → compute base directory (`pathutil.BaseDir`)
3. Execute command-specific logic

### Test Pattern

Tests use `testutil.NewTestRepo(t)` to create a temporary environment with a bare repository + clone. They are integration-style tests that execute real git commands.
