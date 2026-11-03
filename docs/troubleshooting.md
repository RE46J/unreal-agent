# Troubleshooting

## Login is required or expired

An `auth required` status means the selected provider is not ready. Unreal does
not switch providers or use an API key automatically.

1. For Claude, open another Bash shell, run `claude`, and enter `/login` inside
   Claude Code. In a Docker Sandbox, run `sbx exec -it <sandbox> claude` from a
   Windows terminal instead. OAuth expiry can also appear as
   `external_reauth_required`, `OAuth session expired`, or `OAuth token has expired`.
2. For Codex, run `codex login` in Bash. For Docker host-managed OAuth, repair
   the host login and Sandbox route instead; see [Providers](providers.md).
3. Return to Unreal and run `/model refresh`, or select the unavailable provider
   in `/model` to retry discovery.
4. If the Session stopped on the error, run `/resume`.

A working login in another OS environment does not establish that the CLI used
by this Linux Host can use it. Unreal's `/login` displays instructions; it does
not run either login flow.

## Startup or command lookup fails

1. In Bash, check command resolution:

   ```bash
   type -a unreal claude codex
   "$HOME/.local/bin/unreal" --workspace-info
   ```

2. If another alias, function, or executable masks `unreal`, use the absolute
   command above. Load `. "$HOME/.config/unreal-agent-common/path.sh"` for the
   installed PATH and `ucd` function, or open a new Bash shell.
3. If there is no Session yet, run `unreal development` from the intended
   project and confirm creation. Bare `unreal` only lists existing Sessions.
4. From the extracted kit, run `bash doctor.sh` to check packaged binaries and
   environment compatibility. It does not prove that login or inference works.

`unreal --workspace-info` reports the current workspace. For a Session elsewhere,
first run `ucd development` with its actual name, then request workspace info.
A config/workspace mismatch should be investigated at the printed paths; avoid
replacing configuration or Session files to clear the error.

## Models are missing or stale

1. Check that the intended external CLI is installed and signed in.
2. For ordinary Codex OAuth, open the Codex CLI once to populate
   `models_cache.json` beside its selected auth file.
3. Run `/model refresh` in Unreal. It rechecks provider availability as well as
   catalog metadata, including providers not registered at startup.
4. If Docker Codex CLI enumeration fails, the last successful saved catalog may
   remain visible. A list alone does not prove account access to a model.

For Docker OAuth, verify the kit route and forward proxy described in
[Providers](providers.md#docker-sandboxes-codex-oauth). A proxy 401/403 does not
justify copying a token into Unreal or choosing an API-key route.

## Reconnect or resume

1. Use the same `unreal development` command to reattach after Ctrl+D or a lost
   terminal connection. Existing stopped Sessions resume through the launcher.
2. Inside the TUI, use `/resume` for a stopped Session. If the old provider is
   unavailable, select another in `/model` first.
3. If a message submission has an uncertain result, use `/retry`. It preserves
   the submission identity. Check the resulting conversation before sending a
   new copy of the same task.
4. If another process owns the Session, close that owning screen/process as
   appropriate and reconnect. Do not delete lock files or live Session data.

Reconnection rebuilds the display from canonical history. A Host restart does
not automatically respawn interrupted children; their cancellation is reported
to the parent, which can decide the next action.

## Find logs

Run `unreal --workspace-info` in the workspace. The **State** path contains
`host.log` and the `sessions/` directory. Common-kit defaults live beneath:

- Config: `~/.config/unreal-agent-common/repos/`
- State: `~/.local/state/unreal-agent-common/`
- Launcher settings: `~/.config/unreal-agent-common/launcher.json`

Read the relevant `host.log` around the failure. Session JSONL files are
canonical conversation records, not disposable debug logs. Avoid publishing
credentials or private conversation content with a bug report.

## Read analysis and measurements

1. Enter `/analyze` to open the analysis picker.
2. Choose a view below. Enter or Esc returns from a view to the picker.
3. Choose Export to save metadata/statistics as JSON or Markdown. Use
   `/export all` for a public conversation Markdown export with measurements.

| View | What to look for |
| --- | --- |
| Overview | Provider/model, response/tool counts, elapsed time, Host source revision and modified-build marker |
| Usage | Input, cached input, reasoning/output totals, and Claude's turn-total usage observation |
| Turns | Per-turn provider/model/effort and usage |
| Tools | Operation outcomes and Claude Bash hook intervals (`native_bash`) |
| Agents | Discovered children, outcomes, and `child_start` template/model/effort records |
| Context | Context window, average input, omitted result counts, estimated savings, prune boundaries, Claude compaction |
| Timeline | Recorded event order and available activity measurements |
| Errors | Recorded failures and available safe provider diagnostics |

Cache percentage is cached input divided by input for reported usage. The
footer's context number is the latest request's context, not the sum of inputs
across a tool loop. Average input can include summarization calls. Pruning
savings are estimates; they do not promise a particular cache percentage.

Generation speed uses reported output tokens and generation timing. `~` marks
client-clock timing. Claude Bash elapsed time runs from its pre-tool hook to the
reported result or termination; it is not the OS process's own CPU/runtime clock.
Claude turn-total usage is already included in response totals: do not add it
again.

Claude compact records show start/end, elapsed time, input before compaction,
usage when reported, and completed/interrupted/failed status with a reason.
Interruption by a new input can be normal. Missing usage is **unknown**, not zero.
A `partial/incomplete history` or `resync` label limits the report to observed
canonical data. Analyze a subsequent real run when the needed measurement is
absent; a local fixture is not evidence of that provider incident.
