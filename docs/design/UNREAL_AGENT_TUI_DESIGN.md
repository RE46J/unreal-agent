# Unreal Agent TUI Design Specification

**Language:** English | [日本語](../ja/design/UNREAL_AGENT_TUI_DESIGN.md)

- **Target:** Interactive terminal UI (`bin/unreal-agent attach` on Linux / macOS / WSL)
- **Status:** Implemented display specification (2026-10-28), including data provenance and verification rules
- **Core Premise:** The TUI is a disposable projection of the background Host. All displayed values are derived exclusively from existing Host projections (`tui.Snapshot`, `host.View`, and `viewer.Row`; see Appendix A).

---

## 0. Conventions Used in This Document

- **UI Language Policy:** Status badges, cards, gauges, and progress phases in the TUI use Japanese labels; keyboard hints, slash commands, and tool names remain in English (see Section 7.2).
- **Mockup Symbols:**
  - `█` — Terminal hardware cursor position
  - `…` (alone at the start of a line) — Omitted lines
  - Colors and text attributes are defined in Sections 7.2–7.5.
- **Dimensions:** Written as `Columns × Rows`. Every rendered frame leaves the rightmost 1 column blank (Section 7.6). CJK characters count as 2 columns (Section 7.7).

---

## 1. Design Goals

| ID | Goal | Description |
| --- | --- | --- |
| **G1** | Conversation First | Prioritize the conversation and active work; fold background parent-child traffic into a 1-line summary and completed children into compact status beads. |
| **G2** | Instant Live Status | Keep live status (agent phase, tool execution, child agents, connection state) in fixed, predictable screen regions. |
| **G3** | Quiet Normal, Clear Alerts | Compress routine completions into counts/beads; highlight failures, stalled children, and pending confirmations with strong color bands and explicit words. |
| **G4** | Separate Canonical vs. Temporary | Visually distinguish streaming drafts (`⋮`), pending submissions, and transient notices from committed canonical history. |
| **G5** | Cards for Parallel Sub-Agents | Render active child agents as hierarchical 2-line cards sharing stable `[A]`, `[B]`, `[C]` badges across chat chips, cards, and background work summaries. |
| **G6** | Keyboard Discoverability | Typing `/` opens command autocomplete. The context-sensitive Keybar always shows what `Ctrl+C` will do right now. |
| **G7** | Color-Depth & Width Safety | Pair foreground and background colors and support truecolor, 256-color, 16-color, `NO_COLOR`, Plain, and ASCII modes without relying on color alone. |
| **G8** | Honest Projection | Display unknown values as `—` or `?` (or hide them). Never fabricate token counts or liveness; report only observed facts (e.g., `動きなし 8分`). |

---

## 2. Layout Overview & ASCII Mockups

### 2.1 Split View (Default on Terminals ≥120 Columns)

Colored bands and chips are shown in `[ ]` below. Each agent card is 2 lines high; grandchildren are indented by 2 columns per depth level.

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ [jh] [● 稼働中 7h45m] [Full Access] [Claude Opus 5.5 · max]                                                                    │
├ Chat (left) / Work Pane (right) ───────────────────────────────────────────────────────────────────────────────────────────────┤
│▎ 実装とレビューを進めて                                                │▌◆ メイン                                 7h45m        │
│                                                                        │▌    考え中 6s                            ▂▃▅▃▂▁▂▃▅▃   │
│● 作業を渡しました。                                                    │                                                       │
│  [✓ Read AGENTS.md]                                                    │▌[A] 表示の実装           gpt-6.1-sol·xhigh 1h12m       │
│  [▸ [A] 表示の実装]                                                    │▌    Bash go test ./…                     ▂▃▅▇▅▃▂▅▇▅   │
│                                                                        │  ▌[B] 独立レビュー       gpt-6.1-sol·high 58m         │
│  ┄ 裏の作業  2ターン · ツール3回 · 報告 [A][C]                         │  ▌    応答待ち                           ▁▂▃▂▁▂▃───   │
│                                                                        │▌[C] 調査                  gpt-6.1-sol·high 40m        │
│● 【要確認】失敗の扱いを確認してください。                              │▌    動きなし 8分                         ▅▃────────   │
│⠁ 考え中 6s                                                             │▌[F] Issueの整理           gpt-6.1-sol·high ✗ 失敗     │
│                                                                        │▌                                                      │
│                                                                        │終了 ✓✓✗✓✓  ほか 795                                   │
├ Attention Banner (shown only when needed, full width) ─────────────────────────────────────────────────────────────────────────┤
│ ▲ 要確認  失敗の扱いを確認してください。  ↑ 会話で確認                                                                         │
├ Keybar (when height ≥16 rows) ─────────────────────────────────────────────────────────────────────────────────────────────────┤
│  ⏎ send   / commands   ^C stop session   ^D detach                                                                             │
├ Rule ──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│› Ask, or type / for commands                                                                                                   │
├ Gauges (1 line) ───────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│文脈 ▕█████░░░░░░░▏394k (39%) · 速度 ▂▃▅▇▆▅ ~69.0 tok/s · キャッシュ ▕██████████▏98.37%                                   │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 View Modes (`/view`)

