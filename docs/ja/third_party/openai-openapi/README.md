# OpenAI OpenAPI 仕様書ベンダリング (`third_party/openai-openapi`)

**言語:** [English](../../../../third_party/openai-openapi/README.md) | 日本語

このディレクトリには、`internal/openaiapi` の Go 型定義を生成するためにベンダリングされた [`openai/openai-openapi`](https://github.com/openai/openai-openapi) の OpenAPI 仕様書が格納されています。

---

## 取得元とバージョン情報

- **リビジョン:** `dc708bbe9a149bc35132c567ef3a3fdd7a24ab49`
- **取得元 URL:** `https://raw.githubusercontent.com/openai/openai-openapi/dc708bbe9a149bc35132c567ef3a3fdd7a24ab49/openapi.yaml`
- **SHA-256:** `ab0c5306e390c64efbf50bbf71f02aa0dad2dafcaa96066a592186daa6103b87`
- **ジェネレータ:** [`g-logunov/oapi-codegen` @ `0e050ab76086`](https://github.com/g-logunov/oapi-codegen/tree/0e050ab7608663a1ab9b243270f25181a4c88f22)（`v2.8.0` + nullable-union 修正版、[`internal/apigen/go.mod`](../../../../internal/apigen/go.mod) でバージョン固定）

---

## Go 型定義 (`types.gen.go`) の再生成手順

Go 1.27 以上、Python 3.12 以上、および `PyYAML==6.0.3` が必要です。分離された Python 仮想環境を作成し、リポジトリのルートから以下を実行します。

```sh
python3 -m venv /tmp/unreal-openapi-generator
/tmp/unreal-openapi-generator/bin/python -m pip install 'PyYAML==6.0.3'
PATH=/tmp/unreal-openapi-generator/bin:$PATH go generate ./internal/openaiapi
go test ./internal/openaiapi ./internal/apijson ./internal/jsonoptions ./harness/llm/responsesapi
/tmp/unreal-openapi-generator/bin/python -B -m unittest discover -s internal/openaiapi -p '*_test.py'
```

### 補足

- スキーマへのパッチ処理は [`internal/openaiapi/generate.py`](../../../../internal/openaiapi/generate.py) 側で行われるため、ベンダリングされた `openapi.yaml` 自体は変更されません。
- リポジトリにコミットされている `types.gen.go` は、再生成した結果とバイト単位で完全一致する必要があります。
- 生成された型の JSON シリアライズ仕様と制限事項については [`internal/apijson`](../../internal/apijson/README.md) を参照してください。

メモリ内パッチは discriminator を実際の wire 値へ対応付け、同じ discriminator 値を持つ message variant の曖昧な mapping を除去し、再帰的な filter 参照を解決してから生成します。PyYAML の固定版は環境の要件であり、generator 自身が他のすべての版を拒否するわけではありません。
