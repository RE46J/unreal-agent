# Usage

This guide describes the **installed common-kit wrapper**. Source-built launcher
options are documented in [Architecture](architecture.md#native-launcher).

## Open or find a Session

1. Open Bash in your project and run `unreal development`.
2. If that name does not exist anywhere, confirm creation with `y`. Its workspace
   is the current Git worktree root, or the current directory outside Git.
3. On later visits, run the same command from any directory to reconnect.

| Command | Result |
| --- | --- |
| `unreal` | List existing main Sessions, newest first, and choose by number |
| `unreal development` | Open the Session named `development`, or offer to create it |
| `unreal --where development` | Print that Session's workspace path |
| `ucd development` | Change this Bash shell to that Session's workspace |
| `unreal --workspace-info` | Show the current workspace's config, state, and socket paths |
| `unreal --readme` | Show the kit's packaged README |

An unnamed launch never creates a Session. If the same name exists in multiple
workspaces, the current workspace's copy wins; otherwise choose a numbered
candidate. `--where` uses the same lookup and does not start a Host. `ucd` is a
Bash function installed through `~/.config/unreal-agent-common/path.sh`.

Explicit `--instance` and `--config` options select an operator destination in
the current workspace. These bypass global Session lookup. They are mainly for
separate development configurations; ordinary use needs only a name.

## Work in the terminal

1. Type `/` to search the command menu. Use Up/Down and Tab for selection and
   completion, then Enter to open the command's choices.
2. Use `/model` for provider, model, and effort; use `/permissions` for approvals.
3. Type your task and press Enter.
4. Use Page Up/Page Down to scroll and `/help` to show the available keys.

Split is the initial view. It shows Chat alongside Orchestration when the terminal
is at least 120 columns wide. Narrower terminals display Chat. `/view` lets you
choose Chat, Orchestration, or Split. Orchestration shows agents and Unreal-owned
Operations. Child work is visible there and in Split; agent control commands are
not listed in the normal command menu.

The footer shows current context, generation speed, and cache use. Unknown
measurements appear as a dash. A speed prefixed with `~` uses client-clock timing;
Claude uses this measurement. Operation and approval waits are not a generation
rate timer. See [Troubleshooting](troubleshooting.md#read-analysis-and-measurements).

## Input and pasted text

| Key or action | Result |
| --- | --- |
| Enter | Send the message |
| Ctrl+Enter, Shift+Enter, Alt+Enter, or Ctrl+J | Insert a newline when the terminal sends the corresponding key sequence |
| Backslash at the end of a line, then Enter | Insert a newline without special terminal key support |
| Ctrl+A | Select all composer text |
| Type or paste with text selected | Replace the selection |
| Backspace/Delete with text selected | Clear the selection |
| Esc or an arrow key | Deselect text |
| Home | Move to the start of the composer |

Long multiline pastes appear as `[Pasted text #1 +N lines]`. The full text is sent
when you submit. `@` file search helps insert workspace paths; an ignore-file or
scan error is shown rather than silently returning an incomplete index.

For Windows file transfer, the optional
[Sandbox drop helper](../scripts/windows/sandbox-drop.cmd) can be copied into the
Windows SendTo folder and renamed to the target Sandbox name, such as
`development.cmd`. Dropped files go to `/home/agent/drop/`; their Sandbox paths
are copied to the clipboard for pasting into Unreal.

## Copy and export

1. Run `/copy` to open the message picker. It includes user and agent messages.
2. Move with Up/Down. Use Space to select several messages, then Enter to copy.
   Choose the all-messages entry to copy the whole public conversation.
3. Use `/copy all` for the full public conversation as Markdown, or `/copy last`
   for the latest public agent response.

Copy uses the OSC 52 terminal escape sequence. Your terminal, and any remote-shell
or terminal-multiplexer path, must support clipboard writes and allow them. A
successful send to the terminal is not proof that the system clipboard changed.
If copying has no effect, check terminal clipboard settings or export a file.

1. Run `/export` and choose Latest response, Full conversation, or Choose a response.
2. Alternatively, use `/export last`, `/export all`, or `/export pick` directly.
3. Open the Markdown file at the path printed by Unreal.

The default export directory is
`${XDG_STATE_HOME:-$HOME/.local/state}/unreal-agent/exports`. Conversation export
uses canonical public history, including content scrolled out of the TUI. Copy
and export exclude private reasoning, internal compaction, and private background
coordination. A full conversation export includes recorded statistics.

## Stop, detach, and resume

`/stop` opens four choices. **Main agent** is selected initially.

| Choice | Command | Effect |
| --- | --- | --- |
| Sub-agents | `/stop sub` | Cancel every child; the main agent continues |
| Main agent | `/stop main` | Interrupt the main agent's current work; children continue |
| Main agent and sub-agents | `/stop main+sub` | Interrupt the main agent and cancel children |
| Session | `/stop session` | Stop the Session; `/resume` starts it again |

Ctrl+D or `/detach` leaves the terminal while the Session runs. Ctrl+C cancels a
pending terminal wait; otherwise it stops the Session. After an uncertain input
submission, `/retry` resends the same request identity without inventing a new
message.

Run `/resume` to resume a stopped Session, or reopen it with `unreal development`.
The saved permission mode is restored. An unavailable provider can be changed
with `/model` even while stopped, followed by `/resume`. Changing providers is
always explicit.

## Background work and children

Claude can hand implementation or research to GPT children. Child notices can
wake a background turn in the parent without a new user message. Routine
coordination is folded into a compact work row in Chat. Messages marked
`【報告】` (report) or `【要確認】` (needs your reply) remain visible in full. An
unanswered confirmation stays visible until you reply.

Use Split or Orchestration to follow child activity and Operations. The parent
handles child approval requests under the selected permission mode. See
[Providers](providers.md#delegated-gpt-work) and [Permissions](permissions.md).
