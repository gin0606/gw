# gw

`git worktree` の薄いラッパー CLI。パス自動計算とフック機能を備えています。

[English](README.md)

## 概要

`gw` はブランチ名から worktree のパスを自動計算し、フックによるカスタム自動化をサポートすることで、git worktree の管理を簡単にします。

**設計方針:** フックで実現可能な機能は本体に組み込まず、コアを薄く保ちます。

## インストール

```sh
go install github.com/gin0606/gw/cmd/gw@latest
```

## 使い方

```
usage: gw <command> [<args>]

Commands:
   init      Initialize .gw/ configuration
   add       Create a new worktree
   rm        Remove a worktree
   go        Print worktree path
   version   Print version information
```

### `gw init`

`.gw/` ディレクトリをデフォルト設定とフックテンプレートで初期化します。

```sh
gw init
```

以下のファイルが作成されます:
- `.gw/config` — デフォルトの `worktrees_dir` 設定
- `.gw/hooks/post-add` — コメントアウトされたフックテンプレート

### `gw add <branch> [--from <ref>]`

worktree を作成します。ブランチ名から自動計算されたパスが stdout に出力されます。

```sh
# 既存ブランチの worktree を作成
gw add feature/user-auth

# 特定の ref から新規ブランチで worktree を作成
gw add feature/new-feature --from origin/main

# worktree を作成して cd
cd "$(gw add feature/user-auth)"
```

`--from` を省略してブランチが存在しない場合、`origin/<デフォルトブランチ>` から作成されます（リモート ref が存在しない場合は `<デフォルトブランチ>` にフォールバック）。

### `gw rm <path> [--force]`

worktree をパス指定で削除します。絶対パスまたは相対パスを受け付けます（`gw list` の出力をそのまま使えます）。

```sh
# worktree を削除
gw rm /path/to/repo-worktrees/feature-user-auth

# gw list と組み合わせる
gw rm "$(gw list | grep feature-user-auth)"

# 強制削除（未コミット変更があっても削除）
gw rm /path/to/repo-worktrees/feature-user-auth --force
```

### `gw go <identifier>`

既存の worktree のパスを出力します。シェル連携を想定しています。

```sh
# worktree に cd
cd "$(gw go feature/user-auth)"
```

### `gw version`

```sh
gw version
# gw version 0.1.0
```

## 設定

リポジトリルートの `.gw/config` に TOML 形式で設定を記述します。

```toml
# worktree の格納先（絶対パスまたはリポジトリルートからの相対パス）
worktrees_dir = "../my-worktrees"
```

| キー | 説明 | デフォルト |
|---|---|---|
| `worktrees_dir` | worktree の格納先ベースディレクトリ | `../<リポジトリ名>-worktrees/` |

## フック

リポジトリルートの `.gw/hooks/` に実行可能ファイルを配置します。

### フック一覧

| フック名 | トリガー | 実行ディレクトリ |
|---|---|---|
| `pre-add` | worktree 作成前 | リポジトリルート |
| `post-add` | worktree 作成後 | worktree ディレクトリ |
| `pre-remove` | worktree 削除前 | worktree ディレクトリ |
| `post-remove` | worktree 削除後 | リポジトリルート |

### 環境変数

フック内では以下の環境変数が利用できます。

| 変数 | 説明 |
|---|---|
| `GW_REPO_ROOT` | メインリポジトリルートの絶対パス |
| `GW_WORKTREE_PATH` | worktree の絶対パス |
| `GW_BRANCH` | ブランチ名 |

### 例: worktree 作成後に依存関係を自動インストール

`.gw/hooks/post-add`:

```sh
#!/bin/sh
if [ -f package.json ]; then
  npm install
fi
```

## ライセンス

[MIT](LICENSE)
