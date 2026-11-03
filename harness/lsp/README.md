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

---

## 3. Safe Workspace Edits (`rename` and `apply_code_action`)

All LSP workspace edits are executed through `mutation.Service`:

1. **Authorized snapshot:** Before `rename` or `code_actions`, the client captures a bounded workspace snapshot and synchronizes modified text documents.
2. **Validation:** Rejects out-of-workspace URIs, `.git` metadata, overlapping edit ranges, mismatched document versions, and non-edit commands (`executeCommand`, file deletes/renames).
3. **Atomic preflight & execution:** Converts the `WorkspaceEdit` into a single `Mutation` request with each file's observed SHA-256 precondition and the stable `Session ID + Operation ID`.
4. **Crash recovery:** If a crash interrupts an edit, recovery checks the durable Mutation receipt before starting any language server—returning the proven outcome or `indeterminate` without re-applying edits.

---

## 4. Resource Bounds

- **Protocol framing:** Max 4 MiB message body, 8 KiB header
- **Semantic result size:** Max 256 KiB JSON (oversized results return a typed `limit` error rather than truncated JSON)
- **Workspace sync bounds:** Up to 2,048 synchronized documents / 16 MiB total text
- **Limits:** Up to 32 configured servers, 128 cached code-action candidates, 64 cached push-diagnostic documents, 30s default request timeout
