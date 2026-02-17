# gw 実装計画

## 概要

`gw` は git worktree の薄いラッパー CLI ツール。パス自動計算とフック実行を核とする。
仕様: [SPECIFICATION.md](./SPECIFICATION.md)

---

## プロジェクト構成（予定）

```
cmd/gw/main.go          # エントリーポイント・CLI ルーティング
internal/
  git/git.go            # git 操作（リポジトリルート検出、デフォルトブランチ取得、ブランチ存在確認等）
  config/config.go      # .gw/config (TOML) のパース
  pathutil/pathutil.go  # パス計算（ベースディレクトリ、サニタイズ、最終パス）
  hook/hook.go          # フックシステム（実行・環境変数セット）
  resolve/resolve.go    # identifier 解決（サニタイズ名マッチ → ブランチ名スキャン）
  testutil/repo.go      # テストヘルパー（一時 git リポジトリ作成等）
  cmd/
    add.go              # gw add
    remove.go           # gw rm
    go.go               # gw go
    version.go          # gw version
```

---

## 実装フェーズ

依存関係の少ない基盤から積み上げる。各フェーズはテストを含む。

### Phase 1: CLI スケルトンと `gw version`

最小の動くバイナリを作る。

- [x] `cmd/gw/main.go`
  - [x] 引数パース・コマンドルーティング
  - [x] usage 出力（引数なし・不正コマンド → stderr、終了コード 1）
- [x] `internal/cmd/version.go`
  - [x] `gw version <VERSION>` を stdout に出力
  - バージョン文字列は `var version = "0.1.0"` としてソースに保持し、リリース時に `-ldflags '-X main.version=...'` で上書き
- [x] テスト

**依存:** なし

---

### Phase 2: git 操作ユーティリティ

以降のすべてのコマンドが依存する git 操作の基盤。テストヘルパーもここで構築する。

- [x] `internal/testutil/repo.go`: テストヘルパー（詳細はテスト戦略セクション参照）
- [x] `internal/git/git.go`
  - [x] `RepoRoot(dir string) (string, error)`: リポジトリルート検出
    - `dir` を cwd として `git rev-parse --git-common-dir` を実行
    - 結果が相対パスの場合（メインリポジトリでは `.git` が返る）、`dir` 基準で絶対パスに変換
    - `filepath.Dir()` でリポジトリルートを導出
    - worktree 内から実行された場合でもメインリポジトリのルートを返す
  - [x] `DefaultBranch(repoRoot string) (string, error)`: デフォルトブランチ取得（`origin/HEAD` → 未設定ならエラー）
  - [x] `BranchExists(repoRoot, branch string) (bool, error)`: ローカルブランチ存在確認
  - [x] `RemoteRefExists(repoRoot, ref string) (bool, error)`: リモートブランチ存在確認（`origin/<branch>`）。`gw add` で `--from` 省略時のデフォルト ref 判定に使用
  - [x] `RepoName(repoRoot string) string`: リポジトリ名取得（ベースディレクトリ名）
  - [x] `ListWorktrees(repoRoot string) ([]Worktree, error)`: `git worktree list --porcelain` をパースし、worktree 一覧を返す。identifier 解決（Phase 6）で使用
- [x] テスト（git 統合テスト）

**依存:** なし

---

### Phase 3: 設定ファイルとパス計算

worktree パスの計算に必要な設定読み込みとサニタイズ。

- [x] `internal/config/config.go`
  - [x] `.gw/config` (TOML) のパース
  - [x] `worktrees_dir` の読み込み（デフォルト: `../<repo-name>-worktrees/`）
  - ファイルが存在しない場合はデフォルト値を使用
  - 未知のキーは無視する（前方互換性のため）
- [x] `internal/pathutil/pathutil.go`
  - [x] ベースディレクトリ解決（設定 or デフォルト）。存在しない場合は自動的に作成する (§2.1)
  - [x] ブランチ名サニタイズ（`/` → `-`、先頭末尾 `-` 除去）
  - [x] バリデーション（空文字列、`.`、`..` はエラー）
  - [x] `ComputePath(baseDir, branch) string`: 最終パス計算（`<base_dir>/<sanitized-branch>`）
  - [x] `ValidatePath(path) error`: 衝突チェック（ディレクトリ既存ならエラー）
