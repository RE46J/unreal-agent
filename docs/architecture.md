# Architecture

## Host and Session ownership

The [Host](../harness/host/host.go) owns one Coordinator per live Session. A
process-shared writer lock is acquired before creating or resuming a Session.
The canonical Session store owns history; viewer state and provider conversation
state are projections or associated runtime state.

The runtime factory constructs a fresh context builder, provider client, tools,
and Operation Manager for the Session lifetime. Disconnecting an attach client
or canceling a submission does not end that lifetime. Explicit Session stop or
Host shutdown cancels execution and drains handlers before releasing ownership.

Each submission carries the current generation and a stable Inbox ID. A receipt
means its input was committed, not that requested side effects completed.
Identical retries reuse the receipt; conflicting payloads and stale generations
are rejected. Reconnect inspects/resubscribes to rebuild the projection.
Generations and revisions describe live state; canonical sequences identify
persisted history.

Stop completion is recorded separately. Pending stops are recovered before
Operation dispatch. Host restart cancels unfinished children and retains an end
notice for the parent; it does not automatically respawn them.

## Operations, Registry, and execution

The [Registry](../harness/tool/registry.go) owns enabled definitions and their
Translators: Bash, ViewImage, SkillUse, native files, AST tools, and the observed
`claude_code` tool. Protocol and subagent tools are composed by the runtime.
A Translator validates a model call and submits an Operation specification. It
does not perform filesystem or network effects itself.

The Coordinator records model responses, tool-call status, and immutable plans.
The Operation Manager owns execution and cancellation. Executors return typed
outcomes; canonical receipts remain the basis for continuation and recovery.
An observed Claude Code receipt records a tool the external CLI already ran;
it never authorizes Unreal to run that same tool a second time.

Permission is checked at the concrete execution boundary. With fixed policies,
`host.Options.Policy` supplies immutable authority and nil denies all. With human
modes enabled, it supplies the inspection baseline; the Host's controller and
configured maximum determine effective operation permissions. Parent policy
restrictions remain in force. Instruction text and provider output grant no
capabilities. See [Permissions](permissions.md) and [Tools](tools.md).

## Providers and context

The provider registry resolves nonsecret runtime selections. Credentials are
resolved by their external owner for requests, rather than stored in the runtime
identity. `/model` changes are canonical selections; refresh also rediscovers
available backends.

- `openai-codex` uses Unreal's adapter and tool execution pipeline.
- `claude` runs Claude Code, uses its conversation and built-in tools, and
  delegates approval decisions to Unreal. CLI tool outcomes become observed
  Operations. Declined Bash commands use Unreal's Bash executor.

The [runtime composition](../cmd/internal/agentrunner/runtime.go) keeps these
routes behind the same Host. There is no automatic switch on a provider error.
[Context](context.md) describes the different projection/compaction paths.

LSP clients manage language-server protocol work, with file changes routed
through the shared mutation service. DAP clients manage debugger adapters and
debug sessions. Their protocol state and lifetimes remain separate from model
inference, terminal rendering, and authorization.

## TUI and exports

The terminal is a Host gateway client. It projects history and live Operations
into Chat, Orchestration, or Split. It resynchronizes after a gap or ownership
change; transient progress is not a durable completion receipt. Analysis, copy,
and Markdown export read canonical public history rather than only rendered
rows. Project Instruction content remains in canonical storage; public Host
history projects its metadata.

## Native launcher

`make build` creates `bin/unreal-agent`, `bin/unreal`,
`bin/unreal-agent-runner`, and `bin/unreal-agent-auth`. `bin/unreal` is a symlink
to the Host executable; dispatch uses the executable name. These instructions
apply to the **source-built native launcher**, not the common-kit wrapper.

1. Prepare a provider subscription login as in [Providers](providers.md).
2. Build from the repository root:

   ```bash
   make build
   bin/unreal --help
   ```

3. Run `bin/unreal development` to open a named Session. With no name, native
   `bin/unreal` uses `default`. It does not show the common-kit Session list.

On first normal launch, provider discovery creates a runtime configuration if
none exists. Claude is preferred when both providers are available.
`UNREAL_HARNESS_LLM_PROVIDER` and `UNREAL_HARNESS_LLM_MODEL` can explicitly select
the initial provider/model. Existing configuration is not replaced by bootstrap.
The launcher starts a background Host when necessary, waits for readiness, then
creates, attaches, or resumes the Session. Ctrl+D detaches the viewer.

[Native flags and defaults](../cmd/unreal-agent/launcher.go):

| Flag | Default and meaning |
| --- | --- |
| `--config` | With no explicit path, `${XDG_CONFIG_HOME:-$HOME/.config}/unreal-agent/runtime.json`; an explicit file selects operator configuration |
| `--state-directory` | `${XDG_STATE_HOME:-$HOME/.local/state}/unreal-agent` |
| `--session-directory` | `STATE/sessions` |
| `--socket` | Private `$XDG_RUNTIME_DIR/unreal-agent/host.sock` when usable, otherwise `STATE/run/host.sock` |
| `--credential-directory` | Unset; optional managed key store for developer configurations |
| `--codex-auth-file` | External Codex discovery unless a path is specified |
| `--startup-timeout` | `0`: wait until canceled; a positive Go duration bounds readiness waiting |
| `--bootstrap-codex-disabled` | `false`: disable Codex during first-run discovery when set |
| `--bootstrap-claude-disabled` | `false`: disable Claude during first-run discovery when set |
| `--bootstrap-codex-docker-oauth` | `false`: explicitly select Docker Codex OAuth during bootstrap |
| `--bootstrap-codex-auto` | `false`: validate automatic Codex route detection during bootstrap |
| `--bootstrap-claude-binary` | Unset: optional external Claude executable |
| `--help`, `-h` | Show native usage |

XDG base directories must be absolute. Options may appear before or after the
Session name. Host compatibility checks compare configuration, Session store,
credential store, Codex auth paths, and normal-mode binding; they do not compare
an executable digest or protocol revision. The common-kit wrapper separately
detects installed build/profile changes and restarts the same workspace Host.

The operator JSON loader rejects unknown members. The current top-level
configuration contains `Runtime`, `Permissions`, optional `PermissionModes`,
`Providers`, `Launcher`, `Context`, `LanguageServers`, `DebugAdapters`, and
`Subagents`; see [composition](../cmd/unreal-agent/run.go). `Runtime.Claude`
and a Claude backend's `Claude` hold its `Binary`. Discovery's executable override
is `Launcher.ClaudeBinary`. The provider selection retry key is `max_attempts`.
Prefer the generated configuration over copying an unrelated environment's JSON.

For lower-level Host CLI options, run `bin/unreal-agent serve -h` or
`bin/unreal-agent attach -h`. The normal launcher supplies those paths and owns
background startup. The common-kit wrapper additionally owns workspace lookup,
per-workspace config/state, and installed-version handling.
