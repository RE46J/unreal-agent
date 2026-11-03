# unreal-agent-runner

**Language:** English | [日本語](../../docs/ja/cmd/unreal-agent-runner/README.md)

`unreal-agent-runner` executes a single prompt or JSON request headlessly, streams canonical Session items to `stdout` as JSONL, and exits when the task finishes.

> **Intended Use:** This is a developer and benchmark evaluation tool. For interactive daily development with subscription logins, use the [common-kit terminal (`unreal`)](../../docs/install.md).

---

## 1. Build and Run

Requires the Go version in [`go.mod`](../../go.mod) and a C compiler (`CGO_ENABLED=1` for Tree-sitter AST tools).

From the repository root:

```bash
make build
bin/unreal-agent-runner -h
```

### Running a Prompt with an OpenAI API Key

By default, the runner uses the `openai` provider. (API-key routes are for evaluation and incur API charges; they are separate from the interactive TUI's subscription setup.)

To enter an evaluation API key without recording it in your shell history:

```bash
read -r -s -p 'Evaluation OpenAI API key: ' OPENAI_API_KEY
printf '\n'
export OPENAI_API_KEY
bin/unreal-agent-runner -p 'Inspect this project and explain how to run its tests.'
```

### Specifying a Workspace and Saving JSONL Output

```bash
bin/unreal-agent-runner -workspace . -p 'Summarize this project.' > run.jsonl
```

> **Security Note:** `unreal-agent-runner` is not an OS sandbox. Use an appropriately isolated development/evaluation environment; a container or VM is recommended when OS isolation is needed.

---

## 2. JSON Requests and Configuration

Instead of `-p`, you can pass a structured JSON request as a positional argument or via `stdin`:

```bash
bin/unreal-agent-runner '{"prompt":"Summarize this project."}'
printf '%s\n' '{"prompt":"Summarize this project."}' | bin/unreal-agent-runner
```

### Request Schema & Flags

- Flags must appear before a positional JSON argument (`-p` and positional JSON cannot be combined).
- Unknown JSON fields are rejected.
- Common JSON fields: `prompt`, `messages`, `model`, `thinking_level`, `session_id`, `system_prompt`, `max_attempts`, `disallowed_tools` (run `bin/unreal-agent-runner -h` for the complete schema).

JSONL on `stdout` may be incomplete if writing fails. The persisted Session store is the canonical record; the output stream is a client view.

### Storage and Provider Environment Variables

- **Session store:** Defaults to `${XDG_STATE_HOME:-$HOME/.local/state}/unreal-agent/sessions` (override with `-session-directory`). Passing a `session_id` creates or resumes a persisted Session.
- **Optional log directory:** Use `-log-directory <dir>` to write per-run JSONL logs.
- **Provider & model selection:**
  - `UNREAL_HARNESS_LLM_PROVIDER`: `openai` (default), `openai-codex` (uses external Codex OAuth), `openrouter`, `fireworks`, or `ollama`
  - `UNREAL_HARNESS_LLM_MODEL`: Model identifier for the selected provider

See [provider definitions](../../cmd/internal/agentrunner/providers.go) and the [Harbor Benchmark Adapter](../../benchmarks/harbor/README.md) for full benchmark setup.
