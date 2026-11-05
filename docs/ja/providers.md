# プロバイダ設定 (Providers)

**言語:** [English](../providers.md) | 日本語

Unreal Agent は1つのインストール環境と共通の `/model` メニューで **Claude** と **Codex** の両方に対応しています。どちらも既存のサブスクリプションログインを使用し、プロバイダの自動切替や API キーへのフォールバックは行いません。OpenCode Go は、管理ストアに保存した API キーで追加できます（[OpenCode Go](#opencode-go-api-key) 参照）。

---

## モデルの選択と切り替え (`/model`)

1. TUI で `/model` を入力し、プロバイダ（`Claude` または `Codex`）を選択します。
2. 利用可能なモデルを選び、続けて推論の深さ（effort）を選択します。
3. Unreal の起動中に外部 CLI でログインし直した場合や CLI を更新した場合は、`/model refresh` を実行する（または `/model` で未利用のプロバイダを選択する）と認証状態とモデル一覧を再検出します。

> **補足:**
> - 初回起動時は、両方使える場合は **Claude**、片方のみの場合は使える側のプロバイダが初期選択されます（環境変数 `UNREAL_HARNESS_LLM_PROVIDER` / `UNREAL_HARNESS_LLM_MODEL` で明示指定も可能）。
> - 未ログインのプロバイダもメニュー内に理由と準備手順付きで表示されます。
> - モデル一覧に表示されていても、実際に推論できるかはご自身のサブスクリプション契約プランに依存します。

---

## Claude（Claude Code CLI 連携）

### セットアップ手順

1. Unreal を実行する Linux 環境に公式の `claude` CLI をインストールします。
2. 別の Bash シェルで `claude` を起動し、Claude Code 内で `/login` を実行して Claude サブスクリプションでログインします。
3. Unreal に戻り、`/model refresh` を実行します。

### 仕組み

- **プロバイダ ID:** `claude`
- **会話とツールの管理:** Claude Code 自身が会話履歴と内蔵ツールを管理します。Unreal は `claude` CLI を起動してプロンプトを渡し、実行されたツールの結果を `claude_code` Operation として記録します。
- **権限と Bash フォールバック:** Claude Code の内蔵ツール呼び出しはすべて Unreal の承認フックを通過します。Claude Code 自身が Bash コマンドの実行を辞退した場合は、Unreal のネイティブ Bash 実行器が同じ権限モードのもとで引き継いで実行し、結果を Claude に返します。
- **モデル一覧:** Claude Code の初期化レスポンスから自動取得されます。
- **会話圧縮と指示書:** [コンテキストと指示書](context.md) を参照してください。

---

## Codex（通常のサブスクリプション経路）

### セットアップ手順

1. 公式の `codex` CLI をインストールし、Bash で `codex login` を実行して ChatGPT サブスクリプションでログインします。
2. 初回導入時などにモデル一覧が未作成と表示された場合は、一度 `codex` CLI を開いてローカルの `models_cache.json` を生成させます。
3. Unreal を起動するか、既存セッションで `/model refresh` を実行します。

### 仕組み

- **プロバイダ ID:** `openai-codex`
- **実行パイプライン:** Unreal 内蔵の Codex アダプタを通じてリクエストを送り、Unreal のネイティブファイルツール・AST ツール・Bash を実行します。
- **認証情報の管理:** OAuth トークンの保存と更新は外部の `codex` CLI が担います。Unreal は認証ファイルを読み取り専用で参照し、独自のキーストアへトークンをコピーしません。
- **認証ファイルの探索順序:**
  1. `--codex-auth-file` フラグ（ソースビルド版ランチャー）
  2. `OPENAI_CODEX_AUTH_FILE` 環境変数
  3. `$CODEX_HOME/auth.json`
  4. `$HOME/.codex/auth.json`
  ※ 通常のプライベートファイルである必要があり、同じフォルダ内の `models_cache.json` からモデル一覧を読み取ります。

- **モデル一覧の取得規則:** 通常 Codex のモデル探索は、選択した認証ファイルと同じディレクトリの `models_cache.json` の読み取りのみを行います（`/model refresh` 時にも CLI を起動して列挙することはしません）。モデル一覧の取得と認証・推論権限の検査は独立しています。

---

<a id="docker-sandboxes-codex-oauth"></a>
## Docker Sandboxes Codex OAuth

既存の **Docker Sandbox** 内で Unreal を動かす場合、Docker の認証フォワードプロキシを経由してホスト側の OpenAI OAuth ログインを安全に利用できます。

### セットアップ手順

1. **Docker ホスト側:** Sandbox ツールで OpenAI OAuth ログインを設定します。
   ```bash
   sbx secret set openai --oauth
   ```
   詳しくは [Docker 公式の Codex セットアップ手順](https://docs.docker.com/ai/sandboxes/agents/codex/) を参照してください。
2. **Sandbox 内:** [インストール手順](install.md) に従って共通キットを導入します。既定の自動判定（`--codex-auto`）でも認識されますが、明示的に指定する場合は `bash development.sh` に `--docker-sandbox-oauth` を追加します。
3. Unreal を起動し、`/model` で **Codex** を選択します。

### 検証条件とモデルキャッシュの仕組み

- **環境と設定の検査:** `SBX_CRED_OPENAI_MODE=oauth`、非秘匿の認証マーカーファイル、`.codex/config.toml` の `sandboxd` プロバイダ設定、および Codex エンドポイント URL を検査します（設定内の `forced_login_method = "api"` はプロキシ経路用の指定であり、API キー課金にはなりません）。
- **フォワードプロキシ:** 環境変数 `HTTPS_PROXY`（または `https_proxy`）に `http://host.docker.internal:3128` または `http://gateway.docker.internal:3128` が設定されている必要があります（大文字・小文字で異なる値が設定されている場合は拒否されます）。プロキシが通信時にホストの認証情報を付与するため、Unreal がホストのトークンを読み取ったり証明書を導入したりすることはありません。
- **モデル一覧の取得とキャッシュ:**
  - まず `models_cache.json` を確認し、存在しない場合は `codex app-server` のメタデータ取得（`model/list`）のみを実行します。取得結果は `~/.cache/unreal-agent/codex-cli-catalog.json` に保存され、24時間再利用されます。
  - `models_cache.json` がない状態で `/model refresh` を実行すると、保存済み CLI カタログのキャッシュを迂回して再取得します。プロキシ経路を検証した後、取得が失敗・遅延した場合は最後の成功済みカタログを保持します。
  - 一覧取得は推論の認証ではなく、直接通信・別の認証源・API キー・他プロバイダへのフォールバックは行いません。

<a id="opencode-go-api-key"></a>
## OpenCode Go（API キー）

OpenCode Go は、管理ストアに API キーを保存すると追加できる 3 つ目のプロバイダです。接続先は OpenCode Go サブスクリプションのエンドポイント（`https://opencode.ai/inference/go/openai/v1`、Responses API）だけで、従量課金の Zen エンドポイントへは接続しません。利用できるのは、この窓口で実機に確かめた 23 個のモデルです（先頭は DeepSeek V4.1 Flash、`deepseek-v4.1-flash`）。Gemini 4 Argon、MiMo-V2.6-Flash、Contributor の 2 個（送った内容が学習に使われる）、Free のモデルは出しません。

### セットアップ手順

1. 管理ストア（`~/.local/state/unreal-agent/credentials`、プロバイダ `opencode-go`、id `primary`）に、`unreal-agent-auth login ... --key-stdin` でキーを保存します。TUI の `/login` に、その環境での実際のコマンドが表示されます。
2. `/model refresh` を実行します。`/model` に「OpenCode Go · available」と表示され、Host の再起動は不要です。
3. `/model` で選ぶと、その会話自身のモデルだけが切り替わります。モデルは 23 個から選び、effort があるモデルは続けて effort も選びます（一覧に effort がないモデルは、その段階が出ません）。

### 仕組み

- **プロバイダ ID:** `opencode-go`
- **登録:** config への登録は不要です。管理ストアにキーがある間だけ登録され、キーがなければ何も変わらず、`/model` に現れず、OpenCode へ何も送りません。
- **接続先:** 上記の Go サブスクリプションのエンドポイントだけに送ります。Zen や別プロバイダ、API キーへ自動で切り替えることはなく、選択したモデルが利用できない場合は要求が失敗します。

詳細は [OpenCode Go](../opencode-go.md) を参照してください。

---

<a id="delegated-gpt-work"></a>
## Delegated GPT work（GPT サブエージェントへの作業委譲）

Codex が登録され、Permission Modes が有効で `Maximum` が設定されており、明示的な `Subagents` 設定で既定値が置き換えられていない場合、以下のテンプレートが標準で登録されます。各テンプレートのモデルと推論の深さ（effort）は親セッションの選択モデルとは独立して固定されています。

| テンプレート / 役割 | モデル・Effort | 主な用途 |
| --- | --- | --- |
| `lead` | `gpt-6.1-sol` (`max`) | 実装作業の進行・統括（配下に `worker-*` や `collector` を起動可能） |
| `worker-low`, `worker-medium`, `worker-high`, `worker-xhigh`, `worker-max` | `gpt-6.1-sol`（各 effort） | コード実装および担当範囲のテスト実行 |
| `collector` | `gpt-6-luna` (`medium`) | 検索、ファイル・ログからの値抽出、URL/API 確認、CI 状態確認、読み取り専用の要約 |

### 委譲の動き

1. Claude に「サブエージェントに委譲して」「lead に実装を渡して」と指示します。
2. Claude は `~/.local/bin/unreal-agent subagent` コマンドでサブエージェントを起動してタスクを渡し、いったん自分のターンを終了します。
3. サブエージェントが完了または連絡を送ると、親セッションが自動的に再開して結果を処理します。
4. サブエージェントの権限モードは親セッションの選択モード（および上限ポリシー）に従い、承認要求は親エージェントが判断します。詳しくは [権限管理](permissions.md) を参照してください。

> **環境変数の継承:** 子は Host の環境（`PATH`、`HOME`、ツール設定など）を継承し、除去するのは `OPENAI_CODEX_ACCESS_TOKEN` と `OPENAI_CODEX_ACCOUNT_ID` のみです。環境隔離を保証するものではありません。

### サブエージェントのモデルを選ぶ（`/model-sub`）

`/model-sub` はサブエージェントが使うモデルを選びます。**OpenAI Codex**（既定、従来どおり）か、**OpenCode Go** → 実機で確かめた 23 個のモデル（先頭は DeepSeek V4.1 Flash）です。選択は `STATE/subagent-model.json` に保存され、このマシンのすべての Session に適用され、その後に開始する子から有効になります。実行中の子は自身のモデルを維持します。`/children start ...` で明示的に runtime を指定した子と、config ファイルで定義した template は影響を受けません。

OpenCode Go を選んだ場合、どのテンプレートも `/model-sub` で選んだモデルで動きます。DeepSeek V4.1 Flash では、これまでどおり次のとおりです。

| テンプレート | Effort |
| --- | --- |
| `lead`, `worker-max` | `max` |
| `worker-xhigh`, `worker-high`, `worker-medium` | `high` |
| `worker-low`, `collector` | `low` |

役割ごとの effort は、選んだモデルの一覧から決まります。`lead` と `worker-max` は一番高い effort、`worker-low` と `collector` は一番低い effort、他の worker は自分の effort があればそれ、なければ既定の effort（既定が自分より低いときは、自分より低い中で一番高い effort。`qwen3.8-max` では `worker-high` が `medium`）です。一覧に effort がないモデルは、effort なしで動きます。選択したモデルが利用できない場合（上限到達、障害）、子の起動または要求は失敗し、Codex や別プロバイダへ自動で切り替わることはありません。Codex なしで起動した Host では、テンプレートは Host 起動時に OpenCode Go が選択されていた場合のみ存在し、後からの選択は保存され次の Host 起動から使われます（`/model-sub` がその旨を表示します）。
