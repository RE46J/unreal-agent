# Unreal Agent asp パイロット配布キット

**言語:** [English](../../../../scripts/pilot/README.md) | 日本語

Linux amd64 / glibc @MIN_GLIBC@ 以上向けのローカルスナップショット（基準リビジョン: `@REVISION@`）です。
CGO / Tree-sitter 有効の4つのバイナリを同梱しており、`asp` 内に Go や C コンパイラは不要です。認証情報・設定・セッション履歴は同梱していません。

1. **Windows ホストから `asp` へファイルをコピーする**

   配布ファイルがある Windows フォルダを開き、以下を実行します。

   ```text
   sbx cp ./@ARCHIVE@ asp:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 asp:/tmp/@ARCHIVE@.sha256
   ```

   参照: [Docker 公式 sbx cp ドキュメント](https://docs.docker.com/reference/cli/sbx/cp/)
   コピー先に同名ファイルがすでにある場合は、別の専用ディレクトリを指定し、手順2の `/tmp` もそのパスに読み替えてください。

2. **`asp` 内でインストールと診断を一括実行する**

   ```sh
   (set -eu; cd /tmp; sha256sum -c @ARCHIVE@.sha256; work=$(mktemp -d /tmp/unreal-pilot.XXXXXXXX); tar -xzf @ARCHIVE@ -C "$work"; bash "$work/@KIT_ROOT@/install.sh")
   ```

   チェックサム、OS / CPU、glibc バージョン、共有ライブラリを検査したうえで、`$HOME/.local/opt/unreal-agent-pilot/asp/@REVISION_SHORT@-<content-id>/` に配置します。
   - 同じキットを再実行した場合は、内容検証後に既存の配置を再利用します（改変された配置やシンボリックリンク、ロック残存時は上書きせず停止します）。
   - 同梱の `doctor.sh` は空の一時 `HOME` で4つのバイナリの `--help` / `methods` のみを検査します（実推論、Host 起動、OS パッケージ変更は行いません）。

### 補足と安全性

- **必要な既存コマンド:** `bash` および一般的な Linux coreutils（`tar`, `sha256sum`, `uname`, `getconf`, `ldd`, `stat`, `od`, `tr`, `env`, `timeout`, `mktemp`, `mkdir`, `cp`, `chmod`, `mv`, `cmp`, `grep`, `rm`, `rmdir`, `cat`）。
- **既存環境への非干渉:** 既存の Codex CLI、`PATH`、認証ファイル、ランタイム設定、Session ストアは一切変更しません。
