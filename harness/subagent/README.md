# Asynchronous Child Agents (`harness/subagent`)

**Language:** English | [日本語](../../docs/ja/harness/subagent/README.md)

`harness/subagent` manages asynchronous child agent subprocesses (`unreal-agent child --stdio`), bidirectional parent-child messaging, and durable lifecycle recovery.

---

## 1. Parent & Child Composition

### Parent Host Setup

1. Retrieve the parent Session owner via `host.SessionFromContext(ctx)`.
2. Construct `subagent.NewManager(ctx, Config{Owner: owner, Directory: sessionDirectory, Binary: absoluteAgentBinary, Arguments: []string{"child", "--stdio"}, Templates: allowedTemplates})`.
3. Register the manager with `LocalOperationManager`, add `tool/subagent.Extensions(owner.ID, templates, false)` to the tool registry, and call `Close` when the parent runtime shuts down.

- **Template security:** Binary paths, arguments, provider/profile selections, and capability bounds are fixed in trusted Host templates (`Subagents` config). The model can only pick a registered template name and supply a task string.
- **Permission intersection:** Child tool capabilities are always a subset of the parent's policy. No credentials are serialized into child plans.

### Child Process (`unreal-agent child --stdio`)

Calls `subagent.Serve(ctx, os.Stdin, os.Stdout, childFactory)`:
- Opens a child Host (`Lifecycle="child"`, `Heartbeat=0`) with tools restricted to bounded executors plus `SendParent` and `Finish`.
- Inherits the parent's `ProjectInstructionSnapshot` (`AGENTS.md` snapshot) and shared `MutationStateDirectory` so multi-process file locking stays coordinated.

---

## 2. Wire Protocol, Persistence, and Completion

- **Deterministic Child ID:** Derived from the parent Session ID and public `SubagentStart` Operation ID.
- **Length-prefixed JSON wire protocol:** 4-byte big-endian length header followed by versioned JSON (up to 256 KiB per frame) over stdio pipes.
- **Commit-before-ACK peer messaging:** Parent-child messages are persisted as canonical `InputPeer` records before an ACK is returned, preventing spoofed or lost messages.
- **Explicit `Finish` contract:** Ending a normal model turn puts a child into a waiting state—it does **not** complete the child. Completion requires an explicit `Finish` tool call (which is rejected if any other tool calls or Operations are still pending) and is recorded as a canonical `HostRecord(kind=finish)`.
- **Graceful cancellation:** Persists a cancel control input and `canceling` checkpoint, sends a stop request to the child, and escalates to process-group termination if the child does not exit within 2 seconds.

---

## 3. Operator Control APIs

- `ReadChild` — Reads a bounded canonical history page, active Operations, `ChildConfig`, and `Finish` status.
- `SteerChild` — Sends a follow-up message to a running child via a durable parent operation-intent input (with idempotent `InputID`).
- `CancelChild` — Cancels a running child Operation.
- `ResumeChild` — Verifies that a live parent Host owns and has recovered a non-terminal child (never spawns duplicate child processes).
