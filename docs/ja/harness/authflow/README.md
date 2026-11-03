# 認証フロー管理 (`harness/authflow`)

**言語:** [English](../../../../harness/authflow/README.md) | 日本語

開発・評価用の認証方式管理と `unreal-agent-auth` CLI を提供するパッケージです。

> **補足:** 日常の対話型 TUI 利用では外部 CLI のサブスクリプション認証（`claude` / `codex`）を使用します。本パッケージの管理キーストアは開発・ベンチマーク評価用途向けです。

---

## 1. 認証方式の対応状況（2026-09-27 レビュー時点）

| プロバイダ / 方式 | 状態 | 所有者と推論経路 |
| --- | --- | --- |
| **OpenAI API key** | 対応 | ハーネスのプライベートストアで管理。既存の Responses アダプタを使用 |
| **Anthropic API key** | 設定保存のみ対応 | ストアへの保存は可能。推論には Anthropic アダプタの明示的な組み込みが必要 |
| **Claude.ai Pro/Max OAuth** | 非対応 | サードパーティによる直接の OAuth 仲介は規約上不可（公式 `claude` CLI を使用） |
| **OpenAI Codex 直接 OAuth** | 見送り | 外部 CLI が所有する認証情報（`~/.codex/auth.json` 等）を読み取り専用で参照 |
| **GitHub Copilot OAuth** | 見送り | 独立したランタイム契約が必要 |

### 公式ドキュメント参照先

- [OpenAI API key and Responses quickstart](https://developers.openai.com/api/docs/quickstart)
- [Codex authentication](https://learn.chatgpt.com/docs/auth)
- [Anthropic API authentication](https://platform.claude.com/docs/en/api/overview)
- [Claude Code credential policy](https://code.claude.com/docs/en/legal-and-compliance#authentication-and-credential-use)
- [Copilot SDK authentication](https://docs.github.com/en/copilot/how-tos/copilot-sdk/auth/authenticate)

---

## 2. `unreal-agent-auth` CLI の使い方

ビルドコマンド：

```bash
go build ./cmd/unreal-agent-auth
```

### サブコマンド一覧

フラグはサブコマンドの**後ろ**に指定します。

```bash
# 対応している認証方式の一覧を表示
unreal-agent-auth methods

# リダイレクトされたパイプまたはファイルから API キーを登録（端末エコー防止のため対話 TTY の標準入力は拒否されます）
unreal-agent-auth login --store /private/path --provider openai --id work --method api_key --key-stdin < /path/to/keyfile

# 保存済み認証のメタデータ一覧を表示（秘密値は出力されません）
unreal-agent-auth list --store /private/path

# 保存済み認証の削除
unreal-agent-auth logout --store /private/path --provider openai --id work --method api_key
```

### セキュリティとランタイムの原則

- **コマンド引数や端末エコーの禁止:** 秘密鍵はフラグ引数として渡せず、パイプやファイルから `--key-stdin` で渡す必要があります。
- **Session / 履歴への混入防止:** `LoginRequest` は `Inbox`・`Operation`・Session 履歴を経由せず、非秘匿の `Metadata` のみを返します。
- **`unreal-agent-runner` での利用:** `UNREAL_HARNESS_CREDENTIAL_DIRECTORY` と `UNREAL_HARNESS_CREDENTIAL_ID` を設定します。管理ストアの ID を指定した場合、環境変数の API キーへの暗黙フォールバックは行われません。
- **ログイン時の推論リクエストなし:** ローカル `login` の成功はキーの保存成功を意味するものであり、推論権限（entitlement）の確認ではありません（確認のためのモデルリクエストは送りません）。
- **入力前の方式検証:** 秘密値を要求する前に `ValidateMethod` を呼び出し、非対応の認証方式を入力前に拒否します。
