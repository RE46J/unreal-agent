# 配布とリリース (Distribution)

**言語:** [English](../distribution.md) | 日本語

Unreal Agent の公式配布形式は **Linux amd64 向け共通キット**（CGO / Tree-sitter 有効）です。Claude と Codex の両方で同じキットを使用し、認証情報は外部 CLI が管理します。

---

## 1. ローカルチェックと CI

コミットを作成する前に、必ず `make prepush` を実行します。

```bash
make prepush
# 特定のベースリビジョン以降のコミット済み変更も含めて検査する場合:
make prepush BASE=<rev>
```

### 検査の内容

| 区分 | コマンド / ワークフロー | 検証内容 |
| --- | --- | --- |
| **コミット前チェック** | `make prepush` | `gofmt`、`go vet`、変更された Go パッケージと逆依存先のテスト（Go 以外の変更時は全 Go テスト）、`scripts/test_*.py`、および `benchmarks/harbor/` 変更時の Harbor テスト |
| **通常 CI** | [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) | Ubuntu 24.04 上での `make test check build`（全パッケージ `-race` 付き）、Python テスト、Harbor アダプタのビルド・Docker スモークテスト |
| **Fuzzing** | [`.github/workflows/fuzz.yml`](../../.github/workflows/fuzz.yml) | 定期・手動実行の Go ファジング（通常のテスト実行でも保存済みシードコーパスは毎回実行されます） |

---

## 2. スナップショットと共通キットのビルド手順

### 必要なツール（Linux amd64）

- [`go.mod`](../../go.mod) 指定バージョンの Go
- C コンパイラ（`CGO_ENABLED=1`）
- Python 3.12 以上
- GoReleaser `v2.18.2`

### 手順

1. **スナップショットスクリプトのテストと GoReleaser の準備:**
   ```bash
   make test-snapshot
   unreal_release_tools=$(mktemp -d /tmp/unreal-release-tools.XXXXXXXX)
   GOBIN="$unreal_release_tools" go install -ldflags='-X main.version=2.18.2' github.com/goreleaser/goreleaser/v2@v2.18.2
   "$unreal_release_tools/goreleaser" --version
   ```

2. **スナップショットのビルドと検証:**
   ```bash
   make snapshot GORELEASER="$unreal_release_tools/goreleaser"
   ```
   [`scripts/snapshot.py`](../../scripts/snapshot.py) は `dist/snapshots/snapshot-*/` に新しい専用ディレクトリを確保し、GoReleaser をスナップショットモード（公開なし）で実行します。生成されたアーカイブに対して、ファイル構成、実行権限、チェックサム、CGO/VCS メタデータ、`--help` 実行、およびパッケージ内の Host attach/detach テストを自動検証します。

3. **共通キット（Common Kit）のパッケージング:**
   `snapshot verified:` で表示されたアーカイブのフルパスと、`SNAPSHOT.json` に記録された40文字のコミットリビジョンを入力します。
   ```bash
   read -r -e -p 'Verified snapshot archive: ' unreal_snapshot
   read -r -p 'Full source revision: ' unreal_revision
   python3 scripts/common_development_bundle.py --snapshot "$unreal_snapshot" --revision "$unreal_revision"
   ```

### 生成される配布ファイル

[`scripts/common_development_bundle.py`](../../scripts/common_development_bundle.py) により、以下の2ファイルが出力されます。
- `unreal-agent-linux-amd64-REV.tar.gz`
- `unreal-agent-linux-amd64-REV.tar.gz.sha256`

（`REV` はソースリビジョンの先頭7文字です。アーカイブ内には4つのバイナリ、`KIT.json`、`SNAPSHOT.json`、`SHA256SUMS`、`README.md`、`LICENSE`、`install.sh`、`doctor.sh`、`development.sh` 等が含まれます。）

---

## 3. 同梱される実行ファイルと更新時の挙動

| 実行ファイル | 役割 |
| --- | --- |
| `unreal` | セッション作成・接続・再開を行うネイティブランチャー（共通キット導入時は `~/.local/bin/unreal` ラッパーから呼ばれます） |
| `unreal-agent` | Host の `serve` / `attach` および `subagent` コマンドインターフェース |
| `unreal-agent-runner` | 開発・評価用のヘッドレス JSON リクエスト / JSONL ランナー |
| `unreal-agent-auth` | 開発構成向けの認証管理 CLI |

共通キットを更新した後、既存のワークスペースで次回 `unreal <セッション名>` を起動すると、ラッパーがビルドやプロバイダプロファイルの変更を検知して同じインスタンスの Host を自動再起動し、**既存のセッションと設定をそのまま引き継ぎます**。インストール済みバージョンは「現在」「1つ前」「実行中プロセスが使用中のもの」を残して自動的に整理されます。

---

## 4. GitHub Release ワークフロー

[Release ワークフロー](../../.github/workflows/release.yml)は `TomyFovos/unreal-agent` リポジトリの `v*` タグに対して実行されます。

1. **前提チェック:** タグ付けされたコミットが `main` ブランチの履歴に含まれていることを確認し、そのコミットの CI が成功するのを待ちます。
2. **ビルドジョブ（読み取り専用権限）:** GoReleaser `v2.18.2` を導入し、スナップショットと共通キットをビルドしてチェックサムを検証します。
3. **公開ジョブ（`contents: write` 権限）:** GitHub Release を作成し、共通キットの `.tar.gz` と `.sha256` をアップロードします（タグ名に `-` を含む場合はプレリリース扱いになります）。

> **運用ルール:** `git push`、PR 作成・マージ、タグ作成、および Release 公開を行うには、対象のコミットに対するユーザーの事前の明示的な承認が必要です。

---

## 5. 配布前の受入確認チェックリスト

キットを配布する前には、以下の手順で実機確認を行います。

1. **ローカル検査と CI の確認:** 対象リビジョンで `make prepush` と CI が通過していることを確認します（ログイン依存でスキップされたテストと実際に通過したテストを区別します）。
2. **WSL / 実環境での導入・実推論テスト:** 本番のセッションデータを壊さない別のインストール先（テスト環境）に共通キットを導入し、TUI を起動して、実際のサブスクリプションでモデル呼び出しが1往復成功することを確認します。
3. **変更箇所の動作確認:** 再接続・再開、権限モードの保持、ファイル編集結果の実際のバイト列など、変更に関連するユーザーの操作経路を確認します。
4. **結果の報告:** 検証したキット名・リビジョン・環境・実推論の結果（および未検証の経路があればその旨）を報告したうえで、[インストール手順](install.md) とともにキットを案内します。
