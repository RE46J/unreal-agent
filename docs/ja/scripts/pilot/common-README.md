# Unreal Agent 共通開発キット

**言語:** [English](../../../../scripts/pilot/common-README.md) | 日本語

**Codex か Claude のどちらか一方が使えれば始められます。** 同じ `/model` メニューから接続先・モデル・effort を選択できます。API キーは使用しません。

`asp` / `devlore` を含む Linux amd64 環境で同じキットを使用します。glibc @MIN_GLIBC@ 以上、Python 3、Bash、Git が必要です（ローカル snapshot: `@REVISION@`、認証情報は非同梱）。

---

## 1. 接続先のログイン準備

Unreal を実行する Linux 環境で、少なくとも片方のプロバイダにログインします。

- **Codex:** 公式の Codex CLI をインストールし、Bash で `codex login` を実行します（Docker Sandboxes のホスト管理 OAuth 設定がある場合はそのまま利用できます）。
- **Claude:** 公式の Claude Code をインストールし、Bash で `claude` を起動して `/login` を実行し、Claude サブスクリプションでログインします。

Unreal 内の `/login` でもこの手順を確認できます。後からもう片方にログインした場合は、Unreal を閉じずに `/model refresh`（または `/model` でその接続先を選択）するだけで再検出されます。

---

## 2. インストール手順

### Windows PowerShell から Sandbox へ転送する場合（`sbx` 利用環境）

```powershell
sbx cp .\@ARCHIVE@ asp:/tmp/@ARCHIVE@
sbx cp .\@ARCHIVE@.sha256 asp:/tmp/@ARCHIVE@.sha256
sbx cp .\@ARCHIVE@ devlore:/tmp/@ARCHIVE@
sbx cp .\@ARCHIVE@.sha256 devlore:/tmp/@ARCHIVE@.sha256
```

### Linux / Sandbox 内の Bash でインストール

```bash
cd /tmp
archive='@ARCHIVE@'
test -f "$archive" && test -f "$archive.sha256" && sha256sum -c "$archive.sha256" &&
kit=$(mktemp -d "$HOME/unreal-common.XXXXXXXX") &&
tar -xzf "$archive" -C "$kit" &&
cd "$kit/@KIT_ROOT@" &&
bash development.sh --configure-providers --replace-pilot-launcher
```

「インストール完了」の後、接続先ごとの利用可能状態が表示されます（未ログインの接続先には準備手順が表示されます）。

---

## 3. 起動と基本操作

新しい Bash を開き、作業対象のプロジェクトフォルダで実行します。

```bash
cd ~/workspace/your-existing-project
unreal development
# Ctrl+D で画面を離れた後も、同じコマンドで再接続できます
unreal development
```

※ `unreal` が別のコマンドに隠れる場合は `type -a unreal` で確認し、`"$HOME/.local/bin/unreal"` を使用してください。

### 主なスラッシュコマンド

- `/model` — 接続先・モデル・effort を選択（両方使える初回は Claude、片方のみなら使える側で起動）。
- `/permissions` — `Read Only`（読み取り専用）、`Ask for Approval`（実行前に確認）、`Full Access`（確認なし）を選択。※ `Full Access` は OS の隔離ではありません。
- `/view` — `Split`（初期表示）、`Chat`、`Orchestration` を切り替え。
- `/copy`、`/export` — 会話のコピー、Markdown ファイルへの書き出し。
- `/stop`、`/resume` — 作業の停止対象を選択、停止したセッションを再開。
- `/analyze` — 詳しい診断・統計情報を表示。

---

## 4. 既存ワークスペースでの更新

新しいキットで同じインストール手順を実行します。次にそのワークスペースで `unreal <名前>` を実行すると、更新を自動検出してバックグラウンドの Host を再起動し、**これまでの会話と設定を引き継ぎます**（新しいセッションを作り直す必要はありません）。

### 確認コマンドとインストーラオプション

- `unreal --workspace-info` — 現在のワークスペースの設定・状態保存場所を表示。
- `unreal --readme` — この同梱 README を表示。
- `bash doctor.sh` — 実行ファイルと動作環境（glibc 等）を検査。
- `development.sh` のオプション: `--docker-sandbox-oauth` / `--external-codex` / `--codex-auto`、および `--disable-codex` / `--disable-claude`。

---

## 5. 合成ワークスペースでの受入テスト

`unreal --acceptance-fixture repair` を実行すると、実プロジェクトに触れずに、200行の `sample.txt`（中央行のみ `UNREAL_REPAIR_BEFORE`）と `AGENTS.md` を含む一時 Git リポジトリを作成して起動します。

`/model` と `/permissions` を設定したうえで、以下を送信して動作確認できます。

```text
sample.txtをnative readで200行全体読み、受信した本文とwhole-file revisionだけで
中央のUNREAL_REPAIR_BEFOREをUNREAL_REPAIR_AFTERへnative editしてください。
再readし、200行と他の行が保持されたことを観測してください。
Bashは/bin/sh -cです。POSIX sh構文で以下を検証してください：
set -eu
[ "$(wc -l < sample.txt)" -eq 200 ]
grep -q '^UNREAL_REPAIR_AFTER$' sample.txt
! grep -q '^UNREAL_REPAIR_BEFORE$' sample.txt
git diff -- sample.txt
実際のexit codeとcanonical receiptを観測してからFinalを返してください。
非0、拒否、秘匿stubを成功にせず、成功時だけUNREAL_ACCEPTANCE_OKと記載してください。
commit/push、shellによる編集、provider切替は禁止です。
```
