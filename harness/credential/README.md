# Credential Store and Refresh (`harness/credential`)

**Language:** English | [日本語](../../docs/ja/harness/credential/README.md)

This package defines safe credential references, secret redaction, cross-process storage locking, and token refresh coordination.

---

## Key Design Rules

1. **Non-secret references in metadata (`Reference`):** Session, Operation, and Host records store only `credential.Reference`—never raw secret values.
2. **Redacted values (`Secret`):** `Secret` automatically redacts itself in `fmt` and JSON output. Raw material is accessed only via `Reveal()` at the storage or HTTP boundary.
3. **Cross-process atomic store (`Local`):**
   - Implemented for Linux and macOS using private `0700` directories, regular `0600` files, symlink refusal (`O_NOFOLLOW`), `flock`, atomic rename, and `fsync`.
   - Lock files are retained on disk so `unlink` cannot split lock ownership across processes.
4. **Crash-safe token rotation (`Manager`):**
   - Before calling a provider's `RefreshFunc`, the manager acquires the per-account lock, rereads the current state, and persists a `refresh_pending` marker.
   - If a process crashes or fails mid-exchange before saving the new rotating token, the record remains in `refresh_pending` and requires re-authentication rather than reusing a potentially invalidated token.
   - `Logout` acquires the same account lock so an in-flight refresh can never resurrect a logged-out credential.
   - Externally owned credentials (such as `~/.codex/auth.json`) are never refreshed by this store.
5. **No reactive 401 model replay:** Model requests resolve valid credential material before the first HTTP call. If a server returns `401`, Unreal surfaces a normalized error rather than automatically replaying the model request.
