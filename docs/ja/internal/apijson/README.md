# 生成 OpenAI API 型向け JSON コーデック (`internal/apijson`)

**言語:** [English](../../../../internal/apijson/README.md) | 日本語

`internal/apijson` パッケージは、`internal/openaiapi` の自動生成された OpenAI Responses API 型で使用される JSON のシリアライズ／デシリアライズ処理を提供します。

---

## 主な仕様と制限事項

- **数値精度の保持:** `any` 型へデコードされた数値は `float64` に丸めず `encoding/json.Number` として精度を保持します。
- **決定論的エンコード:** 出力される JSON は常に決定論的な順序になります。
- **適用範囲の分離:** このパッケージの既定動作は Responses API の通信型にのみ適用されます。正規 Session 履歴やツール Translator の検証、権限、認証処理はそれぞれの厳格なコーデックを維持します。
- **生成型の注意点:**
  - `float32` などの具体的な数値フィールドは Go の型精度に従います。
  - Nullable なポインタフィールドは「フィールド未指定」と「明示的な `null`」を区別しません（必須の nullable pointer が `nil` なら `null` として出力され、任意の nullable pointer が `nil` なら省略されます）。
  - 通常の struct は未知の JSON フィールドを破棄し、raw union / map 型は保持します。
  - デコード処理は OpenAPI の全スキーマ制約（enum や union のすべての規則など）を自動検査しません。Union 型の `As...` メソッドも判別子（discriminator）の妥当性を自動検査しないため、必要に応じて列挙型の `Valid()` メソッドを呼び出し側で明示的に実行してください。

コードジェネレータを更新する際は、[上流の `oapi-codegen` テンプレート](https://github.com/g-logunov/oapi-codegen/blob/0e050ab7608663a1ab9b243270f25181a4c88f22/pkg/codegen/templates/imports.tmpl)から [`imports.tmpl`](../../../../internal/apijson/imports.tmpl) を更新し、`encoding/json` のインポートを本パッケージに置き換えた状態を維持してください。
