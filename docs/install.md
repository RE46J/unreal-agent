# Install

The common kit supports Linux amd64, including WSL and Linux Sandboxes. Both
providers use the same kit. You need Bash, Python 3, Git, `tar`, and `sha256sum`.
The kit's `KIT.json` records its minimum glibc version; `doctor.sh` checks the
packaged executables against your environment.

## Prepare a login

Prepare at least one provider in the Linux environment where Unreal will run.

1. For Claude, install the official Claude Code CLI, run `claude` in Bash, and
   enter `/login` inside Claude Code. Use your Claude subscription.
2. For Codex, install the official Codex CLI and run `codex login` in Bash. Use
   your ChatGPT subscription. Let the CLI manage its login files.
3. For Docker Sandboxes with host-managed Codex OAuth, prepare that host login
   instead. See [Providers](providers.md#docker-sandboxes-codex-oauth).

Unreal's `/login` shows these instructions. It does not accept an API key.
You can add the other provider later with `/model refresh`.

## Get and install the kit

1. Download the common `unreal-agent-linux-amd64-REV.tar.gz` and matching
   `.tar.gz.sha256` file from
   [Releases](https://github.com/TomyFovos/unreal-agent/releases).
   `REV` is the kit's short source revision. Keep both files together.
2. In Bash, use the block below. Enter the full path to the downloaded archive
   when prompted. It verifies the checksum before extracting or installing.

   ```bash
   set -e
   read -r -e -p 'Full path to the kit .tar.gz: ' unreal_archive
   unreal_archive=$(realpath -- "$unreal_archive")
   cd -- "$(dirname -- "$unreal_archive")"
   sha256sum -c "$(basename -- "$unreal_archive").sha256"
   unreal_unpack=$(mktemp -d /tmp/unreal-common.XXXXXXXX)
   tar -xzf "$unreal_archive" -C "$unreal_unpack"
   cd -- "$unreal_unpack/$(basename -- "$unreal_archive" .tar.gz)"
   bash doctor.sh
   bash development.sh --configure-providers --replace-pilot-launcher
   . "$HOME/.config/unreal-agent-common/path.sh"
   ```

3. Read each provider's availability message. If neither is available, finish
   one login before starting Unreal.
4. Open Bash in an existing project and start a named Session:

   ```bash
   unreal development
   ```

   Enter `y` if asked to create it. The workspace is the Git worktree root, or
   the current directory outside Git. Choose permissions before requesting edits.

The installed commands are `~/.local/bin/unreal` and
`~/.local/bin/unreal-agent`. New Bash shells load
`~/.config/unreal-agent-common/path.sh`, which adds the command directory and
provides `ucd`. See [Usage](usage.md).

## Transfer to a Docker Sandbox

1. On Windows, keep the downloaded archive and checksum together. Run this whole
   block in **Windows PowerShell**. Enter the archive path and the name of your
   existing Sandbox. Both transfers and the shell connection use that same name.

   ```powershell
   $unrealArchive = Get-Item -LiteralPath (Read-Host 'Full path to the kit .tar.gz')
   $unrealSandbox = Read-Host 'Existing Sandbox name'
   sbx cp $unrealArchive.FullName "${unrealSandbox}:/tmp/$($unrealArchive.Name)"
   if ($LASTEXITCODE -ne 0) { throw 'Archive transfer failed' }
   sbx cp "$($unrealArchive.FullName).sha256" "${unrealSandbox}:/tmp/$($unrealArchive.Name).sha256"
   if ($LASTEXITCODE -ne 0) { throw 'Checksum transfer failed' }
   sbx exec -it $unrealSandbox bash
   ```

2. In **Sandbox Bash**, run this whole block. Enter the transferred archive's
   filename when prompted. Automatic routing validates a Docker OAuth kit when
   present; it can also use an ordinary external Codex login.

   ```bash
   set -e
   cd /tmp
   read -r -p 'Transferred kit filename (.tar.gz): ' unreal_archive
   sha256sum -c "$unreal_archive.sha256"
   unreal_unpack=$(mktemp -d /tmp/unreal-common.XXXXXXXX)
   tar -xzf "$unreal_archive" -C "$unreal_unpack"
   cd -- "$unreal_unpack/${unreal_archive%.tar.gz}"
   bash doctor.sh
   bash development.sh --configure-providers --replace-pilot-launcher
   . "$HOME/.config/unreal-agent-common/path.sh"
   ```

3. Start `unreal development` from your project in that Sandbox.

The transfer syntax follows Docker's
[`sbx cp`](https://docs.docker.com/reference/cli/sbx/cp/) and
[`sbx exec`](https://docs.docker.com/reference/cli/sbx/exec/) commands.

These steps use an existing Sandbox; Unreal's installer does not create one.

## Update

1. Download the new archive and its checksum.
2. Repeat the installation block with the new archive path.
3. Reopen your existing Session with `unreal NAME`, using its actual name.

The next launch detects a changed build or provider profile and restarts that
workspace's Host in the same instance. Sessions and settings carry over. An
identical reinstall reuses the existing instance. The installer keeps the current
version, the previous version, and versions used by live processes; it removes
other installed versions.

Installer options can select `--codex-auto`, `--external-codex`, or
`--docker-sandbox-oauth`. A fresh install defaults to automatic Codex route
detection and Claude enabled; updates keep prior choices. `--disable-codex` and
`--disable-claude` disable a provider. `--claude-binary` selects a Claude executable.
Specify these on `bash development.sh`, not on the installed `unreal` command.
