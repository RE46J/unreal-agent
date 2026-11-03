# @SANDBOX@ — Permission Mode 受入試験キット (Gate D / E)

**言語:** [English](../../../../scripts/pilot/permission-README.md) | 日本語

Linux amd64（`GOAMD64=v1` / CGO / Tree-sitter 有効）向けの Permission Mode 受入検証用スナップショットです。

- **既存の環境を変更しない独立配置:** 既存の Codex / Claude CLI、`PATH`、認証情報、通常の設定・Host・過去セッションは一切変更せず、専用のプライベートディレクトリへ導入します。
- **OS サンドボックスではありません:** Permission Mode はハーネス内の認可を制御するものであり、OS sandbox（OS レベルの隔離）を追加するものではありません。

---

## 1. 転送とセットアップ

1. **Windows 側（アーカイブと `.sha256` があるフォルダで実行）:**

   ```text
   sbx cp ./@ARCHIVE@ @SANDBOX@:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 @SANDBOX@:/tmp/@ARCHIVE@.sha256
   ```

2. **`@SANDBOX@` 内の Bash で実行:**

   ```sh
   set -eu
   cd /tmp
   sha256sum -c @ARCHIVE@.sha256
   permission_unpack=$(mktemp -d /tmp/unreal-permission.XXXXXXXX)
   tar -xzf @ARCHIVE@ -C "$permission_unpack"
   bash "$permission_unpack/@KIT_ROOT@/gate.sh" setup --model @MODEL@ --effort high@OPT_IN@
   ```

   Python 3、Bash、Git、標準 coreutils / `ldd` / `tar`、および glibc @MIN_GLIBC@ 以上が必要です。`setup` は環境検査と専用の一時 Git ワークスペースの準備のみを行い、実推論は送信しません。

3. **以降の操作で使うシェル変数の設定:**

   ```sh
   permission_gate="$HOME/.local/state/unreal-agent-pilot/@SANDBOX@/permissions-@REVISION_SHORT@"
   permission_package="$HOME/.local/opt/unreal-agent-pilot/@SANDBOX@/@REVISION_SHORT@-$(sha256sum "$permission_unpack/@KIT_ROOT@/SHA256SUMS" | cut -c1-16)"
   # 手順書（README）だけを再表示する場合（Host 起動や推論なし）:
   bash "$permission_package/gate.sh" readme
   ```

---

## 2. Gate D — 履歴の保持と Host ライフサイクルの検証

1. `"$permission_gate/unreal" gate-d` で新規セッションを開き、ヘッダーが `Read Only` であること、`/model`、および `/view orchestration` が使えることを確認します。
2. 別の Bash で `bash "$permission_package/gate.sh" prompt d` を表示して TUI へ1回送信し、応答完了後に `Ctrl+D` でデタッチします。
3. 再度 `"$permission_gate/unreal" gate-d` で再接続して履歴を確認し、`Ctrl+D` でデタッチします。
4. 以下のコマンドで検証・Host 停止・再開チェックを実行します。

   ```sh
   bash "$permission_package/gate.sh" verify d
   bash "$permission_package/gate.sh" checkpoint gate-d
   bash "$permission_package/gate.sh" stop
   "$permission_gate/unreal" gate-d
   # 履歴確認後に Ctrl+D でデタッチ
   bash "$permission_package/gate.sh" resume-check gate-d
   ```

---

## 3. Gate E — 実 Operation と人間による承認の検証

`"$permission_gate/unreal" gate-e` を開きます（初期状態は `Read Only`）。各 STAGE のプロンプトを `bash "$permission_package/gate.sh" prompt STAGE` で表示し、順番に1回ずつ送信して応答完了後に `Ctrl+D` でデタッチし、`verify STAGE` を実行します。

| STAGE | 操作と合格条件 |
| --- | --- |
| `read` | `read` / `grep` / `glob` の実 Receipt と結果を確認。 |
| `deny` | `Read Only` のまま `write` と `Bash` を要求。型付きの拒否（Denial）が記録され、ファイルが生成されないこと。 |
| `ask` | `/permissions` で `Ask for Approval` に変更。最初の `write` を Allow（`once.txt` 生成）、次の `write` を Deny。 |
| `work` | `/permissions` で `Full Access` に変更（警告画面を確認して承認）。`sample.txt` に対して `write` → `read` → `edit` → `read` → `Bash` 検証を実行。 |
| `full` | `Full Access` のまま、Bash で失敗する `python3 -m unittest -v` を観測後、コード修正と `work.txt` 作成を行い、再テストで PASS を確認。 |

### 各段階の検証コマンド

```sh
bash "$permission_package/gate.sh" verify read
bash "$permission_package/gate.sh" verify deny
bash "$permission_package/gate.sh" verify ask
bash "$permission_package/gate.sh" verify work
bash "$permission_package/gate.sh" verify full

# Host 再起動後も Permission Mode が維持されることを確認:
bash "$permission_package/gate.sh" checkpoint gate-e
bash "$permission_package/gate.sh" stop
"$permission_gate/unreal" gate-e
# モードと履歴を確認して Ctrl+D でデタッチ
bash "$permission_package/gate.sh" resume-check gate-e
"$permission_gate/unreal" gate-e
# prompt restart を1回送信し、完了を待ってデタッチ
bash "$permission_package/gate.sh" verify restart
```

---

## 4. 失敗時の診断と終了手順

テスト中にプロバイダ通信エラー等が発生した場合は、デタッチしてから以下の読み取り専用コマンドで安全な分類結果と Receipt 件数を確認できます（会話本文や認証情報は表示されません）。

```sh
bash "$permission_package/gate.sh" diagnose gate-e
```

終了時は `Ctrl+D` でデタッチし、必要に応じて `bash "$permission_package/gate.sh" stop` で検証用 Host を停止してください。既存の `codex` / `claude` / 通常版 `unreal` はそのまま利用できます。