- [x] テスト

**依存:** Phase 2（リポジトリルート・リポジトリ名が必要）

---

### Phase 4: フックシステム

add / rm で利用するフック実行基盤。

- [x] `internal/hook/hook.go`
  - [x] フック実行（`.gw/hooks/<hook-name>`）
  - [x] フック不在 → 成功扱い
  - [x] 実行権限チェック（なければエラー）
  - [x] 環境変数エクスポート: `GW_REPO_ROOT`, `GW_WORKTREE_PATH`, `GW_BRANCH`
    - `pre-add` フックでは `GW_WORKTREE_PATH` は作成予定のパス（まだ存在しない）
  - [x] 実行場所（cwd）の制御
  - [x] stdout/stderr を親プロセスの stderr に流す
- [x] テスト

**依存:** なし（インターフェースは Phase 2 のリポジトリルート等を受け取る）

---

### Phase 5: `gw add`

コア機能。worktree の作成。

- [x] `internal/cmd/add.go`
  - [x] 引数パース: `<branch>`, `--from <ref>`
  - [x] 処理フロー:
    1. リポジトリルート検出
    2. worktree パス計算・衝突チェック（`ValidatePath` でディレクトリ既存ならエラー）
    3. ローカルブランチ存在確認・引数検証（既存ブランチ + `--from` → エラー）・起点 ref 解決（新規ブランチ + `--from` 省略時: `RemoteRefExists` で `origin/<デフォルトブランチ>` の存在を確認し、存在すればそれを使用、なければ `<デフォルトブランチ>` にフォールバック）
    4. `pre-add` フック実行（リポジトリルートで）
    5. `git worktree add` 実行
       - 新規: `git worktree add <path> -b <branch> <ref>`
       - 既存: `git worktree add <path> <branch>`
    6. `post-add` フック実行（worktree ディレクトリで）
    7. 作成先パスを stdout に出力
- [x] テスト

**依存:** Phase 2, 3, 4

---

### Phase 6: identifier 解決

`gw go` / `gw rm` が依存する逆引き機構。

- [x] `internal/resolve/resolve.go`
  - [x] `git.ListWorktrees()` を唯一のデータソースとして使用
  - [x] サニタイズ名パスマッチ: worktree 一覧から、パスが `<base_dir>/<sanitize(identifier)>` と一致するものを探す
  - [x] ブランチ名スキャン: worktree 一覧からベースディレクトリ内に絞り込み、ブランチ名が identifier と一致するものを探す
  - [x] すべて失敗 → エラー
- [x] テスト

**依存:** Phase 2（`git.ListWorktrees`）, Phase 3（パス計算・サニタイズ）

---

### Phase 7: `gw go`

worktree パスを stdout に出力するコマンド。

- [x] `internal/cmd/go.go`
  - [x] 処理フロー:
    1. リポジトリルート検出
    2. 設定読み込み・ベースディレクトリ解決
    3. identifier 解決（`git worktree list` ベース）
    4. 絶対パスを stdout に出力
- [x] テスト

**依存:** Phase 2, 3, 6

---

### Phase 8: `gw rm`

worktree の削除。フック連携あり。

- [x] `internal/cmd/remove.go`
  - [x] 引数パース: `<identifier>`, `--force`
  - [x] 処理フロー:
    1. identifier → worktree パス解決
    2. `pre-remove` フック実行（worktree ディレクトリで）
       - 失敗 + `--force` なし → エラー終了
       - 失敗 + `--force` あり → 警告して続行
    3. `git worktree remove [--force] <path>` 実行
    4. `post-remove` フック実行（リポジトリルートで）
- [x] テスト

**依存:** Phase 2, 4, 6

---

## テスト戦略

### テストの分類

| 分類 | 対象 | git リポジトリ | 実行速度 |
|------|------|---------------|---------|
| 単体テスト | 純粋なロジック（サニタイズ、config パース等） | 不要 | 高速 |
| git 統合テスト | git 操作を伴う機能 | 一時リポジトリを作成 | 中速 |
| E2E テスト | `gw` バイナリを実際に実行 | 一時リポジトリを作成 | 低速 |

