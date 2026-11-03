# 非同期サブエージェント (`harness/subagent`)

**言語:** [English](../../../../harness/subagent/README.md) | 日本語

`harness/subagent` パッケージは、非同期で動作する子エージェントのサブプロセス（`unreal-agent child --stdio`）、親子間の双方向メッセージ通信、および永続化・リカバリを管理します。

---

## 1. 親 Host と子プロセスの構成

### 親 Host 側の組み込み

1. `host.SessionFromContext(ctx)` から親 Session のオーナーを取得します。
2. `subagent.NewManager(ctx, Config{Owner: owner, Directory: sessionDirectory, Binary: absoluteAgentBinary, Arguments: []string{"child", "--stdio"}, Templates: allowedTemplates})` を生成し、`LocalOperationManager` に登録します。
3. `tool/subagent.Extensions(owner.ID, templates, false)` をツールレジストリに追加し、親ランタイムの終了時にハンドラの `Close` を呼びます。

- **テンプレートによる安全な制限:** 実行バイナリ、引数、プロバイダ・プロファイル選択、および子に許可する権限は Host 側の信頼されたテンプレート設定（`Subagents`）で固定されます。モデルが指定できるのは登録済みテンプレート名とタスク文字列のみです。
- **権限の制限と共有ロック:** 子のツール権限は常に親ポリシーの部分集合になります。また、親と同じ `MutationStateDirectory` と `ProjectInstructionSnapshot`（`AGENTS.md` のスナップショット）を引き継ぎ、ファイルロックの整合性を保ちます。

### 子プロセス側 (`unreal-agent child --stdio`)

`subagent.Serve(ctx, os.Stdin, os.Stdout, childFactory)` を実行し、`Lifecycle="child"` の子 Host を開きます。子には通常の制限付きファイル操作ツールに加え、親への連絡用 `SendParent` と完了報告用 `Finish` ツールのみが提供されます。

---

## 2. 通信プロトコル・永続化・完了判定

- **決定論的な Child ID:** 親 Session ID と `SubagentStart` の公開 Operation ID から一意に導出されます。
- **フレーム付き JSON 通信:** 標準入出力パイプ上で、4バイトのビッグエンディアン長プレフィックス＋バージョン付き JSON（最大 256 KiB）で通信します。
- **永続化後の ACK 応答 (`InputPeer`):** 親子間のメッセージは正規の `InputPeer` レコードとして `AppendInput` がコミットされた後にのみ ACK を返します。同じペイロードでの再送は元の Receipt を返します。
- **明示的な `Finish` 完了契約:** 通常のモデルターンが終わっただけでは子は「待機状態」となり、完了にはなりません。他の Operation が残っていない状態で明示的に `Finish` ツールが呼ばれたときのみ、正規の `HostRecord(kind=finish)` と親 Operation の結果ハンドルとして完了が記録されます。
- **段階的なキャンセル処理:** キャンセル時は制御入力と `canceling` チェックポイントを保存したうえで通常停止を要求し、2秒以内に終了しない場合はプロセスグループごと強制終了します。

---

## 3. オペレーター制御 API

- `ReadChild` — 子の正規履歴ページ、Operation 一覧、`ChildConfig`、および `Finish` 状態を読み取ります。
- `SteerChild` — 稼働中の子へ追加指示を送ります（パイプへ直接書き込まず、親の永続的な operation-intent 入力として記録されます）。
- `CancelChild` — 稼働中の子をキャンセルします。
- `ResumeChild` — 稼働中の親 Host が未完了の子を所有・復元していることを確認します（二重にプロセスを起動することはありません）。
