# Unreal Agent asp Pilot Distribution Kit

**Language:** English | [日本語](../../docs/ja/scripts/pilot/README.md)

Local Linux amd64 snapshot (requires glibc @MIN_GLIBC@+, baseline revision `@REVISION@`).
Includes all 4 prebuilt binaries with CGO/Tree-sitter enabled—no Go or C compiler is needed inside `asp`. Credentials, configuration, and Session history are not bundled.

1. **Copy the archive from the Windows host into `asp`**

   Open the Windows folder containing the distribution files and run:

   ```text
   sbx cp ./@ARCHIVE@ asp:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 asp:/tmp/@ARCHIVE@.sha256
   ```

   Reference: [Docker sbx cp documentation](https://docs.docker.com/reference/cli/sbx/cp/).
   If `/tmp` already contains files with the same name, copy into a dedicated directory instead and adjust `/tmp` in step 2 accordingly.

2. **Install and run diagnostics inside `asp`**

   ```sh
   (set -eu; cd /tmp; sha256sum -c @ARCHIVE@.sha256; work=$(mktemp -d /tmp/unreal-pilot.XXXXXXXX); tar -xzf @ARCHIVE@ -C "$work"; bash "$work/@KIT_ROOT@/install.sh")
   ```

   This verifies the SHA-256 checksum, OS/CPU architecture, glibc version, and shared libraries before installing into `$HOME/.local/opt/unreal-agent-pilot/asp/@REVISION_SHORT@-<content-id>/`.
   - Re-running the same kit reuses the verified installation; modified directories, symlinks, or stale installer locks stop safely without overwriting.
   - `doctor.sh` runs only `--help` / `methods` checks for the 4 binaries in an empty temporary `HOME` (no live model inference, Host startup, or OS package changes).

### Notes & Guarantees

- **Required standard tools:** `bash` and standard Linux coreutils (`tar`, `sha256sum`, `uname`, `getconf`, `ldd`, `stat`, `od`, `tr`, `env`, `timeout`, `mktemp`, `mkdir`, `cp`, `chmod`, `mv`, `cmp`, `grep`, `rm`, `rmdir`, `cat`).
- **Non-destructive:** Does not modify your existing Codex CLI, `PATH`, auth files, runtime config, or Session store.
