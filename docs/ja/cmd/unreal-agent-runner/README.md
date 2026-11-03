# unreal-agent-runner（ヘッドレス評価ランナー）

**言語:** [English](../../../../cmd/unreal-agent-runner/README.md) | 日本語

`unreal-agent-runner` は、1つのプロンプトまたは JSON リクエストをヘッドレス（画面なし）で実行し、正規 Session アイテムを JSONL 形式で標準出力（`stdout`）へ書き出して終了する開発・ベンチマーク評価用ツールです。

> **用途について:** 日常のサブスクリプション利用では、このツールではなく [共通キットのターミナル (`unreal`)](../../install.md) を使用してください。

---

## 1. ビルドと基本的な実行

[`go.mod`](../../../../go.mod) 指定バージョンの Go と、AST ツール用の C コンパイラ（`CGO_ENABLED=1`）が必要です。

リポジトリのルートで以下を実行します。

```bash
make build
bin/unreal-agent-runner -h
```

### OpenAI API キーを使った評価実行

既定のプロバイダは `openai` です（API キー経路は開発・評価用であり、従量課金が発生する場合があります）。シェル履歴にキーを残さずに実行するには以下のように入力します。

```bash
read -r -s -p 'Evaluation OpenAI API key: ' OPENAI_API_KEY
printf '\n'
export OPENAI_API_KEY
bin/unreal-agent-runner -p 'Inspect this project and explain how to run its tests.'
```

### ワークスペースの指定と JSONL ファイルへの保存

```bash
bin/unreal-agent-runner -workspace . -p 'Summarize this project.' > run.jsonl
```

> **セキュリティ上の注意:** `unreal-agent-runner` 自体は OS サンドボックスではありません。評価実行時は適切に隔離されたコンテナやベンチマーク環境を使用してください。

---

## 2. JSON リクエストと環境変数設定

`-p` フラグの代わりに、引数または標準入力（`stdin`）から JSON リクエストを渡すことができます。

```bash
bin/unreal-agent-runner '{"prompt":"Summarize this project."}'
printf '%s\n' '{"prompt":"Summarize this project."}' | bin/unreal-agent-runner
```

### リクエスト形式とフラグ

- コマンドラインオプションは位置引数の JSON より前に記述します（`-p` と JSON 引数は同時に指定できません）。
- 未知の JSON フィールドはエラーとして拒否されます。
- 主なフィールド: `prompt`, `messages`, `model`, `thinking_level`, `session_id`, `system_prompt`, `max_attempts`, `disallowed_tools`（全スキーマは `bin/unreal-agent-runner -h` で確認できます）。

### セッション保存先とプロバイダ選択

- **セッション保存先:** 既定は `${XDG_STATE_HOME:-$HOME/.local/state}/unreal-agent/sessions` です（`-session-directory` で変更可能）。`session_id` を指定するとセッションの作成・再開が可能です。`-log-directory` で追加の JSONL ログ出力先も指定できます。
- **プロバイダとモデルの環境変数:**
  - `UNREAL_HARNESS_LLM_PROVIDER`: `openai`（既定）、`openai-codex`（外部 Codex 認証を使用）、`openrouter`、`fireworks`、`ollama`
  - `UNREAL_HARNESS_LLM_MODEL`: 使用するモデル名

プロバイダ定義の詳細は [`cmd/internal/agentrunner/providers.go`](../../../../cmd/internal/agentrunner/providers.go)、ベンチマーク評価の手順は [Harbor アダプタ](../../benchmarks/harbor/README.md) を参照してください。
