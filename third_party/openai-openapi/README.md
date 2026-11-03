# Vendored OpenAI OpenAPI Specification (`third_party/openai-openapi`)

**Language:** English | [日本語](../../docs/ja/third_party/openai-openapi/README.md)

This directory contains the vendored OpenAI OpenAPI specification used to generate the Go types in `internal/openaiapi`.

---

## Upstream Provenance

- **Upstream Repository:** [`openai/openai-openapi`](https://github.com/openai/openai-openapi)
- **Commit Revision:** `dc708bbe9a149bc35132c567ef3a3fdd7a24ab49`
- **Source URL:** `https://raw.githubusercontent.com/openai/openai-openapi/dc708bbe9a149bc35132c567ef3a3fdd7a24ab49/openapi.yaml`
- **SHA-256:** `ab0c5306e390c64efbf50bbf71f02aa0dad2dafcaa96066a592186daa6103b87`
- **Code Generator:** [`g-logunov/oapi-codegen` @ `0e050ab76086`](https://github.com/g-logunov/oapi-codegen/tree/0e050ab7608663a1ab9b243270f25181a4c88f22) (`v2.8.0` + nullable-union fix), pinned in [`internal/apigen/go.mod`](../../internal/apigen/go.mod)

---

## Regenerating Go Types (`types.gen.go`)

Requires Go 1.27+, Python 3.12+, and `PyYAML==6.0.3`. Run from the repository root using an isolated Python virtual environment:

```sh
python3 -m venv /tmp/unreal-openapi-generator
/tmp/unreal-openapi-generator/bin/python -m pip install 'PyYAML==6.0.3'
PATH=/tmp/unreal-openapi-generator/bin:$PATH go generate ./internal/openaiapi
go test ./internal/openaiapi ./internal/apijson ./internal/jsonoptions ./harness/llm/responsesapi
/tmp/unreal-openapi-generator/bin/python -B -m unittest discover -s internal/openaiapi -p '*_test.py'
```

### Notes

- **In-memory schema patches ([`internal/openaiapi/generate.py`](../../internal/openaiapi/generate.py)):** Applied in memory so `openapi.yaml` remains unmodified. The patches map discriminators to actual wire values, remove ambiguous message-union mappings where variants share a discriminator value, and resolve recursive filter references before code generation.
- **Byte-for-byte reproducibility:** The checked-in `internal/openaiapi/types.gen.go` must match fresh generation output byte-for-byte. (The documented `PyYAML==6.0.3` pin is an environment requirement for reproducible output; the generator script itself does not reject every other installed PyYAML version.)
- **Serialization behavior:** See [`internal/apijson`](../../internal/apijson/README.md).
