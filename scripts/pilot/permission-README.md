# @SANDBOX@ — Permission Mode 受入キット（Gate D / E）

Linux amd64（GOAMD64=v1、CGO / Tree-sitter 有効）の snapshot です。
Permission Modes は Unreal 内の認可であり、OS sandbox を追加するものではありません。
既存の Codex / Claude CLI、PATH、認証ファイル、通常の config / Host / Session は変更せず、
専用の private 配置・合成 Git workspace で試験します。

## 1. 転送とセットアップ

1. **Windows PowerShell**（archive と `.sha256` のあるフォルダ）:

   ```powershell
   sbx cp ./@ARCHIVE@ @SANDBOX@:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 @SANDBOX@:/tmp/@ARCHIVE@.sha256
   ```

2. **Sandbox Bash**:

   ```sh
   set -eu
   cd /tmp
   sha256sum -c @ARCHIVE@.sha256
   permission_unpack=$(mktemp -d /tmp/unreal-permission.XXXXXXXX)
   tar -xzf @ARCHIVE@ -C "$permission_unpack"
   bash "$permission_unpack/@KIT_ROOT@/gate.sh" setup --model @MODEL@ --effort high@OPT_IN@
   ```

   Python 3、Bash、Git、standard coreutils / ldd / tar、glibc @MIN_GLIBC@ 以上が必要です。
   OS packages は導入しません。setup は環境・認証経路を検査し、専用 workspace と設定を作り、
   実推論は送りません。表示された `LAUNCH:` の専用 `unreal` を使います。
   起動時は短い Quick Start を表示し、Enter 待ちはせず、preflight 後に TUI へ接続します。

   devlore 向けに展開される `--policy-audited --trust-managed-policy` は、従来は
   管理 policy の監査・trust を宣言する入力でした。現行 helper は両フラグを受理しますが、
   `claude_preflight` はその真偽で監査済みと判定せず、サブスク login status を検査します。
   これらのフラグも、read-only baseline や `PermissionModes.Maximum` も、操作の承認ではありません。

   **現行コードの制約:** devlore 用 Gate 設定生成には、現在の runtime loader にない
   旧 `Runtime.ClaudeCode` フィールドが残っています。そのため devlore の Gate 起動成功は
   保証できません。日常利用の Claude は共通キットの `/model` を使います。

3. **以降の手順で使う変数**（上と同じ Bash）:

   ```sh
   permission_gate="$HOME/.local/state/unreal-agent-pilot/@SANDBOX@/permissions-@REVISION_SHORT@"
   permission_package="$HOME/.local/opt/unreal-agent-pilot/@SANDBOX@/@REVISION_SHORT@-$(sha256sum "$permission_unpack/@KIT_ROOT@/SHA256SUMS" | cut -c1-16)"
   # Host 起動・認証 probe・推論なしで、同梱手順だけを表示
   bash "$permission_package/gate.sh" readme
   ```

config は 0600、専用 directory は 0700 です。通常の `~/.local/bin/unreal` や shell 設定は
置換しません。試験の state / workspace は証拠として保持されます。

## 2. Gate D — 履歴と Host lifetime

1. `"$permission_gate/unreal" gate-d` を開き、Read Only、`/model`、`/view orchestration` を確認します。
2. 別 Bash で `bash "$permission_package/gate.sh" prompt d` を表示し、TUI へ1回送信します。
   Final を待って Ctrl+D で detach します。Host は生存します。
3. 同じ `"$permission_gate/unreal" gate-d` で再接続し、履歴を確認して detach します。
4. 以下で checkpoint・専用 Host 停止・再起動後の状態を検証します。

   ```sh
   bash "$permission_package/gate.sh" verify d
   bash "$permission_package/gate.sh" checkpoint gate-d
   bash "$permission_package/gate.sh" stop
   "$permission_gate/unreal" gate-d
   # 履歴を確認し、再送せず Ctrl+D
   bash "$permission_package/gate.sh" resume-check gate-d
   ```

stop は Unix peer UID / PID、実行ファイル、serve 引数、専用 config / store / socket / auth の
パス、process start time を照合し、pidfd でその Host だけへ SIGTERM を送ります。
照合できなければ停止しません。resume-check は canonical 履歴 prefix、terminal receipts、
Operation の重複がないこと、最後に選択した Permission mode の維持を検査します。