### テストヘルパー (`internal/testutil/`)

git 統合テスト・E2E テストで共通利用する一時 git 環境のセットアップ。Phase 2 で構築する。

- [x] `internal/testutil/repo.go`
  - `t.TempDir()` 内に git リポジトリを作成（`git init` + 初回コミット）
  - bare リポジトリを origin として設定（remote ブランチのテスト用）
  - ブランチ作成ヘルパー
  - `.gw/config` 書き込みヘルパー
  - `.gw/hooks/` にフックスクリプト配置ヘルパー（実行権限付与込み）
  - worktree 作成ヘルパー（identifier 解決テスト用に既存 worktree が必要）
  - `t.Cleanup()` で自動クリーンアップ

### E2E テストのバイナリビルド

`TestMain` で `go build -o <tempdir>/gw ./cmd/gw` を実行し、ビルド済みバイナリのパスをパッケージ変数で共有する。各テストケースはこのバイナリを `exec.Command` で実行する。

### 各フェーズのテスト内容

テスト記述の凡例: `→` の左が条件/入力、右が期待結果。仕様参照は `§` で示す。

#### Phase 1: CLI スケルトン

| テスト | 分類 |
|--------|------|
| 引数なしで実行 → stderr に usage を出力、終了コード 1 (§1.5) | E2E |
| 不正コマンド `gw foo` → stderr に usage を出力、終了コード 1 (§1.5) | E2E |
| 不正コマンド `gw foo` → stderr のエラー出力が `gw: error: ...` フォーマットに従う (§6.2) | E2E |
| `gw version` → stdout に `gw version 0.1.0`、終了コード 0 (§1.4, §6.1) | E2E |

#### Phase 2: git 操作ユーティリティ

| テスト | 分類 |
|--------|------|
| メインリポジトリのルートで実行 → `git rev-parse --show-toplevel` と同じパスを返す | git 統合 |
| worktree 内で実行 → worktree のパスではなく、メインリポジトリのルートパスを返す (§冒頭リポジトリ検出) | git 統合 |
| git リポジトリ外で実行 → エラーを返す (§6.3) | git 統合 |
| `origin/HEAD` が設定済み → `git symbolic-ref refs/remotes/origin/HEAD` からブランチ名を抽出して返す（例: `refs/remotes/origin/main` → `main`） (§1.1) | git 統合 |
| `origin/HEAD` が未設定 → エラーを返す (§1.1) | git 統合 |
| 存在するローカルブランチ名を渡す → true を返す | git 統合 |
| 存在しないローカルブランチ名を渡す → false を返す | git 統合 |
| 存在するリモートブランチ `origin/<branch>` を渡す → true を返す | git 統合 |
| 存在しないリモートブランチ `origin/<branch>` を渡す → false を返す | git 統合 |

#### Phase 3: 設定ファイルとパス計算

| テスト | 分類 |
|--------|------|
| `.gw/config` が存在しない → `worktrees_dir` は `../<リポジトリルートのbasename>-worktrees/` (§2.1, §5.1) | 単体 |
| `.gw/config` に `worktrees_dir = "../my-trees"` → ベースディレクトリがリポジトリルートからの相対パスとして解決される (§2.1) | 単体 |
| `.gw/config` に `worktrees_dir = "/tmp/trees"` → ベースディレクトリが `/tmp/trees` になる (§2.1) | 単体 |
| サニタイズ: `feature/user-auth` → `feature-user-auth` (§2.2 変換例) | 単体 |
| サニタイズ: `feature/auth/login` → `feature-auth-login` (§2.2 変換例) | 単体 |
| サニタイズ: `/feature/` → 先頭・末尾のハイフンを除去して `feature` (§2.2 ルール2) | 単体 |
| サニタイズ結果が空文字列 → エラー (§2.2 バリデーション) | 単体 |
| サニタイズ結果が `.` → エラー (§2.2 バリデーション) | 単体 |
| サニタイズ結果が `..` → エラー (§2.2 バリデーション) | 単体 |
| 最終パスが `<base_dir>/<sanitized-branch>` 形式になる (§2.3) | 単体 |
| サニタイズ後のディレクトリが既に存在する → エラー (§2.2 衝突) | git 統合 |
| `.gw/config` に未知のキーがある → エラーにならずパースが成功する (§5) | 単体 |
| `.gw/config` が不正な TOML → パースエラー (§5) | 単体 |

