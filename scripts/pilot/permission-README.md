# @SANDBOX@ — Permission Mode Acceptance Kit (Gate D / E)

**Language:** English | [日本語](../../docs/ja/scripts/pilot/permission-README.md)

Linux amd64 (`GOAMD64=v1`, CGO / Tree-sitter enabled) acceptance snapshot for verifying Permission Modes in `@SANDBOX@`.

- **Isolated private installation:** Does not modify your existing Codex/Claude CLI, `PATH`, auth files, default config/Host, or existing Sessions.
- **Not an OS sandbox:** Permission modes enforce harness authorization; they do not add an OS sandbox.

---

## 1. Transfer and Setup

1. **From Windows (in the directory containing the archive and `.sha256`):**

   ```text
   sbx cp ./@ARCHIVE@ @SANDBOX@:/tmp/@ARCHIVE@
   sbx cp ./@ARCHIVE@.sha256 @SANDBOX@:/tmp/@ARCHIVE@.sha256
   ```

2. **Inside `@SANDBOX@` Bash:**

   ```sh
   set -eu
   cd /tmp
   sha256sum -c @ARCHIVE@.sha256
   permission_unpack=$(mktemp -d /tmp/unreal-permission.XXXXXXXX)
   tar -xzf @ARCHIVE@ -C "$permission_unpack"
   bash "$permission_unpack/@KIT_ROOT@/gate.sh" setup --model @MODEL@ --effort high@OPT_IN@
   ```

   Requires Python 3, Bash, Git, standard coreutils/`ldd`/`tar`, and glibc @MIN_GLIBC@+. `setup` verifies the environment and creates an isolated test Git workspace without sending live model requests.

3. **Shell helper variables for the gate checks below:**

   ```sh
   permission_gate="$HOME/.local/state/unreal-agent-pilot/@SANDBOX@/permissions-@REVISION_SHORT@"
   permission_package="$HOME/.local/opt/unreal-agent-pilot/@SANDBOX@/@REVISION_SHORT@-$(sha256sum "$permission_unpack/@KIT_ROOT@/SHA256SUMS" | cut -c1-16)"
   # Replay this README at any time without starting a Host:
   bash "$permission_package/gate.sh" readme
   ```

---

## 2. Gate D — History Persistence and Host Lifetime

1. Open a new Session with `"$permission_gate/unreal" gate-d` (confirm `Read Only` in the header, `/model`, and `/view orchestration`).
2. In another Bash shell, print the Gate D prompt with `bash "$permission_package/gate.sh" prompt d`, send it once in the TUI, wait for the final response, and detach with `Ctrl+D`.
3. Reattach with `"$permission_gate/unreal" gate-d` to confirm the history is intact, then detach (`Ctrl+D`).
4. Run the verification, stop, and resume check sequence:

   ```sh
   bash "$permission_package/gate.sh" verify d
   bash "$permission_package/gate.sh" checkpoint gate-d
   bash "$permission_package/gate.sh" stop
   "$permission_gate/unreal" gate-d
   # Confirm history in TUI, then press Ctrl+D
   bash "$permission_package/gate.sh" resume-check gate-d
   ```

---

## 3. Gate E — Real Operations and Human Approvals

Open `"$permission_gate/unreal" gate-e` (starts in `Read Only`). For each stage below, display the prompt with `bash "$permission_package/gate.sh" prompt STAGE`, send it once in the TUI, wait for completion, detach (`Ctrl+D`), and run `verify STAGE`:

| Stage | Action & Pass Criteria |
| --- | --- |
| `read` | Executes `read`, `grep`, and `glob` with canonical receipts. |
| `deny` | Requests `write` and `Bash` while in `Read Only`. Must produce typed Unreal denials and create zero files. |
| `ask` | Switch `/permissions` to `Ask for Approval`. Allow the first write (`once.txt`) and Deny the second write. |
| `work` | Switch `/permissions` to `Full Access` (confirm warning). Performs native `write` -> `read` -> `edit` -> `read` -> `Bash` verification on `sample.txt`. |
| `full` | In `Full Access`, runs failing `python3 -m unittest -v` via Bash, edits code + writes `work.txt`, and re-runs tests to pass. |

### Verification Commands

```sh
bash "$permission_package/gate.sh" verify read
bash "$permission_package/gate.sh" verify deny
bash "$permission_package/gate.sh" verify ask
bash "$permission_package/gate.sh" verify work
bash "$permission_package/gate.sh" verify full

# Verify permission mode retention across Host restart:
bash "$permission_package/gate.sh" checkpoint gate-e
bash "$permission_package/gate.sh" stop
"$permission_gate/unreal" gate-e
# Confirm mode & history, then detach (Ctrl+D)
bash "$permission_package/gate.sh" resume-check gate-e
"$permission_gate/unreal" gate-e
# Send `prompt restart` once, wait for completion, then detach
bash "$permission_package/gate.sh" verify restart
```

---

## 4. Diagnostics and Cleanup

If a provider request fails during testing, detach and run the safe read-only diagnostic (prints error classification and receipt counts without exposing prompts or credentials):

```sh
bash "$permission_package/gate.sh" diagnose gate-e
```

To finish, detach with `Ctrl+D` and optionally stop the dedicated gate Host with `bash "$permission_package/gate.sh" stop`. Your normal `codex`, `claude`, and `unreal` installations remain untouched.