- **Split:** Side-by-side Chat and Work pane on terminals ≥120 columns. Automatically falls back to Chat below 120 columns and restores Split when widened again.
- **Chat:** Shows the conversation. On terminals ≥120 columns, if child agents exist, their cards also appear on the right; on narrower terminals, active/failed children appear in a 1-line dock above the input area.
- **Orchestration:** Uses the full main area for agent cards and completion beads (`Page Up` / `Page Down` scrolls).

---

## 3. Screen Anatomy & Visual Hierarchy

### 3.1 Screen Regions

Empty docks and attention banners collapse to 0 rows.

| Region | Rows | When Shown | Contents |
| --- | --- | --- | --- |
| **R1 Header** | 1 | Always | Session name, runtime/connection status, permission mode, model & effort bands |
| **R2 Transcript** | Variable (remaining height) | Chat / Split | Messages, tool chips, background work summary (`裏の作業`), control inputs, streaming draft |
| **R3 Pulse** | 0–1 | When an active phase exists | Bottom line of transcript: spinner + phase label + elapsed time |
| **R4 Dock** | 0–variable | When non-empty | Narrow-screen child summary or `/child` focus sheet, command/analysis sheets, connection/resync notices |
| **R5 Rule** | 1 | Always | Horizontal divider above input; shows input mode, paste status, or scroll offset |
| **R6 Composer** | 1–8 | Always | `›` prompt (`cmd` chip when typing a slash command) and editable input text |
| **R7 Keybar** | 0–1 | Height ≥16 rows | Context-sensitive keyboard shortcuts (directly above Rule) |
| **R8 Work Pane** | Right column or main area | Split / Orchestration (or Chat ≥120 cols with children) | Main card, child cards, and completed child beads |
| **R9 Attention** | 0–1 | Session failure, pending approval, or unanswered `【要確認】` | Full-width alert banner directly above Keybar |
| **R10 Gauges** | 1 | Always | Bottom status bar: Context (`文脈`), Speed (`速度`), Cache (`キャッシュ`) |

### 3.2 Conversation Gutter (2 Columns)

All transcript rows use a fixed 2-column left gutter (1 symbol column + 1 space), with message body text starting at column 3 (capped at 100 columns max width):

```text
▎ User message (▎ on every line, shaded with youBlock background up to body width)
● First line of an agent response block
⋮ Streaming draft (⋮ on every line)
⠁ 考え中 6s (Pulse line)
✗ Error message
· Control input (e.g., 停止要求)
  ✓ Read AGENTS.md (Tool chip indented under agent response)
```

### 3.3 Language Localization Boundary

| Rendered in Japanese | Kept in English |
| --- | --- |
| Header status, agent cards, completion beads, footer gauges, attention banner, tool chip statuses, background work summary (`裏の作業`), control inputs, pulse labels, status dock, `/child` focus headings | Keybar hints, Rule labels, input placeholder text, transient command notices, picker titles, `/help`, slash command names/descriptions, tool names, model names, permission mode names |

---

