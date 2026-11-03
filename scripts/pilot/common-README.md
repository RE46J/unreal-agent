# Unreal Agent Common Development Kit (共通開発キット)

**Language:** English | [日本語](../../docs/ja/scripts/pilot/common-README.md)

**You can start as long as either Codex or Claude (どちらか一方) is signed in.** Select your provider, model, and reasoning effort from the unified `/model` menu. No API keys are used.

This common kit works across Linux amd64 environments (including `asp` and `devlore` Sandboxes). It requires glibc @MIN_GLIBC@+, Python 3, Bash, and Git. It packages local snapshot `@REVISION@` without bundling any credentials.

---

## 1. Prepare Provider Login

Sign in to at least one provider in the Linux environment where Unreal runs:

- **Codex:** Install the official Codex CLI and run `codex login` in Bash (or use existing Docker Sandbox OAuth routing).
- **Claude:** Install the official Claude Code CLI, start `claude` in Bash, and run `/login` with your Claude subscription.

Running `/login` inside Unreal displays these instructions. If you sign in to the other provider later, run `/model refresh` (or select that provider in `/model`) to detect it without restarting Unreal.

---

## 2. Installation

### Optional: Transfer from Windows PowerShell to a Docker Sandbox

```powershell
sbx cp .\@ARCHIVE@ asp:/tmp/@ARCHIVE@
sbx cp .\@ARCHIVE@.sha256 asp:/tmp/@ARCHIVE@.sha256
sbx cp .\@ARCHIVE@ devlore:/tmp/@ARCHIVE@
sbx cp .\@ARCHIVE@.sha256 devlore:/tmp/@ARCHIVE@.sha256
```

### Install in Linux / Sandbox Bash

```bash
cd /tmp
archive='@ARCHIVE@'
test -f "$archive" && test -f "$archive.sha256" && sha256sum -c "$archive.sha256" &&
kit=$(mktemp -d "$HOME/unreal-common.XXXXXXXX") &&
tar -xzf "$archive" -C "$kit" &&
cd "$kit/@KIT_ROOT@" &&
bash development.sh --configure-providers --replace-pilot-launcher
```

After installation completes, the script prints the availability status of each provider and shows sign-in instructions for any unavailable provider.

---

## 3. Starting and Using Unreal

Open a new Bash shell in your project directory and run:

```bash
cd ~/workspace/your-existing-project
unreal development
# After detaching with Ctrl+D, reconnect with the same command:
unreal development
```

*(If `unreal` is shadowed by another command, check `type -a unreal` and run `"$HOME/.local/bin/unreal"`.)*

### Essential Slash Commands

- `/model` — Choose provider, model, and effort (defaults to Claude if both are ready, or the available one).
- `/permissions` — Choose `Read Only`, `Ask for Approval`, or `Full Access` (note: `Full Access` is not an OS sandbox).
- `/view` — Switch between `Split` (default), `Chat`, and `Orchestration`.
- `/copy`, `/export` — Copy messages to clipboard or export Markdown files.
- `/stop`, `/resume` — Stop active work or resume a stopped Session.
- `/analyze` — Inspect detailed session usage and context metrics.

---

## 4. Updating an Existing Workspace

Run the same installation steps with a newer kit archive. The next time you run `unreal <NAME>` in that workspace, the launcher detects the update, automatically restarts the background Host (再起動), and carries over your existing conversations and settings (会話と設定を引き継ぎます).

### Useful Diagnostic & Setup Flags

- `unreal --workspace-info` — Show config, state, and log paths for the current workspace.
- `unreal --readme` — Display this bundled README.
- `bash doctor.sh` — Verify packaged binaries and glibc compatibility.
- Optional `development.sh` flags: `--docker-sandbox-oauth`, `--external-codex`, `--codex-auto`, `--disable-codex`, `--disable-claude`.

---

## 5. Acceptance Check in a Synthetic Workspace

Running `unreal --acceptance-fixture repair` creates an isolated temporary Git repository containing a 200-line `sample.txt` (with `UNREAL_REPAIR_BEFORE` on the middle line) and `AGENTS.md`, without touching your real projects.

After checking `/model` and approving permissions in `/permissions`, send:

```text
Read all 200 lines of sample.txt with native read, then use native edit with the observed whole-file revision to change the middle line from UNREAL_REPAIR_BEFORE to UNREAL_REPAIR_AFTER.
Read the file again to confirm all 200 lines are preserved, then verify with POSIX sh via Bash:
set -eu
[ "$(wc -l < sample.txt)" -eq 200 ]
grep -q '^UNREAL_REPAIR_AFTER$' sample.txt
! grep -q '^UNREAL_REPAIR_BEFORE$' sample.txt
git diff -- sample.txt
Report UNREAL_ACCEPTANCE_OK only if all commands succeed with exit code 0 and canonical receipts. Do not commit, push, edit via shell, or switch providers.
```
