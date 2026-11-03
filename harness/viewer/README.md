# Child Session and Operation Viewer (`harness/viewer`)

**Language:** English | [日本語](../../docs/ja/harness/viewer/README.md)

`harness/viewer` is a read-only observation and control client used by the TUI to project parent/child Sessions, live Operations, and token usage statistics. It never opens a Session store directly, acquires a writer lock, spawns processes, or calls an LLM.

---

## 1. Architecture and Frontend Integration

1. Construct a viewer model with `SubagentOptions()`, then call `NewClient(reader, model, controls)`.
2. `ParentReader` scopes reads to the attached parent Session and the direct children discovered from its canonical `SubagentStart` Operations.
3. `Watch(ctx, sessionID, onChange)` maintains a single live event subscription and triggers UI redraw callbacks whenever state changes (`Refresh` performs a one-shot read-only rebuild).
4. Use `Model.Select`, `Model.Rows`, `Model.Detail`, and `Client.History` (with `NextAfter` cursor pagination) to inspect child transcripts without mutating token usage aggregates.

Cancel the watch context and wait for `Watch` to return before replacing a watch for the same Session (`ErrWatching` otherwise). Callbacks run on the `Watch` goroutine and must return promptly; the model supports concurrent UI reads.

---

## 2. Source-of-Truth & Accounting Rules

- **Parent execution state:** Comes from the latest parent Operation record.
- **Child completion (`Finish`):** Comes strictly from the child's canonical `Finish` record (`HostRecord(kind=finish)`), never inferred from a stopped process or assistant text.
- **Liveness vs. persisted state:** Missing generation, disconnection, or an event gap marks liveness as `unknown` (`?`) and triggers a clean resync from canonical snapshots.
- **Deduplicated token accounting:** Token usage sums persisted model responses deduplicated by `session/turn + response ID` (or sequence number), so reconnects or history replays never double-count tokens. Missing usage is marked `unknown` (`—`), never `0`.
- **Bounded memory:** The transcript cache keeps up to 128 items by default (max 4,096), and terminal control characters are stripped from all rendered strings.

Generation/revision changes and sequence gaps require resynchronization; paged history reads do not advance live cursors or duplicate usage. Incomplete history or missing fields yield partial accounting. Elapsed values are wall time between recorded start/end observations and include downtime, not CPU time. A persisted child snapshot proves recorded state, not that its runtime is still alive.

---

## 3. Child Controls and CLI Commands

When `HostControls` is attached, the TUI exposes child commands over the existing private Unix domain socket:
- `/child <ID>` — Focuses a child agent and refreshes its persisted snapshot once per second.
- `/child-history [after] [limit]` and `/child-next` — Pages through the child's persisted transcript.
- `/child-send <TEXT>`, `/child-cancel`, `/child-resume` — Forwards control requests to the owning parent Host with a stable `InputID`.
- `/child-retry` — Safely retries an uncertain child control request using its original `InputID`.
