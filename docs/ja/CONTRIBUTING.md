# コントリビューションガイド (Contributing)

**言語:** [English](../../CONTRIBUTING.md) | 日本語

本リポジトリは [unreallabsai/unreal-agent](https://github.com/unreallabsai/unreal-agent) のフォークであり、Claude と Codex を使った日常開発向けにメンテナンスされています。

- **Issue:** 質問、バグ報告、機能要望は GitHub Issues で歓迎しています。
- **Pull Request:** 現在、外部からの Pull Request のレビューは行っていません。

## 必要な開発環境

- **Go:** [`go.mod`](../../go.mod) に記載されているバージョン
- **C コンパイラ (CGO):** Tree-sitter AST ツールに必要（Linux/WSL では `gcc`、macOS では Xcode Command Line Tools）
- **Python 3:** Python 3.12 以上（ビルド・スナップショット・検証スクリプトで使用）
- **Git**

配布キット（[Distribution](distribution.md)）のターゲット環境は **Linux amd64** です。

## ビルドとテストの手順

### 1. バイナリのビルド

リポジトリのルートで以下を実行します。

```bash
make build
```

`bin/` ディレクトリに以下の4つの実行ファイルが生成されます。
- `bin/unreal`（`unreal-agent` へのシンボリックリンク。セッションの起動・接続用ランチャー）
- `bin/unreal-agent`（Host サーバー、attach クライアント、subagent CLI）
- `bin/unreal-agent-runner`（評価・開発用ヘッドレスランナー）
- `bin/unreal-agent-auth`（開発用クレデンシャル管理 CLI）

### 2. 開発中の単体テスト実行

変更したパッケージに対応するテストを実行します。既存のアサーション、キャンセル挙動、エラー時の扱いを緩めないでください。また、スキップされたテスト（SKIP）を PASS とみなさないでください。

### 3. コミット前のチェック (`make prepush`)

コミットを作成する前に必ず以下を実行します。

```bash
make prepush
# 特定のリビジョン以降のコミット済み変更も含めて検査する場合:
make prepush BASE=<rev>
```

`make prepush` は以下を順番に実行します。
1. `gofmt -l cmd harness internal` と `go vet ./...`
2. 変更された Go パッケージおよび逆依存パッケージのテスト（Go 以外のファイルが変更された場合は `./...` 全件）
3. `scripts/` 配下の Python テスト（`test_*.py`）
4. `benchmarks/harbor/` が変更された場合は Harbor のチェックと単体テスト（[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) で指定されたバージョンの `uv` が必要）

### 4. マイルストーンでの全体 Race テスト

区切りのタイミングでは、データ競合検出（`-race`）を有効にして全テストを実行します。

```bash
go test -race -count=1 -p 1 ./...
```

CI では `main` への push、Pull Request、手動実行時に Ubuntu 上で `make test check build`（`-race` 込み）を実行し、別ジョブで Harbor、Docker スモークテスト、配布スクリプトのテストを実行します。

## 維持すべき設計原則

コードを変更する際は、以下の設計境界を維持してください。
- **Host と Session の所有権:** バックグラウンドの Host が Session の状態、書き込みロック、正規（canonical）履歴を所有する。
- **Operation と Receipt（受領証）:** Translator は純粋関数として保ち、副作用はすべて Operation Manager を通じて永続化された Receipt とともに実行する。
- **権限の強制:** 実行境界（`permission.Policy` および対話的権限モード）で確実に検査する。
- **プロバイダランタイム:** 別プロバイダや API キーへの自動フォールバックを行わない。
- **コンテキストエンジンとプロジェクト指示書:** 決定論的なコンテキスト構築とスナップショット規則を維持する。
- **LSP / DAP の分離:** 言語サーバーやデバッガのライフサイクルを推論や TUI 描画と混同しない。

## 関連ドキュメント

- [アーキテクチャ (Architecture)](architecture.md)
- [コンテキストと指示書 (Context)](context.md)
- [ツールとファイル変更 (Tools)](tools.md)
- [配布とリリース (Distribution)](distribution.md)
- [ヘッドレスランナー (Runner)](cmd/unreal-agent-runner/README.md)
- [Harbor ベンチマークアダプタ (Harbor)](benchmarks/harbor/README.md)
