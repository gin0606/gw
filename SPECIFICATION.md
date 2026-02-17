# gw 仕様書

**バージョン:** 0.1.0

`gw` は git worktree の薄いラッパー CLI ツールである。パス自動計算とフック実行を核とする。

**設計方針:** フックで実現可能な機能は基本的に本体に組み込まない。

**出力規約:** stdout にはデータ（パス等）のみを出力し、stderr にログ・フック出力・エラーを出力する。

**リポジトリ検出:** worktree 内から実行された場合でも、メインリポジトリを正しく検出して動作する。

---

## 1. コマンド

**共通ルール:** 各コマンドは定義されていない引数・オプションが渡された場合はエラーとする。

### 1.1 `gw init`

`.gw/` ディレクトリと初期ファイルを作成する。

#### 引数・オプション

なし。

#### 処理フロー

1. リポジトリルートを検出（worktree 内からも可）
2. `<repoRoot>/.gw/` が既に存在する場合はエラー
3. 以下のディレクトリ・ファイルを作成:
   - `.gw/config`
   - `.gw/hooks/post-add`（実行権限付き `0755`、全行コメント）

#### 生成ファイル

**`.gw/config`:**（リポジトリ名が `myproject` の場合）

```toml
# See https://github.com/gin0606/gw
worktrees_dir = "../myproject-worktrees"
```

`worktrees_dir` にはデフォルト値（`../<repo-name>-worktrees`）を展開して記述する。

**`.gw/hooks/post-add`:**

```sh
#!/bin/sh
# This hook is called after a worktree is created.
# See https://github.com/gin0606/gw for other available hooks.
#
# Available environment variables:
#   GW_REPO_ROOT       - Main repository root
#   GW_WORKTREE_PATH   - Worktree path
#   GW_BRANCH          - Branch name
#
# Example: Install dependencies
# npm install
```

#### 出力

- **stdout**: なし
- **stderr**: `Initialized .gw/ in <repoRoot>`

---

### 1.2 `gw add <branch> [--from <ref>]`

worktree を作成し、作成先パスを stdout に出力する。

#### 引数・オプション

| 引数/オプション | 必須 | 説明 |
|---|---|---|
| `<branch>` | はい | ブランチ名 |
| `--from <ref>` | いいえ | 指定した ref から新規ブランチを作成。省略時は `origin/<デフォルトブランチ>`（存在しなければ `<デフォルトブランチ>` にフォールバック）。既存ブランチ指定時は使用不可。デフォルトブランチは `origin/HEAD` （`git symbolic-ref refs/remotes/origin/HEAD`）から取得し、未設定の場合はエラー |

#### 処理フロー

