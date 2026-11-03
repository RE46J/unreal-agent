# クレデンシャル保存と有効期限 (`harness/credential`)

**言語:** [English](../../../../harness/credential/README.md) | 日本語

認証情報の安全な参照（`Reference`）、秘密値の自動マスク（`Secret`）、プロセス間ロック付きストレージ、および認証情報の有効期限確認を行うパッケージです。

---

## 主な設計ルール

1. **メタデータには参照のみを保存 (`Reference`):** Session、Operation、Host のメタデータには非秘匿の `credential.Reference` のみを記録し、トークンやキーの実体は保存しません。
2. **秘密値の漏洩防止 (`Secret`):** `Secret` 型は JSON シリアライズや文字列フォーマット時に自動でマスクされます。実体の取り出し（`Reveal()`）はストレージおよび HTTP 送信境界でのみ行います。
3. **プロセス間ロック付きローカルストア (`Local`):**
   - Linux / macOS 向けに、`0700` ディレクトリ、`0600` 通常ファイル、シンボリックリンク拒否（`O_NOFOLLOW`）、`flock`、アトミック `rename`、`fsync` で実装されています。
   - ロックファイル自体は削除（unlink）せず保持することで、複数プロセス間でロック対象が分裂するのを防ぎます。
4. **有効期限の扱い (`Manager`):**
   - `RefreshFunc` や管理トークンのローテーションは実装されていません。期限切れの managed OAuth は `reauth_required`、external は `external_reauth_required` を返します。
   - `RefreshPending` / `refresh_pending` は旧レコードの互換読み込み用であり、現在の refresh 手順ではありません。
   - 外部 CLI の認証情報の管理・更新は CLI が行い、このストアでは更新しません。
5. **401 エラー時の自動再送なし:** モデル呼び出しは最初の HTTP リクエスト前に現在の認証情報を取得します。副作用の重複を防ぐため、401 応答を受けた際にモデルリクエストを自動リプレイすることはありません。

差し替え用 `Store` も、同じアカウントを共有するプロセス間で `WithCredential` を直列化し、ロック中に再読み込みしてアトミックに commit する必要があります。`Transaction` は callback 内だけで有効です。
