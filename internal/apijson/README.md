# JSON Codec for Generated OpenAI Types (`internal/apijson`)

**Language:** English | [日本語](../../docs/ja/internal/apijson/README.md)

`internal/apijson` provides the deterministic JSON marshaling and unmarshaling helpers used by the generated OpenAI Responses API wire types in `internal/openaiapi`.

---

## Key Behaviors and Limitations

- **Precision preservation:** JSON numbers decoded into `any` are preserved as `encoding/json.Number` instead of losing precision via `float64`.
- **Deterministic encoding:** Struct and map serialization is deterministic.
- **Scoped strictly to wire types:** These rules apply only to the generated OpenAI Responses API codec. Canonical Session storage, Tool Translators, Permissions, and credentials keep their own strict codecs.
- **Known limitations of generated types:**
  - Concrete Go float fields (`float32`/`float64`) retain standard Go precision limits.
  - Nullable pointer fields cannot distinguish between an omitted JSON field and an explicit `null`.
  - Union `As...` accessors do not automatically validate enum discriminators—call `.Valid()` explicitly where required.
  - Standard structs drop unknown JSON fields, while raw union/map types preserve them.

When upgrading the OpenAPI code generator, refresh [`imports.tmpl`](imports.tmpl) from the [upstream `oapi-codegen` template](https://github.com/g-logunov/oapi-codegen/blob/0e050ab7608663a1ab9b243270f25181a4c88f22/pkg/codegen/templates/imports.tmpl) while keeping `encoding/json` replaced by `internal/apijson`.

A nil required nullable pointer encodes as `null`; a nil optional nullable pointer is omitted. Decoding does not validate every OpenAPI schema constraint (such as all enum or union rules); callers must perform the required validation.
