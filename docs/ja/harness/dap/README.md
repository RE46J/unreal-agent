# DAP デバッガランタイム (`harness/dap`)

**言語:** [English](../../../../harness/dap/README.md) | 日本語

`harness/dap` は Debug Adapter Protocol (DAP) のサブプロセスを管理し、エージェントに構造化されたデバッグ操作ツールを提供します。

プロトコル仕様: [Debug Adapter Protocol Overview](https://microsoft.github.io/debug-adapter-protocol/overview)

---

## 1. Host への組み込みとランナー設定

### Host 構成

1. Host の権限コンテキストと固定の `AdapterConfig` リストを渡して `dap.NewManager` を生成します。
2. `operation.RemoteJobHandler` として登録し、ツールレジストリに `tool/dap` の `Definition` と `Translator`（`New(manager.Generation())`）を追加します。
3. マルチセッション環境では `agentrunner.DebugTools` で `ToolFactory` をラップすることで、Session ごとに独立したデバッガ世代（generation）が割り当てられ、Session 終了時に確実にクリーンアップされます。

### ランナー用の環境変数 (`UNREAL_HARNESS_DAP_ADAPTERS`)

信頼されたアダプタ設定を JSON 配列で指定します。

```json
[{"id":"lldb","path":"/opt/bin/lldb-dap","directory":"/workspace","arguments":[],"allowed_attach_pids":[]}]
```

- **信頼された設定とモデル引数の分離:** アダプタの実行パス、引数、`launch_fields`、`attach_fields`、およびアタッチ許可 PID（`allowed_attach_pids`）は Host 側の固定設定からのみ読み込まれ、モデルのリクエストからは指定できません。
- **セッションメタデータ:** 生の設定 JSON ではなく、そのダイジェスト値のみが Session メタデータに記録されます。

---

## 2. 対応コマンドとリソース上限

### 利用可能なコマンド

`launch`, `attach`, `setBreakpoints`, `threads`, `stackTrace`, `scopes`, `variables`, `evaluate`, `continue`, `pause`, `next`, `stepIn`, `stepOut`, `inspect`, `disconnect`

- **停止エポック (`stop_epoch`) の検証:** フレームや変数の参照には `stop_epoch` が含まれ、プログラムの実行を再開した後は無効になります。
- **再起動時の自動再実行なし:** 計画には論理セッション ID と Host 世代（`generation`）が記録されます。Host 再起動時には前世代の計画はすべて `expired`（期限切れ）となり、デバッグ対象を勝手に再起動・再アタッチ・ステップ実行することはありません。

### リクエストの寿命と結果サイズ

リクエストは実行 context の cancel/deadline に従い、対応する操作では `Count` で要求・返却件数を指定します。wire frame は正の `Content-Length` を検査します。

---

## 3. 権限とプロセスの終了処理

- **必要な権限:** サブプロセス起動のため、明示的な非制限（unrestricted）プロセス・ファイル・ネットワーク権限が必要です。加えて `attach` には `allowed_attach_pids` の許可、`evaluate` には `DAP.Evaluate` 権限が必要です。
- **起動対象とアタッチ対象の安全な切り離し:**
  - **新規起動したプロセス (`launch`):** Manager の通常 close 時に `terminateDebuggee=true` で切断し、アダプタのプロセスグループを終了します。
  - **既存プロセスへのアタッチ (`attach`):** Manager の通常 close 時は `terminateDebuggee=false` で切断し、アタッチ先の PID を直接強制終了することはありません。
  - **明示的な `disconnect` コマンド:** `launch` / `attach` にかかわらずリクエストの `Terminate` 値を `terminateDebuggee` に使用します（Manager の通常 close とは動作が異なります）。
  - **異常終了時の注意点:** アダプタや Host の異常終了時、アタッチ先からの detach 挙動はアダプタ実装に依存するため保証できません。また、PID は起動時の許可リスト値であり、再起動や PID 再利用を超えて同一プロセスを示す永続的な ID ではありません。
- **非対応の逆方向リクエスト:** `runInTerminal` や `startDebugging` などのアダプタからの reverse request には対応していません。
