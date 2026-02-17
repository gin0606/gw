# gw

A thin wrapper around `git worktree` with automatic path calculation and hook support.

[日本語](README.ja.md)

## Overview

`gw` simplifies git worktree management by automatically calculating worktree paths from branch names and providing a hook system for custom automation.

**Design philosophy:** Keep the core thin. Features that can be achieved through hooks are not built into the tool itself.

## Installation

```sh
go install github.com/gin0606/gw/cmd/gw@latest
```

## Usage

```
usage: gw <command> [<args>]

Commands:
   add       Create a new worktree
   rm        Remove a worktree
   go        Print worktree path
   version   Print version information
```

### `gw add <branch> [--from <ref>]`

Create a new worktree. The worktree path is automatically calculated from the branch name and printed to stdout.

```sh
# Create a worktree for an existing branch
gw add feature/user-auth

# Create a worktree with a new branch from a specific ref
gw add feature/new-feature --from origin/main

# Create and cd into the worktree
cd "$(gw add feature/user-auth)"
```

When `--from` is omitted and the branch does not exist, it is created from `origin/<default branch>` (falls back to `<default branch>` if the remote ref does not exist).

### `gw rm <identifier> [--force]`

Remove a worktree. The identifier can be a branch name or a sanitized directory name.

```sh
# Remove a worktree
gw rm feature/user-auth

# Force remove (even with uncommitted changes)
gw rm feature/user-auth --force
```

### `gw go <identifier>`

Print the path of an existing worktree. Designed for shell integration.

```sh
# cd into a worktree
cd "$(gw go feature/user-auth)"
```

### `gw version`

```sh
gw version
# gw version 0.1.0
```

## Configuration

Place a TOML configuration file at `.gw/config` in your repository root.

```toml
# Custom worktree base directory (absolute or relative to repository root)
worktrees_dir = "../my-worktrees"
```

| Key | Description | Default |
|---|---|---|
| `worktrees_dir` | Base directory for worktrees | `../<repo-name>-worktrees/` |

## Hooks

Place executable files in `.gw/hooks/` in your repository root.

### Available hooks

| Hook | Trigger | Working directory |
|---|---|---|
| `pre-add` | Before worktree creation | Repository root |
| `post-add` | After worktree creation | Worktree directory |
| `pre-remove` | Before worktree removal | Worktree directory |
| `post-remove` | After worktree removal | Repository root |

### Environment variables

The following environment variables are available in hooks:

| Variable | Description |
|---|---|
| `GW_REPO_ROOT` | Absolute path to the main repository root |
| `GW_WORKTREE_PATH` | Absolute path to the worktree |
| `GW_BRANCH` | Branch name |

### Example: auto-install dependencies after creating a worktree

`.gw/hooks/post-add`:

```sh
#!/bin/sh
if [ -f package.json ]; then
  npm install
fi
```

## License

[MIT](LICENSE)
