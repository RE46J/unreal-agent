# unreal-agent-runner

Run one prompt or JSON request headlessly and write canonical Session items to
stdout as JSONL. The process exits when the task finishes. This is a developer
and evaluation tool; daily subscription use starts with
[the common-kit terminal](../../docs/install.md).

## Build and run

Use Go from [go.mod](../../go.mod) and a C compiler for AST support. From the
repository root:

```bash
make build
bin/unreal-agent-runner -h
```

OpenAI is the default provider. API-key routes are for development/evaluation and
can incur API charges; they are separate from the product's subscription setup.
To run an OpenAI evaluation, enter its key without putting the value in shell
history:

```bash
read -r -s -p 'Evaluation OpenAI API key: ' OPENAI_API_KEY
printf '\n'
export OPENAI_API_KEY
bin/unreal-agent-runner -p 'Inspect this project and explain how to run its tests.'
```

Choose a workspace and save JSONL output (the current directory is a valid
workspace; replace `.` with an existing project path if needed):

```bash
bin/unreal-agent-runner -workspace . -p 'Summarize this project.' > run.jsonl
```

A runner is not an OS sandbox. Use an appropriately isolated development or
benchmark environment for execution. Its default prompt's sandbox wording does
not create isolation.

## JSON requests and output

Pass a JSON request as an argument, or through stdin:

```bash
bin/unreal-agent-runner '{"prompt":"Summarize this project."}'
printf '%s\n' '{"prompt":"Summarize this project."}' | bin/unreal-agent-runner
```

Options go before a positional JSON request. `-p` and positional JSON are mutually
exclusive. Unknown request fields are rejected. Common fields are `prompt`,
`messages`, `model`, `thinking_level`, `session_id`, `system_prompt`,
`max_attempts`, and `disallowed_tools`; `-h` shows the full schema.

Session files default to
`${XDG_STATE_HOME:-$HOME/.local/state}/unreal-agent/sessions`; override with
`-session-directory`. `-log-directory` adds an optional JSONL log directory.
A `session_id` can create or resume a persisted Session. Output failure can leave
only a prefix of JSONL on stdout; the Session store remains authoritative.

Set `UNREAL_HARNESS_LLM_PROVIDER` to `openai`, `openai-codex`, `openrouter`,
`fireworks`, or `ollama`; `UNREAL_HARNESS_LLM_MODEL` chooses a model.
`openai-codex` uses external Codex authentication. This runner's provider list is
separate from the interactive Claude/Codex menu. See
[provider definitions](../../cmd/internal/agentrunner/providers.go) and the
[Harbor adapter](../../benchmarks/harbor/README.md) for evaluation setup.
