# Prompt and Tool Profiles (`harness/profile`)

**Language:** English | [日本語](../../docs/ja/harness/profile/README.md)

`harness/profile` resolves model-family prompt and tool-description adjustments at Host composition time and records the resolved `Selection` in Host metadata alongside `provider.Selection`.

---

## Built-in Profiles

Profiles are matched against the explicitly declared provider/model family (never by loose substring matching on model names):

- **`conservative` (`v1`):** Default profile adding guidance on verifying observed file contents and SHA-256 revisions.
- **`openai-reasoning` (`v1`):** Tailored for declared OpenAI reasoning model families, adding bounded `read`/`edit`/`write` guidance to existing tool descriptions.

### Boundaries

`Compose` only adjusts prompt and tool description text. It preserves the enabled tool set, tool names, parameter schemas, and Host lifecycle rules, and has no access to permissions, credentials, or Session controls.