#### Phase 4: フックシステム

| テスト | 分類 |
|--------|------|
| `.gw/hooks/<hook-name>` が存在しない → エラーなしで処理続行 (§3.3) | git 統合 |
| `.gw/hooks/<hook-name>` が存在し実行権限あり、exit 0 → エラーなしで処理続行 (§3.3) | git 統合 |
| `.gw/hooks/<hook-name>` が存在するが実行権限なし → エラー (§3.3) | git 統合 |
| フック内で `$GW_REPO_ROOT` がメインリポジトリルートの絶対パスと一致する (§3.2) | git 統合 |
| フック内で `$GW_WORKTREE_PATH` が worktree の絶対パスと一致する (§3.2) | git 統合 |
| フック内で `$GW_BRANCH` が対象ブランチ名と一致する (§3.2) | git 統合 |
| `pre-add` フックの cwd がリポジトリルートと一致する (§3.1) | git 統合 |
| `post-add` フックの cwd が worktree ディレクトリと一致する (§3.1) | git 統合 |
| `pre-remove` フックの cwd が worktree ディレクトリと一致する (§3.1) | git 統合 |
| `post-remove` フックの cwd がリポジトリルートと一致する (§3.1) | git 統合 |
| フックの stdout が親プロセスの stderr に出力される (§3.3) | git 統合 |
| フックの stderr が親プロセスの stderr に出力される (§3.3) | git 統合 |
| フックが非ゼロ終了コードを返す → 呼び出し元にエラーとして伝播する (§3.4) | git 統合 |

#### Phase 5: `gw add`

| テスト | 分類 |
|--------|------|
| 存在しないブランチ名を指定、`--from` 省略 → `git worktree add <path> -b <branch> origin/<デフォルトブランチ>` が実行され、stdout に `<base_dir>/<sanitized-branch>` の絶対パスが出力される (§1.1 処理フロー5, 7) | E2E |
| 存在しないブランチ名 + `--from v1.0` → `git worktree add <path> -b <branch> v1.0` が実行され、stdout に絶対パスが出力される (§1.1) | E2E |
| 存在しないブランチ名、`--from` 省略、`origin/<デフォルトブランチ>` が存在しない → `<デフォルトブランチ>` にフォールバック (§1.1 処理フロー3) | E2E |
| 既存ブランチ名を指定、`--from` なし → `git worktree add <path> <branch>` が実行され、stdout に絶対パスが出力される (§1.1 処理フロー5) | E2E |
| 既存ブランチ名 + `--from` 指定 → エラー、終了コード 1 (§1.1 処理フロー3, §6.3) | E2E |
| 既存ブランチ名 + `--from` 指定 → stderr のエラー出力が `gw: error: ...` フォーマットに従う (§6.2) | E2E |
| git リポジトリ外で `gw add` を実行 → エラー、終了コード 1 (§6.3) | E2E |
| worktree 内から `gw add` を実行 → メインリポジトリを正しく検出し、正常に worktree を作成 (§冒頭リポジトリ検出) | E2E |
| `pre-add` フックが非ゼロ終了 → worktree が作成されない、終了コード 1 (§3.4) | E2E |
| `post-add` フックが非ゼロ終了 → worktree は作成済み、stderr に警告、終了コード 0 (§3.4) | E2E |
| サニタイズ後のディレクトリが既に存在する → エラー、終了コード 1 (§2.2 衝突, §6.3) | E2E |
| `origin/HEAD` が未設定 + 新規ブランチ + `--from` 省略 → デフォルトブランチ取得失敗でエラー、終了コード 1 (§1.1) | E2E |
| `origin/HEAD` が未設定 + 既存ブランチ + `--from` なし → デフォルトブランチ不要のため正常に worktree を作成 (§1.1) | E2E |
| 対象ブランチが既に別の worktree でチェックアウト済み → git コマンド失敗でエラー、終了コード 1 (§6.3) | E2E |
| 既存ブランチ + `--from` 指定 → エラー、`pre-add` フックは実行されない (§1.1 処理フロー3) | E2E |

