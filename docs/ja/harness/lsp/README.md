# 言語サーバー管理 (`harness/lsp`)

**言語:** [English](../../../../harness/lsp/README.md) | 日本語

`harness/lsp` パッケージは、LSP 3.17（Language Server Protocol）の stdio 通信・ライフサイクル管理と、共通の [Mutation サービス](../../../../harness/mutation/mutation.go)を用いた安全なコード編集機能を実装しています。

プロトコル仕様: [LSP 3.17 Specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)

---

## 1. Host への組み込みとランナー設定

### Host 構成

ワークスペースの所有ライフサイクルごとに、ネイティブ編集と同じ `mutation.Service` を共有する1つの `lsp.Manager` を作成します。
- 言語ごとに実行ファイルの絶対パス、引数、環境変数、対象拡張子を明示的に設定します（パスの自動探索や自動インストールは行いません）。
- 各 Session に `lsp.Definition()`、`lsp.Translator{}`、および `lsp.NewHandler(ctx, manager, sessionID)` を登録します。
- マルチセッション環境では `agentrunner.NewLanguageTools` の `Wrap` / `Close` を使って Session ツールファクトリをラップできます。

### ランナー用の環境変数 (`UNREAL_HARNESS_LSP_SERVERS`)

```json
[{"language":"go","path":"/opt/bin/gopls","arguments":["serve"],"extensions":[".go"]}]
```

サーバーが設定されており、かつ `disallowed_tools` に含まれていない場合にのみ、モデルに対して `LSP` ツールが公開されます。

---

## 2. ツール機能と文字位置エンコーディング

`LSP` ツールは `language`、`action`、およびアクション別の引数を受け取ります。
- **参照・解析アクション:** `definition`, `references`, `diagnostics`, `document_symbols`, `workspace_symbols`, `hover`, `code_actions`
- **編集アクション:** `rename`, `apply_code_action`
- **ライフサイクル制御:** `restart`, `shutdown`（直近に観測したプロセスの `generation` 指定が必要）

### ドキュメント同期と位置変換

- **0始まりの位置指定:** 既定は `UTF-16` ですが、`UTF-8` や `UTF-32` も指定可能で、Unicode コードポイントを分割せずにサーバー側のエンコーディングへ変換します。
- **バージョン付き同期:** ファイルの SHA-256 ダイジェストが変化した際に `didOpen` / 全文 `didChange` を送り、ドキュメントバージョンが一致する診断結果（Diagnostics）のみを返します。

---

## 3. 安全なワークスペース編集 (`rename` / `apply_code_action`)

LSP によるファイル変更はすべて共通の `mutation.Service` を通じて行われます。

1. **スナップショットの取得と同期:** `rename` や `code_actions` の実行前に、許可されたワークスペーススナップショットを取得して対象テキストを同期します。
2. **事前検証:** ワークスペース外の URI、Git メタデータ、重複する編集範囲、バージョン不一致、およびファイル編集以外のコマンド実行（`executeCommand` やファイルの削除・移動）を拒否します。
3. **Mutation の実行:** すべての変更対象ファイルの事前 SHA-256 ダイジェスト（新規作成時は不在条件）と安定した `Session ID + Operation ID` を `Mutation.Execute` に渡し、ロックとプリフライト検査を行ってから書き換えます。
4. **クラッシュリカバリ:** 編集途中で中断された場合は、言語サーバーを起動する前に永続化された Mutation の証跡（receipt）を確認し、確定結果または `indeterminate`（結果不明）を返します（二重書き込みは行いません）。

---

## 4. リソース上限

- **プロトコルメッセージ:** 本文最大 4 MiB、ヘッダー最大 8 KiB
- **結果 JSON サイズ:** 最大 256 KiB（超過時は不正な切り詰め JSON ではなく明示的な `limit` エラーを返します）
- **同期ドキュメント上限:** 最大 2,048 ファイル・合計 16 MiB
- **その他上限:** 言語サーバー最大 32 個、コードアクション候補 128 件、キャッシュ診断 64 ファイル、タイムアウト既定 30 秒
