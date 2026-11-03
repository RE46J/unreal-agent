# Tools and File Mutation

**Language:** English | [日本語](ja/tools.md)

Unreal's [Tool Registry](../harness/tool/registry.go) provides native file tools, Tree-sitter AST tools, `Bash`, `ViewImage`, `SkillUse`, and observed `claude_code` records.

- **Codex (`openai-codex`) and Sub-agents:** Use Unreal's native/AST/Bash execution pipeline described below.
- **Claude (`claude`):** Uses Claude Code's built-in tools (recorded as `claude_code` Operations), with declined Bash commands routed to Unreal's Bash executor.

---

## 1. Native File Tools

Native file tools operate strictly inside the Session's workspace root directory—even when `Full Access` is enabled. Always use workspace-relative paths.

| Tool | Parameters & Behavior |
| --- | --- |
| `read` | `path`; optional `start_line`, `lines`, `max_bytes`. Reads the file and returns a whole-file SHA-256 `revision` (even when reading a line range). |
| `write` | `path`, `content`, `expected`. Writes a whole file. Use `{"exists":false}` to create a new file, or pass the `revision` from `read` to overwrite an existing file. |
| `edit` | `path`, `old_text`, `new_text`, `expected`; optional `replace_all:true`. Replaces exact text when the file matches `expected` revision. |
| `grep` | `path`, `pattern` (Go regex); optional `glob`, `limit`, `max_bytes`. Searches file contents deterministically. |
| `glob` | `path`; optional `glob`, `limit`, `max_bytes`. Lists matching file paths in sorted breadth-first order. |

### Revision Safety & Limits

- **Optimistic concurrency control (`revision`):** Every `read` returns `{"exists":true,"digest":"<sha256>"}` for the entire file. Passing this object as `expected` in `write` or `edit` guarantees that Unreal will reject the edit (`revision_conflict`) if the file was modified in the meantime.
- **Request validation:** `version` and `action` are owned by the tool/executor and must not appear in model arguments; unknown fields are rejected. New files require an already existing parent directory.
- **Output bounds:** `max_bytes` defaults to `16 KiB` and search `limit` defaults to `200` matches. If output hits a limit, the result sets `truncated: true`. Output limits bound returned text, not the input file size or revision hashing.
- **Glob & symlink rules:** Search (`grep` / `glob`) skips symlinks. Glob patterns use `*`, `?`, and character classes within a path component; a whole `**` component matches zero or more directory components. Reading a binary file returns its revision metadata without raw binary bytes.
- **Typed failure reasons:** Failures include typed reasons such as `workspace_boundary`, `parent_missing`, `invalid_revision`, and `revision_conflict` (`code=invalid` alone does not mean permission denial).

See [`harness/tool/native/native.go`](../harness/tool/native/native.go) and [`harness/native/native.go`](../harness/native/native.go).

---

## 2. AST Search and Editing (`ast_grep` and `ast_edit`)

`ast_grep` and `ast_edit` use **Tree-sitter** to search and refactor **Go**, **JavaScript**, and **Python** code structurally.

### Query Format

Each query must define exactly one `@match` capture in every match for the node to match or replace, and can define named sub-captures (such as `@callee` or `@args`) for predicates or replacement templates:

```text
((call_expression
  function: (identifier) @callee
  arguments: (arguments) @args) @match
 (#eq? @callee "old"))
```

With the query above, a replacement template of `updated${args}` rewrites `old(1)` to `updated(1)` (`$$` inserts a literal `$`).

- **Validation & timeouts:** Ambiguous captures and overlapping match ranges are rejected. Parser and query execution each have a 500 ms budget; timed-out partial results are not accepted as complete edits.
- **Preview output limit:** Preview output defaults to `16 KiB`; clipping previews does not reduce the match count or the set of target revisions checked.

### Safe Two-Step Edit Workflow (`ast_edit`)

1. **Preview (`apply: false`, the default):** Run `ast_edit` with `language`, `query`, `targets`, and `replacement`. Unreal parses the source files and returns per-file `revision` digests, match counts, byte ranges (`start_byte` / `end_byte`), and previews without modifying disk. Replacement text is assembled without reparsing it; valid replacement syntax is not guaranteed.
2. **Apply (`apply: true`):** Re-run `ast_edit` with `apply: true`, passing `expected_matches` from the preview and every target file's `expected` revision, including targets with no matches.
3. **Verify:** Run your project's compiler and tests to verify types and runtime behavior.

> **Build Requirement:** AST tools require `CGO_ENABLED=1` and a C compiler. Release and snapshot builds always enable CGO. (In a `CGO_ENABLED=0` build, AST tools return `unsupported_backend_cgo_required`.)
>
> See [`harness/ast/ast.go`](../harness/ast/ast.go) and [`harness/ast/engine.go`](../harness/ast/engine.go).

---

## 3. Shared Atomic Mutation Guarantees

Native file writes/edits, AST edits, and LSP workspace edits all pass through the shared [Mutation Service](../harness/mutation/mutation.go):

### What the Mutation Service Guarantees (Linux / macOS)

- **Canonical ordering & locking:** Resolves relative paths and acquires advisory file locks across all target files in canonical order (stored in a shared private mutation directory outside the target files).
- **No symlink traversal:** Uses pinned directory file descriptors (`os.Root` / `O_NOFOLLOW`) and refuses symlinks or hard-linked targets.
- **Preflight before write:** Verifies every target file's `expected` SHA-256 revision and permission policy while holding the locks before modifying the first file.
- **Per-file atomic replacement:** Writes to a temporary file in the same directory, `fsync`s, and atomically renames over the target.
- **Crash recovery without blind replay:** Before dispatching an edit, the Coordinator records the immutable Operation plan. The mutation service writes durable intent and outcome proofs tied to the `Session ID + Operation ID + Plan Hash`. If a crash occurs mid-operation, recovery inspects the proof: completed edits return their recorded result, and interrupted edits with unknown state are marked `indeterminate` rather than blindly re-applied.

### What It Does Not Guarantee

- **Not a multi-file rollback transaction:** Each individual file replacement is atomic, but if a batch edit fails or is canceled partway through, earlier files in the batch are not rolled back; each file's exact status (`applied`, `failed`, `not_applied`, `indeterminate`) is reported.
- **External writers:** `Bash` commands, external text editors, and debuggers write directly to the OS filesystem outside this locking protocol.
- **Cooperating writer requirements:** All cooperating writers must use the same canonical workspace root and shared durable mutation state directory. Retain receipts and proofs for as long as their Operations may be recovered, and never reuse an Operation identity for a different plan.
- **Filesystem scope:** These guarantees target cooperating local filesystems; remote/distributed filesystems, mount changes, and external directory renames are outside the guarantee.
