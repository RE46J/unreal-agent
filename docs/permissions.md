# Permissions

**Language:** English | [日本語](ja/permissions.md)

Use `/permissions` to control what tools and shell commands the agent can run, and to review any pending approval requests.

---

## Permission Modes

New workspaces start in **Read Only** mode by default. Whenever you change the mode, Unreal saves your choice for that workspace and restores it automatically across Host restarts and Session resumes.

| Mode | What Runs Automatically | What Requires Approval / Is Blocked |
| --- | --- | --- |
| **Read Only** | File reads (`read`, `grep`, `glob`, `ast_grep`), `ViewImage`, `SkillUse`, and recognized read-only Bash commands | File writes/edits and non-read-only Bash commands are **denied** |
| **Ask for Approval** | All inspection tools and recognized read-only Bash commands | File writes/edits and privileged operations **pause and wait for your Allow/Deny decision** |
| **Full Access** | All operations permitted by the Host's maximum policy run without prompts | Nothing prompts (still bounded by workspace path rules and operator maximum policy) |

---

## Approving or Denying Operations

When in **Ask for Approval** mode:

1. If an agent requests a file edit or privileged command, an attention banner appears above the input bar (or open `/permissions` directly).
2. Review the displayed tool name, command, or file target.
3. Select **Allow** (focused by default) or **Deny**.

### Approval Lifecycle Rules

- **Single-operation scope:** Selecting **Allow** grants permission for that specific operation only. Switching to **Full Access** is a separate explicit choice.
- **No timeout:** Pending human approval prompts wait indefinitely until you decide.
- **Cancellation on stop or mode change:** Stopping the Session or changing the permission mode in either direction cancels pending approval prompts and revokes active execution authority. While a Session is stopped, its effective mode is `Read Only` until you `/resume` (which restores your saved mode).
- **Detaching is safe:** Pressing `Ctrl+D` to detach from the terminal does not reset your permission mode.

---

## How Permissions Apply to Claude and Sub-Agents

- **Claude Code:** Built-in Claude Code tool calls pass through Unreal's permission hook. If Claude Code refuses to run a Bash command on its own, Unreal handles it through Unreal's Bash executor under your selected permission mode (`Read Only`, `Ask for Approval`, or `Full Access`).
- **Child agents (Sub-agents):** Child agents inherit the parent Session's permission mode (up to the configured maximum policy). Rather than interrupting you with separate approval popups for every child, child requests are routed to the parent agent, which only asks you when your confirmation is needed.

---

## Important Security Boundaries

- **Not an OS sandbox:** Permission modes enforce authorization inside Unreal's harness, **not** at the operating-system kernel level. Any allowed shell command, language server, or external binary runs with your OS user's environment and access. Use a Docker Sandbox, container, or VM when you require OS-level isolation.
- **Workspace path boundary for Native/AST tools:** Even in **Full Access**, Unreal's native file tools (`read`, `write`, `edit`, `grep`, `glob`, `ast_grep`, `ast_edit`) are strictly confined to the Session's workspace root directory. They cannot read or modify files outside the workspace. (Bash commands are not restricted to the workspace path.)

See [Tools and File Mutation](tools.md) for file locking and revision guarantees, and [Architecture](architecture.md) for policy composition.
