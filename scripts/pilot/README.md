# Unreal Agent asp Pilot 配布キット

Linux amd64 / glibc @MIN_GLIBC@ 以上用のローカル snapshot（基準 revision: `@REVISION@`）です。

- **4つのビルド済みバイナリを同梱:** `asp` 内のビルド、Go、CGO コンパイラは不要です（CGO / Tree-sitter を維持しています）。
- **スナップショットの性質:** 元の `SNAPSHOT.json` の dirty-worktree 表示を保持しており、正式 Release ではありません。認証情報、設定、Session 履歴は同梱していません。

---

## 導入と診断の2ステップ

1. **Windows ホストから `asp` へコピー**

   配布ファイルのある Windows フォルダを開き、Windows PowerShell で `sbx cp` を実行してください。

   ```powershell
   sbx cp ./@ARCHIVE@ asp:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 asp:/tmp/@ARCHIVE@.sha256
   ```

   - 参照：[Docker 公式 sbx cp](https://docs.docker.com/reference/cli/sbx/cp/)
   - Sandbox 側のコピー先は絶対パスです。既存ファイルがある場合はコピー前に別名の専用ディレクトリを指定し、手順2の `/tmp` もその場所に読み替えてください。

2. **`asp` 内でインストール・診断を一括実行**

   ```sh
   (set -eu; cd /tmp; sha256sum -c @ARCHIVE@.sha256; work=$(mktemp -d /tmp/unreal-pilot.XXXXXXXX); tar -xzf @ARCHIVE@ -C "$work"; bash "$work/@KIT_ROOT@/install.sh")
   ```

   - **配置先:** checksum、OS/CPU、glibc、依存ライブラリを検査してから、`$HOME/.local/opt/unreal-agent-pilot/asp/@REVISION_SHORT@-<content-id>/` へ配置します。
   - **安全な再実行と停止条件:** 同じキットの再実行は内容検証後に既存配置を再利用します。変更された配置や symlink は上書きせず停止し、中断した installer の lock が残る場合も停止します。
   - **同梱 `doctor` の動作範囲:** 4つの binary の `help` / `methods` のみを空の一時 `HOME` で実行します（実推論、プロジェクト操作、Host 起動、OS パッケージ変更は行いません）。

---

## 動作要件と保証範囲

- **必要な既存コマンド:** `bash` と一般的な Linux / coreutils（`tar`, `sha256sum`, `uname`, `getconf`, `ldd`, `stat`, `od`, `tr`, `env`, `timeout`, `mktemp`, `mkdir`, `cp`, `chmod`, `mv`, `cmp`, `grep`, `rm`, `rmdir`, `cat`）。不足や非互換は診断エラーとして表示し、OS パッケージの自動導入は行いません。
- **External Codex 認証の確認範囲:** 既存パスの存在・読み取り可否だけを確認します。内容、有効期限、アカウント、API 接続、ネットワーク疎通は確認せず、追加設定も行いません（診断の `WARN` は、認証や接続の実動作が保証されたという意味ではありません）。
- **既存環境への非干渉:** 既存の Codex CLI、`PATH`、認証ファイル、runtime 設定、Session Store を変更しないため、復帰時はこれまでの Codex 起動操作をそのまま利用できます。Pilot Host は起動しておらず、Pilot を使わない場合も専用ディレクトリを保持して構いません。通常の `unreal` 起動や実推論・編集・Permission 付与は今回の2操作には含めません。