1. リポジトリルートを検出
2. worktree パスを計算（[2. パス計算](#2-パス計算)参照）
3. ブランチの存在を確認し、引数を検証
   - **ブランチが既に存在する場合**: `--from` が指定されていればエラー
   - **ブランチが存在しない場合かつ `--from` 省略**: 起点 ref を決定（`origin/<デフォルトブランチ>` が存在すればそれを使用、存在しなければ `<デフォルトブランチ>` にフォールバック）
4. `pre-add` フックをリポジトリルートで実行（[3. フックシステム](#3-フックシステム)参照）
5. worktree を作成
   - **ブランチが存在しない場合**: `git worktree add <path> -b <branch> <ref>` を実行（新規作成）
   - **ブランチが既に存在する場合**: `git worktree add <path> <branch>` を実行（既存チェックアウト）
6. `post-add` フックを実行（[3. フックシステム](#3-フックシステム)参照）
7. 作成先パスを stdout に出力

#### 出力

- **stdout**: worktree の絶対パス
- **stderr**: ログメッセージ

---

### 1.3 `gw rm <identifier> [--force]`

worktree を削除する。ブランチは削除しない（`git worktree remove` 準拠）。

#### 引数・オプション

| 引数/オプション | 必須 | 説明 |
|---|---|---|
| `<identifier>` | はい | 削除対象の worktree（[4. identifier 解決](#4-identifier-解決)参照） |
| `--force` | いいえ | 未コミット変更がある worktree も強制削除。`pre-remove` フック失敗時も続行 |

#### 処理フロー

1. identifier から worktree パスを解決
2. `pre-remove` フックを worktree ディレクトリ内で実行
   - 失敗かつ `--force` なし: エラー終了
   - 失敗かつ `--force` あり: 警告を出力して続行
3. `git worktree remove [--force] <path>` を実行
4. `post-remove` フックをリポジトリルートで実行

#### 出力

- **stdout**: なし
- **stderr**: ログメッセージ

---

### 1.4 `gw go <identifier>`

worktree のパスを stdout に出力する。`cd "$(gw go <id>)"` のようにシェル連携で使用する。

#### 引数

| 引数 | 必須 | 説明 |
|---|---|---|
| `<identifier>` | はい | 対象の worktree（[4. identifier 解決](#4-identifier-解決)参照） |

#### 出力

- **stdout**: worktree の絶対パス
- **stderr**: ログメッセージ

---

### 1.5 `gw version`

バージョン情報を stdout に出力する（`git --version` 準拠）。

```
gw version <VERSION>
```

### 1.6 引数なし・不正コマンド

引数なしまたは不正なコマンドの場合、usage を stderr に出力し終了コード 1 で終了する（git 準拠）。

---

## 2. パス計算

### 2.1 ベースディレクトリ

worktree を格納する親ディレクトリ。

| 優先順位 | ソース | 値 |
|---|---|---|
| 1 | `.gw/config` の `worktrees_dir` | 指定されたパス（リポジトリルートからの相対パスまたは絶対パス） |
| 2 | デフォルト | `../<repo-name>-worktrees/` （リポジトリの親ディレクトリ基準）。`<repo-name>` はリポジトリルートのディレクトリ名（basename） |

ベースディレクトリが存在しない場合は自動的に作成する。

### 2.2 ブランチ名サニタイズ

ブランチ名をファイルシステム上で安全なフォルダ名に変換する。

**変換ルール:**
1. `/` をハイフン（`-`）に置換
2. 先頭・末尾のハイフンを除去

**変換例:**

| 入力 | 出力 |
|---|---|
| `feature/user-auth` | `feature-user-auth` |
| `feature/auth/login` | `feature-auth-login` |

**バリデーション:** サニタイズ結果が空文字列、`.`、`..` の場合はエラー。

**衝突:** サニタイズ後のディレクトリが既に存在する場合はエラー。異なるブランチ名が同じサニタイズ結果になるケース（例: `feature/foo-bar` と `feature/foo/bar`）を含む。

### 2.3 最終パス

```
<base_dir>/<sanitized-branch>
```

---

## 3. フックシステム

メインリポジトリルートの `.gw/hooks/` ディレクトリに実行可能ファイルを配置する（git hooks パターン）。

### 3.1 フックフェーズ

| フック名 | トリガー | 実行場所 |
|---|---|---|
| `pre-add` | worktree 作成前 | リポジトリルート |
| `post-add` | worktree 作成後 | worktree ディレクトリ |
| `pre-remove` | worktree 削除前 | worktree ディレクトリ |
| `post-remove` | worktree 削除後 | リポジトリルート |

### 3.2 フック環境変数

フック実行時に以下の環境変数がエクスポートされる:

| 変数 | 説明 |
|---|---|
| `GW_REPO_ROOT` | メインリポジトリルートの絶対パス |
| `GW_WORKTREE_PATH` | worktree の絶対パス（`pre-add` フックでは作成予定のパス。ディレクトリはまだ存在しない） |
| `GW_BRANCH` | ブランチ名 |

### 3.3 フック実行ルール

- フックファイルが存在しない場合: 何もせず成功扱い
- フックファイルが存在するが実行権限がない場合: エラー
- フックの stdout/stderr は親プロセスの stderr に流す
- 各フックは独立したサブプロセスで実行する

### 3.4 フック失敗時の動作

| フック | 失敗時の動作 | `--force` 時 |
|---|---|---|
| `pre-add` | **作成を中止**（終了コード 1） | - |
| `post-add` | 警告のみ（worktree は作成済み、終了コード 0） | - |
| `pre-remove` | **削除を中止**（終了コード 1） | 警告して削除を続行（終了コード 0） |
| `post-remove` | 警告のみ（worktree は削除済み、終了コード 0） | - |

### 3.5 フック実行タイミング

`pre-*` フックは、gw 内部の全ての前提条件チェック（引数検証、パス計算、起点 ref の解決等）が成功した後、対応する git コマンドの実行直前に呼び出される。前提条件チェックの失敗時にはフックは実行されない。

---

## 4. identifier 解決

`gw go` / `gw rm` で使用する identifier からworktree パスを逆引きする。

メインリポジトリで `git worktree list` を実行し、その結果を基に解決する。

**解決順序（最初にマッチしたものを返す）:**

1. **サニタイズ名によるパスマッチ**: `git worktree list` の結果から、パスが `<base_dir>/<sanitize(identifier)>` と一致する worktree を探す
2. **ブランチ名によるスキャン**: `git worktree list` の結果から、ベースディレクトリ内の worktree に絞り込み、ブランチ名が identifier と一致するかチェック
3. **すべて失敗**: エラー

---

## 5. 設定ファイル

メインリポジトリルートの `.gw/config` に設定を記述する。ファイルが存在しない場合はすべてデフォルト値を使用する。

### 5.1 設定キー

| キー | 説明 | デフォルト |
|---|---|---|
| `worktrees_dir` | worktree を格納するベースディレクトリ（絶対パスまたはリポジトリルートからの相対パス） | `../<repo-name>-worktrees/` |

**ファイルフォーマット:** TOML

**未知のキー:** 前方互換性のため、未知のキーは無視する。

**設定例:**

```toml
worktrees_dir = "../my-worktrees"
```

---

## 6. エラー処理・終了コード

### 6.1 終了コード

| コード | 意味 |
|---|---|
| `0` | 成功 |
| `1` | エラー |

### 6.2 エラーメッセージ

エラーメッセージは stderr に出力する。

**フォーマット:**

| 種別 | 形式 |
|---|---|
| エラー | `gw: error: <message>` |
| 警告 | `gw: warning: <message>` |

git コマンド由来のエラーは `gw: error:` でラップせず、git の出力をそのまま stderr に流す。

### 6.3 主要なエラーケース

| 状況 | 終了コード |
|---|---|
| git リポジトリ外での実行 | 1 |
| `.gw/` が既に存在する（`gw init` 時） | 1 |
| 不明なコマンド | 1 |
| worktree が見つからない（identifier 解決失敗） | 1 |
| worktree が既に存在する（サニタイズ衝突を含む） | 1 |
| 無効なブランチ名（サニタイズ後のバリデーション失敗） | 1 |
| 既存ブランチに対する `--from` の指定 | 1 |
| git コマンドの失敗 | 1 |

---

## 7. 将来の拡張（初回スコープ外）

- `gw list`: worktree 一覧表示
- git worktree サブコマンドのパススルー
- シェル補完
