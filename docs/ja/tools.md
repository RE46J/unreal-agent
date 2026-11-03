# ツールとファイル変更 (Tools and File Mutation)

**言語:** [English](../tools.md) | 日本語

Unreal の[固定ツールレジストリ](../../harness/tool/registry.go)は、ネイティブファイル操作ツール、Tree-sitter AST ツール、`Bash`、`ViewImage`、`SkillUse`、および Claude Code の観測結果（`claude_code`）で構成されています。

- **Codex (`openai-codex`) およびサブエージェント:** 本書で説明する Unreal のネイティブ / AST / Bash 実行パイプラインを使用します。
- **Claude (`claude`):** Claude Code の内蔵ツールを使用し（結果は `claude_code` として記録）、Claude Code が辞退した Bash コマンドのみ Unreal の Bash 実行器で処理します。

---

## 1. ネイティブファイルツール

ネイティブファイルツールは、`Full Access` モードであっても**常にセッションのワークスペースルート配下に限定**されます。パスはワークスペースからの相対パスで指定します。

| ツール | 引数と動作 |
| --- | --- |
| `read` | `path`（任意で `start_line`, `lines`, `max_bytes`）。ファイルを読み取り、一部行だけを読んだ場合でもファイル全体の SHA-256 `revision` を返します。 |
| `write` | `path`, `content`, `expected`。ファイル全体を書き込みます。新規作成時は `{"exists":false}`、上書き時は `read` で取得した `revision` を指定します。 |
| `edit` | `path`, `old_text`, `new_text`, `expected`（複数箇所を置換する場合は `replace_all:true`）。`expected` リビジョンが一致する場合のみ文字列を置換します。 |
| `grep` | `path`, `pattern`（Go 正規表現。任意で `glob`, `limit`, `max_bytes`）。ファイル内容を決定論的な順序で検索します。 |
| `glob` | `path`（任意で `glob`, `limit`, `max_bytes`）。幅優先・ソート済み順でファイルパスを列挙します。 |

### リビジョン検証と出力上限

- **リビジョンによる競合防止 (`revision`):** `read` はファイル全体のスナップショットに対する `{"exists":true,"digest":"<sha256>"}` を返します。`write` や `edit` はこのリビジョンを `expected` として受け取り、読み取り後にファイルが変更されていた場合は上書きせずに競合エラー（`revision_conflict`）を返します。
- **引数の検証:** `version` と `action` はツール・Executor が設定するためモデル引数では指定できず、未知フィールドも拒否されます。新規ファイルの作成には既存の親ディレクトリが必要です。
- **出力サイズ上限:** `max_bytes` の既定値は `16 KiB`、検索 `limit` の既定値は `200` 件です。上限で切り詰められた場合は結果の `truncated` フラグが `true` になります（出力上限は返却本文の制限であり、入力ファイルのサイズや revision の計算範囲の制限ではありません）。
- **Glob 構文・シンボリックリンク・バイナリ:** `grep` / `glob` はシンボリックリンクをスキップします。glob はパス要素内の `*`、`?`、文字クラスに対応し、要素全体の `**` は0個以上のディレクトリ要素に一致します。バイナリファイルを `read` した場合は本文バイト列を返さずリビジョン情報のみを返します。
- **型付きの失敗理由 (`reason`):** 失敗時には `workspace_boundary`、`parent_missing`、`invalid_revision`、`revision_conflict` などの型付き `reason` が返ります（`code=invalid` だけでは Permission 拒否とは判断できません）。

実装の詳細は [`harness/tool/native/native.go`](../../harness/tool/native/native.go) および [`harness/native/native.go`](../../harness/native/native.go) を参照してください。

---

## 2. AST 検索と構造的編集 (`ast_grep` / `ast_edit`)

`ast_grep` と `ast_edit` は **Tree-sitter** を用いて **Go**、**JavaScript**、**Python** の構文木を検索・書き換えします。

### クエリの書き方

リクエストには `language`、`query`、および対象ファイルの `targets`（各要素に `path`）を指定します。各マッチには置換・抽出対象となる `@match` キャプチャが1つ必要です。

```text
((call_expression
  function: (identifier) @callee
  arguments: (arguments) @args) @match
 (#eq? @callee "old"))
```