## 4. Color System, Typography, and Symbols

### 4.1 Role-Based Color Palette

Colors degrade gracefully across Truecolor (`COLORTERM=truecolor`/`24bit`), 256-color, 16-color, `NO_COLOR` (attributes + `[ ]` brackets only), and Plain (`TERM=dumb` or `UNREAL_AGENT_TUI_PLAIN=1`).

| Role | Foreground (Truecolor / 256 / 16) | Background (Truecolor / 256 / 16) | Usage |
| --- | --- | --- | --- |
| `band.session` | `#EEEDFE` / 255 / 97 | `#3C3489` / 55 / 45 | Session name band |
| `band.ok` | `#E1F5EE` / 194 / 30 | `#085041` / 23 / 42 | Running (`稼働中` / `裏で作業中`) |
| `band.warn` | `#FAEEDA` / 230 / 30 | `#633806` / 94 / 43 | `Full Access`, connecting/reconnecting/resyncing, attention banner |
| `band.bad` | `#FCEBEB` / 224 / 97 | `#791F1F` / 88 / 41 | Failed (`失敗`) / Disconnected (`切断`) |
| `band.neutral` | `#D3D1C7` / 252 / 97 | `#2C2C2A` / 236 / 100 | Model band, idle (`入力待ち`), stopped (`停止`), `Ask`, `Read Only` |
| `chip` | `#D3D1C7` / 252 / 97 | `#24242A` / 235 / 100 | Tool chips and `cmd` input chip |
| `chip.bad` | `#FCEBEB` / 224 / 97 | `#501313` / 52 / 41 | Failed tool chip |
| `card` / `card.alt` | Terminal default | `#1F1F25` (235) / `#232329` (236) / none | Work pane cards (alternating when compact) |
| `youBlock` | Terminal default | `#22222A` / 235 / none | User message block |
| `badge` | `#F1EFE8` / 255 / 97 | `#444441` / 238 / 100 | Child badges `[A]`, `[B]`, `[C]`… |
| `s.run` | `#5DCAA5` / 79 / 36 | — | Active symbol, card edge, activity bars |
| `s.stall` | `#FAC775` / 222 / 33 | — | Stalled, canceling, awaiting approval, unknown |
| `s.bad` | `#F09595` / 210 / 31 | — | Error/failure symbol & edge |
| `s.done` | `#97C459` / 149 / 32 | — | Completed symbol & cache bar |
| `s.main` | `#AFA9EC` / 147 / 35 | — | Main agent `◆`, agent `●`, speed sparkline |
| `s.you` | `#85B7EB` / 111 / 36 | — | User `▎` gutter and `›` prompt |
| `dim` | `#86847C` / 244 / faint | — | Metadata, elapsed times, hints |

### 4.2 Symbols (Unicode & ASCII Fallback)

