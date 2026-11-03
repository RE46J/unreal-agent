# Permissions

Use `/permissions` to choose the Session's mode and review pending operations.
The normal configuration starts in Read Only. Your last selected mode is saved
for the workspace and restored after Host restart and Session resume.

| Mode | Behavior |
| --- | --- |
| Read Only | Inspection tools and recognized read-only Bash commands run; privileged work is denied |
| Ask for Approval | Inspection runs; edits and other privileged operations wait for a decision |
| Full Access | Operations within the configured maximum policy run without individual approval |

Recognized read-only Bash commands can run in every mode. This does not classify
all shell commands as safe. Native reads, search, images, and skill loading are
inspection tools. Native/AST writes and other privileged operations use the
selected mode. An operator's immutable maximum policy still applies.

## Review a request

1. Select Ask for Approval in `/permissions`.
2. When an operation needs approval, read its displayed tool, command, or scope.
3. Choose Allow or Deny. Allow is the initially focused option. A grant applies
   to that operation; Full Access is a separate mode choice.

Human approval requests have no expiry timer. Stopping the Session or changing
its mode can cancel pending decisions and revoke active operation permissions.
A saved mode is restored on resume; an old pending approval is not a reusable
grant. While stopped, the effective mode is Read Only and the saved selection
remains available for restoration. Detaching the terminal does not change the
selected mode.

## Claude and delegated work

Claude Code built-in tool requests pass through Unreal's approval hook. If Claude
Code declines a Bash command itself, Unreal treats it as a normal Bash operation:
recognized read-only commands can run in Read Only, privileged commands need
approval in Ask, and Full Access permits them within the maximum policy.

Children follow the parent Session's mode. The parent agent answers a child's
pending operation request, rather than opening a separate human approval flow
for each child. The parent can ask you when a decision needs your input.

## Scope

These modes control harness authorization; they are not an OS sandbox. Shell
processes, editors, and external programs may access the environment allowed by
the operating system. Use your chosen container or VM if you need OS isolation.
Unreal does not create one as part of normal startup.

Native file tools retain their workspace root even in Full Access. A native path
outside that root is invalid; changing mode does not move it. Bash has a
different filesystem boundary. See [Tools](tools.md) for mutation guarantees and
[Architecture](architecture.md) for policy composition.
