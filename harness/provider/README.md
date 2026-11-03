# Provider Registry and Composition (`harness/provider`)

**Language:** English | [日本語](../../docs/ja/harness/provider/README.md)

`harness/provider` manages the catalog of available LLM providers and models, resolves non-secret runtime selections (`Selection`), and constructs authenticated LLM adapter clients per request.

---

## Key Responsibilities

1. **Explicit Model Catalog (`provider.Defaults(catalog)`):**
   - The embedding application supplies the explicit model catalog; this package never invents model availability or falls back to unadvertised models.
2. **Non-Secret Selection Metadata (`Registry.Build`):**
   - Resolves a versioned `Selection` containing the endpoint, non-secret `credential.Reference`, retry limit (`max_attempts`), declared model family/capabilities, and selection source.
   - This `Selection` is safe to persist in Session and Host metadata.
3. **Per-Request Credential Resolution:**
   - Resolves actual credential material at the start of each logical model request.
   - External Codex OAuth files are read in read-only mode on each turn.
   - No automatic 401 request replay or cross-provider fallback occurs.
4. **Enforced HTTP Transport:**
   - Uses the Host's permission-enforcing HTTP transport with HTTP redirects disabled on authenticated requests.
   - Session-local `.env` files never mutate the process environment.