例えば上記の JavaScript クエリに対して `replacement` に `updated${args}` を指定すると、`old(1)` が `updated(1)` に書き換わります（文字通りの `$` を入れる場合は `$$` を使います）。置換後のテキストは組み立てるだけで再パースしないため、置換後の構文の正しさは保証しません。

- **検証とタイムアウト:** 曖昧な capture や重複した match 範囲は拒否されます。parse と query の実行予算はそれぞれ 500 ms で、timeout 時の部分結果は編集の根拠として受け入れません。
- **プレビュー出力上限:** プレビュー出力の既定値は `16 KiB` です（表示が省略されても match 件数や検証対象の revision は減りません）。

### 安全な2段階編集手順 (`ast_edit`)

1. **プレビュー (`apply: false`、既定値):** `replacement` を指定して `ast_edit` を実行します。ファイルは書き換えられず、各ファイルのリビジョン、マッチ件数、バイト範囲（`start_byte` / `end_byte`）、差分プレビューが返ります。
2. **適用 (`apply: true`):** プレビューで返された `expected_matches` と、すべての対象ファイルの `expected` リビジョン（マッチしなかったファイルを含む）を指定して `apply: true` で実行します。
3. **動作確認:** コンパイラやテストを実行し、型やプログラムの振る舞いを確認します。

> **ビルド要件:** AST ツールには `CGO_ENABLED=1` と C コンパイラが必要です（配布スナップショットでは常に有効です。`CGO_ENABLED=0` でビルドされた場合は `unsupported_backend_cgo_required` を返します）。
>
> 実装の詳細は [`harness/ast/ast.go`](../../harness/ast/ast.go) および [`harness/ast/engine.go`](../../harness/ast/engine.go) を参照してください。

---

## 3. 共通 Mutation サービスの安全性保証

ネイティブファイル編集、AST 編集、および LSP のワークスペース編集は、すべて共通の [Mutation サービス](../../harness/mutation/mutation.go)を経由して行われます。

### 保証される動作（Linux / macOS）

- **正規順ロック:** 相対パスの別名を正規化し、複数ファイルを変更する場合は正規化された順序でアドバイザリロックを取得します（ロックファイルは対象ファイルとは別のプライベートな mutation ディレクトリに置かれます）。
- **シンボリックリンクとハードリンクの拒否:** 固定されたディレクトリ記述子（`os.Root` / `O_NOFOLLOW`）を使用し、対象パス内のシンボリックリンクや複数ハードリンクされたファイルの置換を拒否します。
- **書き込み前の全件プリフライト検証:** 最初の1ファイルを書き換える前に、ロックを保持した状態で全対象ファイルのリビジョンと権限を検証します。
- **ファイル単位のアトミック置換:** 同一ディレクトリ内の一時ファイルへの書き込み・`fsync`・アトミックな `rename` によって各ファイルを置換し、ファイルごとの結果を返します。
- **クラッシュ時の重複実行防止:** Coordinator は実行前に不変の Operation 計画を永続化し、Executor は `Session ID + Operation ID + 計画ハッシュ` に紐づく実行意図・結果の証跡（proof）を記録します。クラッシュからの復旧時にはこの証跡を確認し、結果が不明な場合は勝手に再実行せず `indeterminate`（結果不明）として報告します。

### 保証の範囲外（注意点）

- **複数ファイルのロールバックトランザクションではありません:** 各ファイルの置換はアトミックですが、複数ファイル編集の途中で失敗やキャンセルが発生した場合、すでに書き換わったファイルはロールバックされません（ファイルごとに `applied` / `failed` / `not_applied` / `indeterminate` が報告されます）。
- **外部プロセスによる直接書き込み:** `Bash` コマンド、外部エディタ、デバッガ対象プロセスによるファイル書き込みはこのロック機構の外で行われます。
- **協調する Writer の要件:** 協調するすべての writer は、同じ canonical workspace root と共有の永続 Mutation state directory を使う必要があります。Operation の回復があり得る期間は receipt と proof を保持し、同じ Operation identity を異なる計画に再利用できません。
- **対象ファイルシステム:** 保証対象は協調するローカルファイルシステムであり、remote/distributed FS、mount 変更、外部の directory rename は保証の範囲外です。