#### Phase 6: identifier 解決

| テスト | 分類 |
|--------|------|
| identifier `feature/foo` → `git worktree list` の結果に `<base_dir>/feature-foo` のパスを持つ worktree が存在する → そのパスを返す (§4 解決順序1) | git 統合 |
| サニタイズ名パスマッチなし → `git worktree list` のベースディレクトリ内 worktree からブランチ名が identifier と一致するものを返す (§4 解決順序2) | git 統合 |
| サニタイズ名パスマッチとブランチ名スキャンの両方に該当する → サニタイズ名パスマッチが優先される (§4 解決順序) | git 統合 |
| `<base_dir>/<sanitize(identifier)>` のディレクトリが存在するが `git worktree list` に含まれない → マッチしない (§4) | git 統合 |
| サニタイズ名パスもブランチ名スキャンも一致しない → エラー (§4 解決順序3, §6.3) | git 統合 |

#### Phase 7: `gw go`

| テスト | 分類 |
|--------|------|
| identifier が解決可能 → stdout に worktree の絶対パスを出力、終了コード 0 (§1.3) | E2E |
| identifier が解決不可 → stderr にエラー、終了コード 1 (§1.3, §6.3) | E2E |
| identifier が解決不可 → stderr のエラー出力が `gw: error: ...` フォーマットに従う (§6.2) | E2E |
| worktree 内から `gw go` を実行 → メインリポジトリを正しく検出し、正常にパスを出力 (§冒頭リポジトリ検出) | E2E |

#### Phase 8: `gw rm`

| テスト | 分類 |
|--------|------|
| identifier が解決可能 → `git worktree remove <path>` が実行され、worktree ディレクトリが削除される、終了コード 0 (§1.2) | E2E |
| `--force` 付き + worktree に未コミット変更あり → `git worktree remove --force <path>` が実行され、worktree が削除される (§1.2) | E2E |
| `pre-remove` フックが非ゼロ終了 + `--force` なし → worktree が削除されない、終了コード 1 (§3.4) | E2E |
| `pre-remove` フックが非ゼロ終了 + `--force` あり → stderr に警告、worktree は削除される (§1.2, §3.4) | E2E |
| `post-remove` フックが非ゼロ終了 → stderr に警告、worktree は削除済み、終了コード 0 (§3.4) | E2E |
| identifier が解決不可 → stderr にエラー、終了コード 1 (§6.3) | E2E |
| identifier が解決不可 → stderr のエラー出力が `gw: error: ...` フォーマットに従う (§6.2) | E2E |
| 削除後、ブランチは残存する（`git worktree remove` 準拠） (§1.2) | E2E |
| 正常終了時に stdout が空であること (§1.2) | E2E |

---

## 外部依存

| ライブラリ | 用途 |
|-----------|------|
| TOML パーサー（例: `github.com/BurntSushi/toml`） | `.gw/config` のパース |

標準ライブラリ (`os/exec`, `path/filepath`, `strings` 等) で大部分をカバーする。

---

## 設計判断メモ

- **CLI フレームワークは使わない**: コマンドが少なく（add, rm, go, version）、`os.Args` の手動パースで十分
- **git 操作は `os/exec` で実行**: ライブラリ（go-git 等）は不要。薄いラッパーの方針に合致
- **git コマンドの出力制御**: すべての git コマンド実行で `cmd.Stdout = os.Stderr`, `cmd.Stderr = os.Stderr` とし、gw の stdout をデータ出力専用に保つ
- **エラーメッセージフォーマット**: `gw: error: <message>` / `gw: warning: <message>` の形式で統一する。ただし git コマンド由来のエラーはラップせず、git の出力をそのまま stderr に流す（git 自体の動作に準拠）
- **バージョン管理**: ソースに `var version = "0.1.0"` を保持し、リリースビルド時に `-ldflags '-X main.version=...'` で上書き
- **未知の設定キー**: 前方互換性のため無視する（SPEC 追記予定）
- **テスト戦略**: 詳細はテスト戦略セクション参照。純粋ロジックは単体テスト、git 操作は一時リポジトリでの統合テスト、コマンド層はバイナリ実行の E2E テストの3層構成
