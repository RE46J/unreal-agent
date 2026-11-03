# Context and Project Instructions

Canonical Session history is the original record. Model input is derived from
it. Omission and summarization change the derived view; they do not delete the
original conversation, Operations, or receipts.

## Codex input

The [context builder](../harness/contextbuilder/compaction.go) derives items in
recorded order. Between prune/compact and provider/model boundaries it appends
to the previous input. Instructions and tool schemas are stable; moving operation
state is appended only when it changes. Model and Responses representations are
memoized within the view/model boundary.

The same provider/model can receive its encrypted reasoning and item IDs again.
A provider/model change records a boundary; earlier items use a portable
representation. Tools are ordered deterministically. Request-time retrieval,
reordering, and repeated reconstruction of a changing prompt prefix are not
part of this path.

### Tool-result omission

Large tool output can be reduced to head/tail text when recorded, using the
existing `max_output_length` option (default 40,000 characters). Receipts,
Operations, and any recorded output files remain available. This recording-time
reduction is separate from context pruning.

Before a request, the builder keeps recent tool results and identifies older
results that can be replaced by short references. A later successful read or
edit of the same file can also make an earlier read obsolete. Recognition is
conservative: native read/write/edit and applied AST changes use known paths;
only simple supported Bash reads are interpreted. Failed reads, preview-only
edits, and complex shell expressions do not establish a newer file snapshot.

[Pruning](../harness/contextbuilder/tool_pruning.go) preserves call/result pairs
and IDs. A single recorded `prune` boundary can replace several result bodies.
References name the canonical unit and, when present, its shell output file.
Active Operations, required unacknowledged continuations, user/parent inputs,
model text, and reasoning are protected. Restart replays the same records.

The developer configuration is under top-level `Context`:

| Key | Default | Meaning |
| --- | --- | --- |
| `ToolResultKeepTokens` | `15000` | Recent-result budget before older results become candidates |
| `ToolResultMinPruneTokens` | `15000` | Minimum estimated net savings for a prune boundary |
| `ClaudeCompactThresholdTokens` | `200000` | Claude idle-compaction threshold; separate route below |

These values are resolved in [contextengine](../harness/contextengine/types.go).
Token estimates start from conservative JSON byte sizes and can be calibrated
against measured request input. They are not exact tokenizer measurements.

### Summarization

The compaction threshold is the minimum of 80% of the context window, the window
minus maximum output minus 64,000, and configured `InputBudget`. An unknown
window uses the configured fallback; an unknown maximum output uses
`ResponseReserve`.

If normal omission cannot fit input, the builder first records references for
eligible old results. If more reduction is needed, it requests a summary with
the same provider/model and Session cache identity, appending a summarization
instruction to the derived input. It records the summary and replaced range,
keeping recent history and required continuations. A context-overflow error
forces compaction and one ordinary-request retry. If required input still
cannot fit, it returns `required_context_exceeds_budget`.

This applies to Codex main and child Sessions. The Claude route uses its own
conversation mechanism.

## Claude conversation compaction

Claude Code owns its conversation. When a turn and all its tools have finished,
Unreal checks the most recent context measurement. Above
`Context.ClaudeCompactThresholdTokens` it resumes the same Claude Code conversation
in the background with `/compact` and preservation guidance. The default threshold
is 200,000 tokens.

New user input or a child notice interrupts and reaps the idle compaction process
before starting the next turn. The original conversation remains available; a
later idle period can retry. Only context above twice the threshold triggers
blocking compaction at turn start. Processes using the same conversation are
serialized. Input remains usable while the TUI shows compaction activity.

Successful, failed, and interrupted attempts record timing, reasons, and usage
when supplied. The footer uses the latest `message_start` input plus cache-read
and cache-creation counts as context size, rather than summing a whole tool
loop. Usage totals and cache statistics still accumulate calls.

## Project Instructions

The harness discovers the workspace root's `AGENTS.md` when creating a Session
and persists its exact text as a Project Instructions snapshot. Codex reuses that
snapshot on resume, restart, fork, and child creation. A new Session reads the
current file. Older Sessions without such a record do not discover one on resume.

The harness does not scan nested instruction files or expand imports for this
snapshot. [Discovery](../harness/projectinstructions/discover.go) uses `os.ReadFile`:
a missing file binds no instructions, other read failures fail creation, and
NUL bytes or invalid UTF-8 are rejected. It does not promise no-symlink,
hard-link, or size-limit checks. Snapshot validation checks schema/source/path;
it does not recompute content digest and length on load. Public inspection shows
metadata, while canonical storage retains the text.

Claude differs: Claude Code reads `CLAUDE.md` itself. On each CLI invocation,
Unreal reads the current workspace `AGENTS.md` and appends it unless `CLAUDE.md`
already contains the string `AGENTS.md`. This is a string check, not proof that
an import was evaluated. Thus editing instructions can affect a later Claude
invocation in an existing Unreal Session, while Codex keeps its saved snapshot.
See [Claude prompt composition](../harness/llm/clients/claudeagent/prompt.go).

The Codex system layers are harness preamble, caller system prompt, Project
Instructions, and skill catalog. Skills load on demand. Instructions guide model
behavior and do not grant permissions, tools, credentials, or Session authority.

## Observe results

Use `/analyze`, then Context or Usage, or export the analysis. Reports include
average input across reported calls (including summaries), omitted result count,
estimated saved tokens, prune boundaries, cache ratios, and Claude compact
records. Recording-time head/tail reduction is not a prune count. Unknown usage
remains unknown; no cache percentage is guaranteed. See
[Troubleshooting](troubleshooting.md#read-analysis-and-measurements).
