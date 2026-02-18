# gw

A git worktree wrapper with lifecycle hooks.

[日本語](README.ja.md)

## Overview

`gw` wraps `git worktree` with lifecycle hooks that run your scripts before and after worktree creation and removal — automating setup, teardown, and validation. It also automatically calculates worktree paths from branch names.

**Design philosophy:** Keep the core thin. Features that can be achieved through hooks are not built into the tool itself.

## Installation

```sh
brew install gin0606/tap/gw
```

Or with `go install`:

```sh
go install github.com/gin0606/gw/cmd/gw@latest
```

## Commands

- **`gw init`** — Initialize `.gw/` directory with default configuration and hook templates.
- **`gw add <branch> [--from <ref>]`** — Create a new worktree. The path is calculated from the branch name and printed to stdout. When `--from` is omitted and the branch does not exist, it is created from `origin/<default branch>`.
- **`gw rm <path> [--force]`** — Remove a worktree by its path (absolute or relative). Use `--force` to remove even with uncommitted changes.
- **`gw list`** — Print the absolute path of each worktree, one per line.

## Hooks

Place executable files in `.gw/hooks/` in your repository root. Hooks let you automate any workflow around worktree operations.

### Available hooks

| Hook          | Trigger                  | Working directory  |
| ------------- | ------------------------ | ------------------ |
| `pre-add`     | Before worktree creation | Repository root    |
| `post-add`    | After worktree creation  | Worktree directory |
| `pre-remove`  | Before worktree removal  | Worktree directory |
| `post-remove` | After worktree removal   | Repository root    |

### Environment variables

The following environment variables are available in hooks:

| Variable           | Description                               |
| ------------------ | ----------------------------------------- |
| `GW_REPO_ROOT`     | Absolute path to the main repository root |
| `GW_WORKTREE_PATH` | Absolute path to the worktree             |
| `GW_BRANCH`        | Branch name                               |

### Examples

**Install dependencies and copy untracked files** (`.gw/hooks/post-add`):

```sh
#!/bin/sh
npm install
cp "$GW_REPO_ROOT/.env" "$GW_WORKTREE_PATH/.env"
```

**Stop services before removal** (`.gw/hooks/pre-remove`):

```sh
#!/bin/sh
docker compose down
```

**Delete merged branches after removal** (`.gw/hooks/post-remove`):

```sh
#!/bin/sh
git fetch --prune origin
if git merge-base --is-ancestor "$GW_BRANCH" origin/main 2>/dev/null; then
  git branch -D "$GW_BRANCH"
fi
```

## Recipes

Commands are designed to compose with standard shell tools.

```sh
# Create a worktree and cd into it
cd "$(gw add feature/user-auth)"

# Interactively select a worktree with fzf
cd "$(gw list | fzf)"

# Remove a worktree selected with fzf
gw rm "$(gw list | fzf)"
```

## Shell Completion

Generate completion scripts with `gw completion`:

```sh
# Bash
gw completion bash > /etc/bash_completion.d/gw

# Zsh
gw completion zsh > "${fpath[1]}/_gw"

# Fish
gw completion fish > ~/.config/fish/completions/gw.fish
```

Tab completion is available for subcommands:

- `gw add <TAB>` — local branch names
- `gw add --from <TAB>` — all refs (branches, remotes, tags)
- `gw rm <TAB>` — worktree paths (excluding the main worktree)

## Configuration

Place a TOML configuration file at `.gw/config` in your repository root.

```toml
# Custom worktree base directory (absolute or relative to repository root)
worktrees_dir = "../my-worktrees"
```

| Key             | Description                  | Default                    |
| --------------- | ---------------------------- | -------------------------- |
| `worktrees_dir` | Base directory for worktrees | Adjacent to the repository |

## License

[MIT](LICENSE)
