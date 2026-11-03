# ケーパビリティと権限の強制 (`harness/permission`)

**言語:** [English](../../../../harness/permission/README.md) | 日本語

`harness/permission` パッケージは、ツール呼び出し、ファイルシステムルート、サブプロセス起動、および HTTP 通信に対する不変（immutable）の権限ポリシーを強制します。

---

## 1. ポリシーの構築とコンテキストへの付与

Host は実行所有権のライフサイクルごとに不変の `permission.Policy` を構築し、コンテキストへ付与します。

```go
policy, err := permission.New(permission.Config{
    Tools:          []string{"ViewImage", "SkillUse"},
    ReadRoots:      []string{workspace, skillsDirectory},
    NetworkOrigins: []string{"https://api.openai.com"},
})
// エラー処理後、全 Executor の停止後に policy.Close() を呼ぶ
ctx = permission.WithPolicy(ctx, policy)
manager := operation.NewLocalOperationManagerWithPolicy(ctx, policy)
```

- **フェイルクローズ原則:** 空の `Config` や明示的な `nil` ポリシーはすべてのアクセスを拒否します。`permission.Unrestricted()` は明示的な非制限許可です。policy を付与していない低レベル context は互換性のため unrestricted であり、明示的な `nil` とは異なります。新しい Host は policy を明示的に付与する必要があります。
- **権限の永続化なし:** ツール名や Operation が履歴に保存されていても、それが将来の実行許可になることはありません。再開（resume）時には常に現在の設定から新しいポリシーを構築します。

---

## 2. 各実行境界での検査

1. **Coordinator と Operation Manager:** Translator を呼ぶ前、および Operation の開始・再開前に独立して権限を検査します。拒否された場合はシリアライズ可能な `Denial` を持つ失敗 Operation として記録されます。
2. **ファイルシステム境界 (`ReadFile` / `Create` / `OpenParent`):**
   - 読み取りルートと書き込みルートは独立しており、読み書き両方（`O_RDWR`）には両方の許可が必要です。
   - ディレクトリルートは `os.Root` 記述子によって固定（pin）され、シンボリックリンクの競合や `..` によるルート外脱出を防止します。
   - `CheckPath` は事前確認用であり、ファイル変更を行う Executor はロック保持中に固定記述子を用いて再度認可を検査します。
3. **サブプロセス境界 (`StartProcess`):**
   - 出力キャプチャの作成やプロセス起動の前に検査します。
   - ハーネス自体は OS カーネルレベルのサンドボックスを持たないため、任意のシェル・LSP・DAP・サブエージェントを起動するには、ファイル・ネットワーク・プロセスの明示的な非制限（unrestricted）許可が必要です（制限付き設定でのプロセス起動は `unsupported` で安全に拒否されます）。
4. **ネットワーク境界 (`RemoteClient` / `Policy.RoundTripper`):**
   - すべての HTTP 通信（リダイレクトを含む）に対して、完全一致のオリジン（`scheme://host[:port]`（HTTP / HTTPS））を検査します（ワイルドカードや URL 内のユーザー情報は拒否されます）。
   - 拒否記録には URL、パス、コマンド、ヘッダー、認証情報を含めません。

---

## 3. 子セッションへの権限継承と積集合 (`WithPolicy` / `Intersect`)

`WithPolicy` や `Policy.Intersect` は常に既存ポリシーとの**共通部分（積集合）**をとります。子セッションが `Unrestricted` を要求しても親の権限を超えることは決してありません。

## 4. 信頼境界と保証の範囲

- 組み込む Go コード、Executor、自前の HTTP transport は信頼対象です。Host 内で動く悪意あるコードから policy が保護するわけではありません。
- root 内の hard link は、同じ inode の別名が root 外にあってもアクセスできます。共通 Mutation サービスは別途、複数リンクされた更新対象を拒否します。
- HTTP origin の許可リストは URL の origin を制限し、DNS 解決先や IP 範囲は制限しません。自前の transport は意図した経路と安全性を維持する責任があります。base transport が nil の制限付き policy では ambient proxy を無効にします。
- 現在モデルに見せない、または拒否中のツールも、履歴の receipt を読めるよう結果・履歴 codec を保持します（`Registry.ResolveHistory`）。