## 3. Gate E — 実 Operation と人間の承認

`"$permission_gate/unreal" gate-e` を開きます。新 Session は Read Only です。
各段階の prompt を `bash "$permission_package/gate.sh" prompt STAGE` で表示して1回送信し、
Final 後に Ctrl+D で detach して `verify STAGE` を実行します。

| STAGE | 人間の操作と合格条件 |
| --- | --- |
| `read` | `read` / `grep` / `glob` の実 receipt と結果を確認。 |
| `deny` | Read Only で実際に `write` と `Bash` を要求し、typed denial と出力ファイル不在を確認。モデルの拒否文だけでは不可。 |
| `ask` | `/permissions` → Ask for Approval。最初の write は Allow Once、次は Deny。`once.txt` だけ生成。 |
| `work` | `/permissions` → Full Access。`sample.txt` を native write → read → edit → read → Bash で検証。 |
| `full` | Full Access で正確な `python3 -m unittest -v` の失敗 → code edit / `work.txt` 作成 → 同じテストの成功を確認。 |

Full Access は OS 隔離ではありません。許可された Bash は OS ユーザーの filesystem / network /
environment / shared mounts にアクセスできます。試験 prompt の workspace 指定は OS の隔離ではありません。
選択画面の説明を読んでモードを選びます。別画面の Full Access 確認はありません。

**native write / edit は Session workspace 内に限定**されます。`sample.txt` などの相対パスを
使い、別の `/tmp` repo を作らないでください。新規 write の `expected:{exists:false}` に
 digest は不要ですが、親 directory は存在する必要があります。edit は read が返した whole-file
revision を使います。`workspace_boundary`、`parent_missing`、`invalid_revision`、`revision_conflict`
などの reason で原因を区別し、`code=invalid` だけで Permission 拒否と判断しません。

```sh
bash "$permission_package/gate.sh" verify read
bash "$permission_package/gate.sh" verify deny
bash "$permission_package/gate.sh" verify ask
bash "$permission_package/gate.sh" verify work
bash "$permission_package/gate.sh" verify full

# 最後に選んだ mode と履歴が Host 再起動後も維持されることを確認
bash "$permission_package/gate.sh" checkpoint gate-e
bash "$permission_package/gate.sh" stop
"$permission_gate/unreal" gate-e
# mode と履歴を確認して Ctrl+D
bash "$permission_package/gate.sh" resume-check gate-e
"$permission_gate/unreal" gate-e
# `gate.sh prompt restart` を1回送信し、Final 後に Ctrl+D
bash "$permission_package/gate.sh" verify restart
```

`verify` は canonical decision / Operation、実際の test command と exit code、生成物を検査します。
Operation ID の重複、LLM の文章だけの成功、別コマンドへの置換、受入テストの改変は合格にしません。
Full Access の取消は `/permissions` → Read Only。モード変更は保留承認と実行権限を取り消しますが、
完了した副作用の rollback は保証しません。

## 4. 診断と復帰

`provider: remote_failure` は tool の失敗とは別です。`stage` は通信の段階、`status` は HTTP status、
`reason` は安全な分類、`retryable` はモデルリクエストの再試行可否です。HTTP error と HTTP 200 内の
stream error を区別します。成功済み tool receipt の副作用は完了しており、推論 retry は
Operation の再実行許可ではありません。`/resume` は canonical 履歴と完了 Operation を再利用します。

失敗時は detach し、次の読み取り専用診断を使います。

```sh
bash "$permission_package/gate.sh" diagnose gate-e
```

診断は安全な分類と receipt 件数を表示し、本文・arguments・ID・環境・認証 header は表示しません。
`sample_state=expected_after` は専用 `sample.txt` が正確に `after` と改行になっていることを示します。
旧 binary が記録していない HTTP status は復元できません。失敗 Session を合格へ読み替えず、
既存履歴を保持してください。詳細は `/analyze` の Errors で確認できます。

Ctrl+D で detach し、必要なら `bash "$permission_package/gate.sh" stop` で専用 Host だけを停止します。
既存の `codex` / `claude` / 通常の `unreal` は従来どおり利用できます。認証や Session の削除・移行は不要です。
合成試験の成功は Sandbox 実機の実推論受入とは別です。
