# Harbor ベンチマーク評価アダプタ

**言語:** [English](../../../../benchmarks/harbor/README.md) | 日本語

`unreal-agent-runner` を **Harbor 0.22.0** のエージェント（`unreal-agent`）として実行し、Terminal-Bench などのベンチマーク評価を行うアダプタです。

## 必要な環境

- ルートの [`go.mod`](../../../../go.mod) で指定された Go
- Python 3.12 以上、および `uv`
- Docker または Modal

---

## 1. セットアップとビルド

リポジトリのルートから以下を実行します。

```sh
make -C benchmarks/harbor sync
make -C benchmarks/harbor build REVISION=HEAD
```

コミット済みのソースコードから Linux amd64 向けバイナリをビルドし、リビジョンとチェックサムのマニフェストとともに `bin/harbor/<short-commit>/` へ出力します（既存のバンドルは上書きされません。別ディレクトリへ再ビルドする場合は `BUNDLE` を指定します）。

---

## 2. ベンチマークの実行

### 単一タスクの実行

ビルドしたバンドルへの絶対パスを指定して実行します。

```sh
export OPENAI_API_KEY=...
uv run --project benchmarks/harbor --locked harbor run \
  -p /absolute/path/to/task \
  -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-5.4 \
  --ak bundle="$PWD/bin/harbor/<short-commit>" \
  --ak thinking_level=high
```

### 対応プロバイダとオプション

- **`thinking_level`:** `low`, `medium`, `high`（既定）, `xhigh`, `max`
- **対応モデル指定:**
  - `openai/<model>`（`OPENAI_API_KEY`）
  - `openrouter/<model>`（`OPENROUTER_API_KEY`：1時間 TTL の自動プロンプトキャッシュとセッション固定が有効になります）
  - `fireworks_ai/<model>`（`FIREWORKS_AI_API_KEY`）
- 環境変数を追加で渡す場合は Harbor の `--ae` オプションを使用します。

### Modal 上での Terminal-Bench 4.0 実行例

Modal の認証設定を行ったうえで以下を実行します。

```sh
uv run --project benchmarks/harbor --locked --extra modal harbor run \
  -d terminal-bench/terminal-bench@4.0.0 \
  -e modal -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-6-astra \
  --ak bundle="$PWD/bin/harbor/<short-commit>" --ak thinking_level=max \
  -k 5 -n 40 --job-name tb4-unreal-agent
```

結果の確認：

```sh
harbor view jobs/<job-name>
```

---

## 3. 出力と利用ツール

- **有効なツール:** `Bash` および `ViewImage`（2000×2000 ピクセル以内・5 MB 未満の base64 画像を返し、`agent/images/` 配下にフォーマットと座標スケール情報を記録します）。
- **ログと軌跡データ:** `agent/` 配下にセッションとログが保存され、`agent/trajectory.json` に ATIF 形式の観測結果とトークン使用量合計が出力されます。

---

## 4. テストとスモーク検証

```sh
make -C benchmarks/harbor test check
make -C benchmarks/harbor smoke BUNDLE=../../bin/harbor/<short-commit>
```

スモークテストは Docker とローカルのテスト用モデルを使用して動作を確認します。

参考: [Harbor agents 公式ドキュメント](https://www.harborframework.com/docs/agents)
