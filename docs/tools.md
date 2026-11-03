# Tools and file mutation

The [fixed Registry](../harness/tool/registry.go) composes native file tools,
AST tools, Bash, ViewImage, SkillUse, and observed Claude Code results. Availability
also depends on runtime configuration and permissions. Claude Code uses its own
built-in tools; `claude_code` records their reported outcomes. This document
describes Unreal's native/AST execution path.

Translators validate arguments and encode immutable Operation plans. Executors
perform I/O asynchronously. Typed results are retained in the RemoteJob handle;
a display summary is not decoded as the authoritative result.

## Native files

| Tool | Request and behavior |
| --- | --- |
| `read` | `path`; optional `start_line`, `lines`, `max_bytes`. Returns a whole-file revision even for a range |
| `write` | `path`, `content`, `expected`. Creation uses `{"exists":false}`; replacement uses the read revision |
| `edit` | `path`, `old_text`, `new_text`, `expected`; repeated matches require `replace_all:true` |
| `grep` | `path`, `pattern`; optional `glob`, `limit`, `max_bytes`. Uses Go regular expressions |
| `glob` | `path`; optional `glob`, `limit`, `max_bytes`. Matches paths deterministically |

See the [translator](../harness/tool/native/native.go) and
[executor](../harness/native/native.go). A tool request omits executor-owned
`version` and `action`. Unknown request members are rejected.

`max_bytes` defaults to 16 KiB; search `limit` defaults to 200. These bound
returned content/results. They do not impose a fixed file-size, visited-entry,
or maximum result-count ceiling. Reads load a complete snapshot before selecting
output. Binary reads return revision metadata without text bytes.

Search uses sorted breadth-first traversal, skips symlinks, and checks read
permission at directories/files. Glob syntax includes `/`, `*`, `?`, character
classes, and `**` recursion. Grep counts binary/unreadable candidates as skipped.
A typed `truncated` flag identifies output stopped by requested limits.

Native paths remain inside the immutable Session workspace, even in Full Access.
Use workspace-relative paths. Creation requires an existing parent. A revision
is `{"exists":true,"digest":"..."}` with the lowercase SHA-256 of the whole
snapshot. Copy the actual revision returned by `read`; never infer it from a
visible range. A stale revision returns a conflict instead of overwriting newer
bytes. Validation failures expose typed reasons such as workspace boundary,
invalid revision, or permission denial rather than raw filesystem errors.

## AST search and editing

`ast_grep` and `ast_edit` use Tree-sitter with Go, JavaScript, and Python grammars.
The request supplies `language`, `query`, and `targets` with a `path` for each.
Each match must have exactly one `@match` capture. For example:

```text
((call_expression
  function: (identifier) @callee
  arguments: (arguments) @args) @match
 (#eq? @callee "old"))
```

For this JavaScript query, replacement `updated${args}` changes `old(1)` to
`updated(1)`. Named captures insert exact source bytes; `$$` inserts a dollar.
Unknown/ambiguous captures, unsupported predicates, and overlapping ranges are
rejected. Replacement files are parsed again; syntax validation does not prove
type correctness or behavior.

1. Preview with `ast_edit` and its `replacement`. `apply` defaults to false.
2. Read the returned per-file revisions, match count, ranges, and excerpts.
3. To apply, supply `apply:true`, the preview's `expected_matches`, and every
   target's `expected` revision.
4. Run the relevant compiler/tests to verify program behavior.

The [AST executor](../harness/ast/ast.go) checks targets/counts and sends
replacements through mutation. All targets participate in revision validation,
including no-match files in a successful batch. Output defaults to 16 KiB.
[Parser/query calls](../harness/ast/engine.go) use a 500 ms timeout and reject
incomplete/timed-out work. Cancellation is checked between calls and matches.
There is no separate 250 ms rejection threshold.

AST execution needs `CGO_ENABLED=1` and a C compiler. A build with
`CGO_ENABLED=0` can run other paths, but AST returns
`unsupported_backend_cgo_required`. Distribution snapshots keep CGO enabled.

## Shared mutation guarantees

The [mutation service](../harness/mutation/mutation.go) serves native, AST, and
LSP changes. All cooperating writers must use the same canonical workspace and
shared private mutation state directory. Locks live outside the replaced inode.
Authorization runs while holding the mutation lock. Snapshot reads authorize
independently.

On Linux/macOS with advisory file locks, same-directory atomic rename, and
file/directory fsync, the service:

- Normalizes relative aliases and acquires multi-file locks in canonical order.
- Refuses symlinks in target paths and multiply-linked files, using pinned
  directory descriptors for replacement.
- Checks target revisions and permissions before replacing the first file.
- Replaces each file atomically and reports each target's outcome.

Multi-file mutation is not a crash-atomic transaction and does not roll back.
A later failure or cancellation can leave applied, failed, not-applied, or
indeterminate targets. Remote/distributed filesystems are outside this guarantee.
External directory renames, mount changes, and hostile filesystem administration
need separate isolation.

The Coordinator persists the immutable plan before dispatch. The executor writes
private intent/outcome proofs tied to Session/Operation identity and plan hash.
Recovery returns saved outcome evidence; a missing outcome is indeterminate and
is not blindly retried. Reusing an identity with a different plan fails. Keep
proofs while their Sessions may still be recovered. They support effect recovery;
the Session store remains the conversation authority.

Bash, external editors, and debugger targets can write outside this cooperating
writer protocol. Full Access does not extend mutation guarantees to them.
