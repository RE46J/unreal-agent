# Unreal Agent

A coding agent for day-to-day development with Claude and Codex in one terminal.
This repository is a personal-use fork of
[unreallabsai/unreal-agent](https://github.com/unreallabsai/unreal-agent).
It keeps Unreal's Host/Session and Operation model while adding subscription
provider selection, permissions, and delegated GPT work.

Claude runs through the signed-in Claude Code CLI. Codex uses a ChatGPT
subscription login. Choose either provider with `/model`; one available provider
is enough to start. Unreal does not automatically switch providers or fall back
to an API key.

## Start here

1. Prepare a subscription login: run `claude`, then `/login` inside Claude Code,
   or run `codex login` in Bash.
2. Install the Linux amd64 common kit using [Install](docs/install.md).
3. Open Bash in your project and run `unreal development`. Confirm creation if
   this is a new Session name.
4. Use `/model` to choose a provider, model, and effort. Use `/permissions` to
   choose Read Only, Ask for Approval, or Full Access.
5. Type a task and press Enter. Ctrl+D detaches; `unreal development` reconnects.

The terminal starts in Split view on screens at least 120 columns wide. It shows
Chat on narrower screens. Conversation history belongs to the background Host,
so leaving the terminal does not stop the Session.

## Documentation

English is the source documentation. Some current terminal messages are Japanese.

| For users | Contents |
| --- | --- |
| [Install](docs/install.md) | Common kit, login preparation, updates, Sandbox transfer |
| [Usage](docs/usage.md) | Session lookup, terminal input, copy/export, stop/resume, child display |
| [Providers](docs/providers.md) | Claude Code, Codex, model catalogs, Docker OAuth, delegated roles |
| [Permissions](docs/permissions.md) | Modes, approvals, persistence, parent/child decisions |
| [Troubleshooting](docs/troubleshooting.md) | Authentication, startup, reconnecting, logs, analysis |

| For developers | Contents |
| --- | --- |
| [Contributing](CONTRIBUTING.md) | Development environment and local/CI checks |
| [Architecture](docs/architecture.md) | Ownership, execution boundaries, native launcher flags |
| [Context](docs/context.md) | Projection, omission, compaction, Project Instructions |
| [Tools](docs/tools.md) | Native files, AST editing, mutation guarantees |
| [Distribution](docs/distribution.md) | Snapshots, common kit, CI, Release, install verification |
| [Headless runner](cmd/unreal-agent-runner/README.md) | JSON requests and JSONL output for development/evaluation |
| [Harbor adapter](benchmarks/harbor/README.md) | Benchmark setup and execution |

Full Access controls harness approvals. It does not isolate the operating system.
