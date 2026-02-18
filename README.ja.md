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

## コマンド

- **`gw init`** — `.gw/` ディレクトリをデフォルト設定とフックテンプレートで初期化する。
- **`gw add <branch> [--from <ref>]`** — worktree を作成する。ブランチ名から自動計算されたパスが stdout に出力される。`--from` を省略してブランチが存在しない場合、`origin/<デフォルトブランチ>` から作成される。
- **`gw rm <path> [--force]`** — worktree をパス指定（絶対・相対）で削除する。`--force` で未コミット変更があっても強制削除。
- **`gw list`** — 各 worktree の絶対パスを1行ずつ出力する。

## シェル補完

`gw completion` で補完スクリプトを生成できます。

```sh
# Bash
gw completion bash > /etc/bash_completion.d/gw

# Zsh
gw completion zsh > "${fpath[1]}/_gw"

# Fish
gw completion fish > ~/.config/fish/completions/gw.fish
```

サブコマンドでタブ補完が利用できます。

- `gw add <TAB>` — ローカルブランチ名
- `gw add --from <TAB>` — 全 ref（ブランチ、リモート、タグ）
- `gw rm <TAB>` — worktree パス（メイン worktree を除く）

## レシピ

各コマンドはシェルのパイプと組み合わせて使うことを想定しています。

```sh
# worktree を作成して cd
cd "$(gw add feature/user-auth)"

# fzf で worktree を選択して cd
cd "$(gw list | fzf)"

# fzf で選択した worktree を削除
gw rm "$(gw list | fzf)"
```

## 設定

リポジトリルートの `.gw/config` に TOML 形式で設定を記述します。

```toml
# worktree の格納先（絶対パスまたはリポジトリルートからの相対パス）
worktrees_dir = "../my-worktrees"
```

| キー            | 説明                                | デフォルト                     |
| --------------- | ----------------------------------- | ------------------------------ |
| `worktrees_dir` | worktree の格納先ベースディレクトリ | `../<リポジトリ名>-worktrees/` |

## フック

リポジトリルートの `.gw/hooks/` に実行可能ファイルを配置します。

### フック一覧

| フック名      | トリガー        | 実行ディレクトリ      |
| ------------- | --------------- | --------------------- |
| `pre-add`     | worktree 作成前 | リポジトリルート      |
| `post-add`    | worktree 作成後 | worktree ディレクトリ |
| `pre-remove`  | worktree 削除前 | worktree ディレクトリ |
| `post-remove` | worktree 削除後 | リポジトリルート      |

### 環境変数

フック内では以下の環境変数が利用できます。

| 変数               | 説明                             |
| ------------------ | -------------------------------- |
| `GW_REPO_ROOT`     | メインリポジトリルートの絶対パス |
| `GW_WORKTREE_PATH` | worktree の絶対パス              |
| `GW_BRANCH`        | ブランチ名                       |

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
