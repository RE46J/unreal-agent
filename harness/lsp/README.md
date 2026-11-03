# Managed Language Servers (`harness/lsp`)

**Language:** English | [日本語](../../docs/ja/harness/lsp/README.md)

The `harness/lsp` package implements Language Server Protocol (LSP 3.17) stdio framing, server lifecycle management, and safe workspace refactoring backed by the shared [Mutation Service](../mutation/mutation.go).

Protocol specification: [LSP 3.17 Specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)

---

## 1. Host Composition & Runner Configuration

### Host Integration

Create one `lsp.Manager` per workspace ownership lifetime using the same `mutation.Service` instance as native file tools:
- Configure each language explicitly with an absolute server binary path, arguments, environment, and file extensions (no automatic binary discovery or package installation occurs).
- Attach `lsp.Definition()` and `lsp.Translator{}` to each Session's tool registry, and `lsp.NewHandler(ctx, manager, sessionID)` to its Operation Manager.
- Close the Session handler before releasing the Session writer lock, and close `lsp.Manager` when the workspace Host shuts down.
- For multi-session Hosts, `agentrunner.NewLanguageTools` provides a `Wrap` / `Close` helper around the session tool factory.
- **Permission requirements:** The Manager policy is intersected with each request policy. Starting language servers requires unrestricted process/filesystem/network authority; restricted process policies return `unsupported`.

### Runner Environment Variable (`UNREAL_HARNESS_LSP_SERVERS`)

```json
[{"language":"go","path":"/opt/bin/gopls","arguments":["serve"],"extensions":[".go"]}]
```

The `LSP` tool is exposed to the model only when at least one server is configured and `LSP` is not in `disallowed_tools`.

---

## 2. Tool Surface & Position Encoding

The `LSP` tool accepts `language`, `action`, and action-specific parameters:
- **Inspection actions:** `definition`, `references`, `diagnostics`, `document_symbols`, `workspace_symbols`, `hover`, `code_actions`
- **Refactoring actions:** `rename`, `apply_code_action`
- **Lifecycle actions:** `restart`, `shutdown` (require the last observed process `generation`)

### Synchronization & Positions

- **Zero-based positions:** Default to `UTF-16`; explicit `UTF-8` or `UTF-32` is also supported and converted safely without splitting Unicode code points.
- **Document sync:** Sends `didOpen` followed by versioned full-text `didChange` whenever a file's SHA-256 digest changes. Diagnostics are matched against the exact document version (using pull diagnostics when supported).
- **Server requirements & staleness:** The server must support open/close notifications and full or incremental document changes. If a file changes on disk while a semantic request is in flight, the request returns `stale`.

---

## 3. Safe Workspace Edits (`rename` and `apply_code_action`)

All LSP workspace edits are executed through `mutation.Service`:

1. **Authorized snapshot:** Before `rename` or `code_actions`, the client captures a workspace snapshot and synchronizes modified text documents.
2. **Validation:** Rejects out-of-workspace URIs, `.git` metadata, overlapping edit ranges, mismatched document versions, and non-edit commands (`executeCommand`, file deletes/renames).
3. **Atomic preflight & execution:** Converts the `WorkspaceEdit` into a single `Mutation` request with each file's observed SHA-256 precondition and the stable `Session ID + Operation ID`.
4. **Crash recovery:** If a crash interrupts an edit, recovery checks the durable Mutation receipt before starting any language server—returning the proven outcome or `indeterminate` without re-applying edits.

### Edit Constraints & Code-Action Lifecycle

- **`CreateFile` restrictions:** Supported only when the target file does not yet exist, its parent directory already exists, and no `overwrite` / `ignoreIfExists` options are set.
- **Unsupported edit types:** Annotated edits, edits targeting unobserved existing files, unresolved or disabled code actions, and code actions that require `executeCommand` are rejected.
- **Ephemeral action IDs:** Discovered code-action IDs expire on the next discovery or server generation change and are consumed when applied (re-verifying stored file revisions and document versions upon application).
- **Post-edit sync failure:** If a file edit commits to disk but subsequent server synchronization fails, the edit remains `applied` with a refresh notice—the file is never rolled back or blindly retried.

---

## 4. Request Lifetime and Result Sizing

Requests use their execution context's cancellation/deadline. The current configuration does not define fixed resource caps or a default per-request timeout. Semantic results are returned as JSON without the previously claimed size rejection or truncation.