ASCII mode is enabled via `UNREAL_AGENT_TUI_ASCII=1` or non-UTF-8 locales ( spinner uses `| / - \` instead of `⠁ ⠈ ⠐ ⠠ ⢀ ⡀ ⠄ ⠂`; `UNREAL_AGENT_TUI_NO_ANIMATION=1` freezes spinners to `▸` / `>`):

| Meaning | Unicode | ASCII | Color Role |
| --- | --- | --- | --- |
| Running / Canceling | `▸` | `>` | `s.run` / `s.stall` |
| Completed | `✓` | `+` | `s.done` |
| Failed | `✗` | `x` | `s.bad` |
| Canceled | `⊘` | `-` | `dim` |
| Finished (no Finish report) / Stopped | `▪` | `#` | `dim` / `band.neutral` |
| Awaiting approval / Unknown | `!` / `?` | `!` / `?` | `s.stall` |
| Main card / Agent message | `◆` / `●` | `*` | `s.main` |
| Card left border / User gutter | `▌` / `▎` | `\|` | Status color / `s.you` |
| Draft gutter / Input prompt | `⋮` / `›` | `:` / `>` | `dim` / `s.you` |
| Activity sparkline (6 levels) | `─▁▂▃▅▇` | `.:-=#` | `dim` / `s.run` |
| Speed sparkline (7 levels) | `▁▂▃▄▅▆▇` | `.:--==#` | `s.main` |
| Gauge bar | `█░` | `#-` | Value-dependent |
| Rule / Truncation ellipsis | `─` / `…` | `-` / `~` | `dim` or text color |

---

## 5. Detailed Component Specifications

### 5.1 Responsive Width & Height Breakpoints

- **<120 columns:** Split renders as Chat (no right pane; active/failed children shown in a 1-line dock).
- **120–159 columns:** Right pane width is `clamp(width * 0.3, 34, 44)` in Split (`clamp(width * 0.3, 34, 48)` in Chat with children).
- **≥160 columns:** Right pane width is `clamp(width * 0.3, 48, 60)` in Split (`48` in Chat).
- **Right margin:** 1 blank column at the right edge; 3 blank columns between left and right panes; max render width 511 columns.

### 5.2 Conversation, Wrapping, Pagination & Markdown

- **Background work folding (`┄ 裏の作業`):** Consecutive child/peer messages, end notices, and parent coordination turns fold into a single dim summary line (`┄ 裏の作業  21ターン · ツール47回 · 報告 [A][C][D]`), unless an agent message starts with `【報告】` or `【要確認】` (which always renders in full and breaks the fold).
- **Word wrapping & CJK kinsoku:** Latin words wrap at spaces (long URLs/paths break at grapheme boundaries). CJK wraps between characters while respecting line-start (`、。，．）」』】〉》！？ー`) and line-end (`（「『【〈《`) kinsoku rules.
- **Lazy history pagination:** Initial attach loads the latest 128 canonical items (`InspectBefore`). Pressing `Page Up` (or `Up` / `Home` with an empty input) near the top loads the preceding 128 items on demand without moving subscription cursors.
- **Code blocks & lightweight Markdown (canonical agent messages only):**
  - Fenced code blocks keep faint ```` ``` ```` fences, never wrap at spaces, and place `↪` in the gutter seam when a long line wraps so mouse copy-paste stays clean.
  - Headings (`#`–`###`) and `**bold**` render bold; quotes (`>`) keep a faint `>`; links `[text](url)` render as `text (url)` with faint URL. User inputs and streaming drafts are always rendered verbatim.

### 5.3 Tool Chips

Each tool call renders inline as a chip (up to 40 columns wide, wrapping across lines):
- **Coalescing:** 3 or more completed calls to the same tool within a single response collapse into `✓ Bash ×3`. Running, failed, canceled, awaiting-approval, and `SubagentStart` chips always remain individual.
- **Safe targets:** Only arguments whitelisted by `viewer.SafeToolTarget` (file path, first line of Bash command, pattern, skill name, or child badge + title) are shown.

### 5.4 Child Agent Cards, Badges & Beads

- **Deterministic badges:** All children are ordered by `(CreatedAt, ID)` and assigned `[A]`–`[Z]`, `[AA]`–`[ZZ]`, etc.
- **Card layout (2 lines):**
  - Line 1: `▌[A] Title` + right-aligned `model·effort elapsed` (or `✗ 失敗`).
  - Line 2: Latest active tool/activity (or `動きなし N分` if no activity for ≥5 minutes) + 10-cell 10-minute activity sparkline.
- **Completed beads:** Finished children collapse into `終了 ✓✓✗✓⊘  ほか N` at the bottom of the Work pane (except failed children with active descendants or failures newer than your last message, which remain as red cards).

### 5.5 Header, Pulse, Attention Banner & Gauges

- **Header bands:** `[Session ID]` `[Status + Elapsed]` `[Permission Mode]` `[Provider Model · Effort]`.
- **Pulse priority:** (1) Retrying `再試行中（N回目）` → (2) Streaming with text `生成中` → (3) Streaming without text `考え中` → (4) Waiting `応答待ち` → (5) Tool running `Bash 実行中` / `ツールN件 実行中` → (6) `WaitingForModel` `考え中` → (7) Children/background turn running `裏で作業中 子N`.
- **Attention banner priority:** (1) Session failed (`✗ ...`) → (2) Pending tool approvals (`▲ 承認待ち N件  /permissions`) → (3) Unanswered `【要確認】` message since your last input (`▲ 要確認  ...  ↑ 会話で確認`).
- **Footer gauges:** Context tokens + 12-cell bar (`<70%` `s.main`, `70–89%` `s.stall`, `≥90%` `s.bad`), Speed sparkline (last 8 responses) + `tok/s`, and Cache hit rate + 10-cell bar.

---

## 6. Acceptance Criteria Summary (AC-01 to AC-50)

The implementation is verified against 50 acceptance criteria covering:
- **Layout & Boundaries (AC-01–AC-05):** Frame bounds, 1-column right margin, 0-line collapse for empty docks, bottom follow vs. scrollback, and lossless terminal resize.
- **Header & Gauges (AC-06–AC-08):** Truncation priority, explicit error words, and `—`/`?` for unknown metrics.
- **Conversation & Drafts (AC-09–AC-17):** 2-column gutter alignment, kinsoku wrapping, copy-safe code blocks (`↪`), 64 KiB entry truncation notice, verbatim user text, and atomic draft-to-canonical replacement.
- **Tools & Children (AC-18–AC-29):** 40-column tool chips, `×N` folding for ≥3 completions, hidden internal op IDs, `[A]` badge consistency, 2-line cards, completion beads, and `/child` focus sheet.
- **Input, Commands, Errors & Accessibility (AC-30–AC-50):** Hardware cursor positioning, paste safety, `/` autocomplete, context-aware `Ctrl+C` hints, `NO_COLOR`/Plain/ASCII parity, control-character sanitization, and flicker-free streaming redraws.

### Visual Scene Regression Tests

To regenerate the golden ANSI visual frames (only when intentionally updating visual scenes):

```bash
UNREAL_AGENT_WRITE_VISUAL=1 go test ./cmd/internal/tui -run '^(TestVisualWorkPaneFrames|TestBackgroundActivitySpinsInHeaderChatAndMainCard)$' -count=1
```

Golden files are stored in `cmd/internal/tui/testdata/visual/*.ans`.

---

## Appendix A. Data Provenance Matrix

| UI Element | Source Field | Notes |
| --- | --- | --- |
| Session ID, connection, runtime status | `Snapshot.ID`, `Connected`, `Status`, `OfflineSince`, `Running`, `Failure` | Connection state overrides runtime status in header when offline/resyncing |
| Model, effort, context window | `Snapshot.Selection` (`RuntimeSelection`) | `ContextWindow == 0` renders as unknown (no bar) |
| Session elapsed time | Main `viewer.Row.Elapsed` | Cumulative token usage is shown in `/analyze` |
| Transcript entries | Converted from `host.HistoryItem` | Canonical history is never mutated by rendering |
| Tool chip state, reason, elapsed | Receipt + latest Operation + `viewer.OperationRow.Elapsed` | Matched by `CallID` / `OperationID` |
| Tool target text | `viewer.SafeToolTarget` | Whitelisted display fields only |
| Child spawn & status | `OperationIntent.OperationID` ↔ `Row.ParentOperationID`, `Finish`, `NeedsResync` | Unobserved children do not show activity or usage |
| Child activity sparkline | `Row.RecentActivity`, `LastActivityAt` | Ring buffer of last 256 canonical item timestamps (last 10 minutes) |
| Streaming draft & pulse | `Snapshot.Progress`, `WaitingForModel`, `LatestAt` | Ephemeral; cleared on canonical commit or disconnect |
| Context & speed gauges | `Progress.Statistics`, `LastGeneration.ContextInput()`, `OutputTPS()`, `SpeedHistory` | Claude context uses latest `message_start` (`input + cache_read + cache_creation`) |
| Cache gauge | `Snapshot.CacheLatest` | Latest turn cache ratio |
| Pending approvals & `【要確認】` | `Snapshot.Permissions.Pending`, `Snapshot.Entries` | Drives the full-width Attention banner |
| Project Instructions metadata | `Row.ProjectInstructions` | Source path, byte length, and SHA-256 digest only |
