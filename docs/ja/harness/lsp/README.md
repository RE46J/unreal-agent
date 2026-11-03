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
- Session handler を閉じてから Session writer lock を解放し、workspace Host の終了時に `lsp.Manager` を閉じます。
- マルチセッション環境では `agentrunner.NewLanguageTools` の `Wrap` / `Close` を使って Session ツールファクトリをラップできます。

Manager の policy と各リクエストの policy の積集合を使います。server 起動には非制限の process/filesystem/network 権限が必要で、制限付き process policy は `unsupported` になります。

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
- **バージョン付き同期:** ファイルの SHA-256 ダイジェストが変化した際に `didOpen` / 全文 `didChange` を送り、ドキュメントバージョンが一致する診断結果（Diagnostics）のみを返します（対応する server では pull diagnostics を使用）。

server は open/close 通知と full または incremental document change に対応する必要があります。意味解析リクエスト中にファイルが変われば `stale` を返します。

---

## 3. 安全なワークスペース編集 (`rename` / `apply_code_action`)

LSP によるファイル変更はすべて共通の `mutation.Service` を通じて行われます。

1. **スナップショットの取得と同期:** `rename` や `code_actions` の実行前に、許可されたワークスペーススナップショットを取得して対象テキストを同期します。
2. **事前検証:** ワークスペース外の URI、Git メタデータ、重複する編集範囲、バージョン不一致、およびファイル編集以外のコマンド実行（`executeCommand` やファイルの削除・移動）を拒否します。
3. **Mutation の実行:** すべての変更対象ファイルの事前 SHA-256 ダイジェスト（新規作成時は不在条件）と安定した `Session ID + Operation ID` を `Mutation.Execute` に渡し、ロックとプリフライト検査を行ってから書き換えます。
4. **クラッシュリカバリ:** 編集途中で中断された場合は、言語サーバーを起動する前に永続化された Mutation の証跡（receipt）を確認し、確定結果または `indeterminate`（結果不明）を返します（二重書き込みは行いません）。

`CreateFile` は、対象が不在で、overwrite/ignore-if-exists を指定せず、親ディレクトリが存在する場合のみ対応します。annotation 付き編集、未観測の既存対象、未解決・無効な action、command 実行を要する action は非対応です。発見した action ID は次の探索または server 世代変更で失効し、適用時に消費します。適用時にも保存した revision と文書 version を再検証します。変更後の server 同期に失敗しても適用済みの結果に再同期を促す説明を添え、ファイルを rollback したり無条件で再実行したりしません。

---

## 4. リクエストの寿命と結果サイズ

リクエストは実行 context の cancel/deadline に従います。現在の設定には固定のリソース上限やリクエストごとの既定 timeout はありません。結果は JSON で返し、従来記載されていたサイズによる拒否や切り詰めは行いません。
