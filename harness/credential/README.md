# Credential Store and Expiry (`harness/credential`)

**Language:** English | [日本語](../../docs/ja/harness/credential/README.md)

This package defines safe credential references, secret redaction, cross-process storage locking, and credential expiry checks.

---

## Key Design Rules

1. **Non-secret references in metadata (`Reference`):** Session, Operation, and Host records store only `credential.Reference`—never raw secret values.
2. **Redacted values (`Secret`):** `Secret` automatically redacts itself in `fmt` and JSON output. Raw material is accessed only via `Reveal()` at the storage or HTTP boundary.
3. **Cross-process atomic store (`Local`):**
   - Implemented for Linux and macOS using private `0700` directories, regular `0600` files, symlink refusal (`O_NOFOLLOW`), `flock`, atomic rename, and `fsync`.
   - Lock files are retained on disk so `unlink` cannot split lock ownership across processes.
4. **Expiry handling (`Manager`):**
   - There is no `RefreshFunc` or managed token rotation. Expired managed OAuth credentials return `reauth_required`; expired external credentials return `external_reauth_required`.
   - `RefreshPending` / `refresh_pending` is decoded only for older record compatibility; it does not implement an active refresh protocol.
   - External CLI credentials remain owned and refreshed by the CLI, outside this store.
5. **No reactive 401 model replay:** Model requests resolve valid credential material before the first HTTP call. If a server returns `401`, Unreal surfaces a normalized error rather than automatically replaying the model request.
6. **Custom `Store` contract:** Replacement `Store` implementations must serialize `WithCredential` transactions across processes sharing the account, reread while holding the lock, and commit atomically. The `Transaction` is valid only during its callback.
