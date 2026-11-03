# Context and Project Instructions

**Language:** English | [日本語](ja/context.md)

Unreal Agent never deletes or mutates your canonical Session history. Instead, it builds a **derived context view** for each model request. Tool-result pruning and conversation compaction only reduce what is sent to the model; original messages, Operations, and receipts remain intact in the Session store.

---

## 1. Codex Context Management

For Codex (`openai-codex`), the [context builder](../harness/contextbuilder/compaction.go) constructs model input deterministically from canonical history:

- **Append-only prefix stability:** Between prune/compaction boundaries and model changes, new items are appended in recorded order without reordering or reconstructing earlier items, maximizing prompt cache hits.
- **Provider/model boundaries:** While staying on the same provider and model, encrypted reasoning tokens and item IDs are reused. Switching models records a boundary and converts earlier history into a portable representation.

### Tool-Result Omission (Pruning)

Unreal reduces large or outdated tool outputs in two stages:

1. **Recording-time head/tail truncation:** Very large tool outputs are capped at `max_output_length` (default: 40,000 characters) when recorded into the transcript, while the full output file and Operation receipt remain stored.
2. **Request-time pruning ([`tool_pruning.go`](../harness/contextbuilder/tool_pruning.go)):** Before sending a request, the builder keeps recent tool outputs within a token budget and replaces older tool result bodies with compact reference markers (pointing to the canonical Operation ID and shell output file).
   - **Obsolete file reads:** If a file is read or edited again later via native file tools, Applied AST edits, or simple recognized Bash reads, earlier reads of that same file become eligible for pruning immediately.
   - **Protected items:** Active Operations, unacknowledged continuations, user/parent messages, assistant text, and reasoning are never pruned.

### Automatic Summarization (Compaction)

When pruning old tool results is not enough to fit within the model's context budget:

1. **Compaction threshold:** Triggered at the minimum of:
   - `80%` of the model's context window
   - `context_window - max_output - 64,000`
   - Configured `InputBudget`
2. **How it works:** Unreal first prunes eligible old tool results, then requests a summary using the same provider, model, and Session cache identity. The summary and replaced range are recorded as a compaction boundary while keeping recent turns intact.
3. **Overflow recovery:** If a provider returns a context-overflow error, Unreal forces compaction and retries the request once. If the required recent context still exceeds the budget, it fails with `required_context_exceeds_budget`.

### Context Configuration (`runtime.json`)

Configured under the top-level `Context` key ([`harness/contextengine/types.go`](../harness/contextengine/types.go)):

| Key | Default | Description |
| --- | --- | --- |
| `ToolResultKeepTokens` | `15000` | Token budget reserved for recent tool outputs before older outputs become prune candidates |
| `ToolResultMinPruneTokens` | `15000` | Minimum estimated token savings required before writing a new `prune` boundary |
| `ClaudeCompactThresholdTokens` | `200000` | Input token threshold that triggers background idle compaction for Claude |

---

## 2. Claude Conversation Compaction

Because Claude Code manages its own conversation state, Unreal coordinates compaction through the `claude` CLI:

- **Background idle compaction (default: >200,000 tokens):** When a turn and all of its tool calls finish, Unreal checks the latest context size. If it exceeds `Context.ClaudeCompactThresholdTokens`, Unreal starts a background `/compact` on the same Claude Code conversation with preservation guidance.
  - If you send a new message or a child agent reports back while idle compaction is running, Unreal immediately interrupts the background compaction and runs your turn without delay.
- **Blocking compaction (>2× threshold, i.e., >400,000 tokens):** Only when context exceeds twice the threshold does Unreal run compaction before starting the next turn.
- **Context measurement:** The TUI footer displays the latest `message_start` input tokens (`input + cache_read + cache_creation`), rather than summing every step of an internal tool loop.

---

## 3. Project Instructions (`AGENTS.md` and `CLAUDE.md`)

Project instructions guide coding style and workflow. They never grant tool permissions, credentials, or Session authority.

### How Codex Reads Instructions

- **Snapshot at Session creation:** When creating a new Session, Unreal reads `<workspace>/AGENTS.md` ([`discover.go`](../harness/projectinstructions/discover.go)) and saves its exact text as an immutable snapshot in the Session store.
- **Consistent across resume & children:** Resuming, restarting, forking, or spawning child agents from that Session reuses the saved snapshot. Editing `AGENTS.md` on disk takes effect when you create a new Session.
- **Format rules:** Must be valid UTF-8 with no NUL bytes. Nested `AGENTS.md` files in subdirectories and `@import` directives are not expanded by the snapshot reader.

### How Claude Reads Instructions

- Claude Code reads `CLAUDE.md` natively.
- On each `claude` CLI invocation, Unreal also reads the workspace's current `AGENTS.md` and appends it to the prompt—**unless** `CLAUDE.md` already contains the substring `AGENTS.md` (such as `@AGENTS.md`). See [`prompt.go`](../harness/llm/clients/claudeagent/prompt.go).
- Because Claude reads the file on each invocation, edits to `AGENTS.md` or `CLAUDE.md` take effect on subsequent turns even within an existing Session.

---

## 4. Inspecting Context & Usage Metrics

Run `/analyze` in the TUI and select **Context** or **Usage** (or export the report) to inspect:
- Average input tokens across requests
- Number of omitted tool results, prune boundaries, and estimated saved tokens
- Prompt cache hit ratios
- Claude compaction history (timing, pre-compaction input size, and completed/interrupted/failed status)

See [Troubleshooting — Read analysis and measurements](troubleshooting.md#read-analysis-and-measurements) for details on interpreting these numbers.
