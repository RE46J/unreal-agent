# Providers

Claude and Codex share the same installation and `/model` menu. Daily use relies
on subscription logins. There is no automatic provider or API-key fallback.

## Select a model

1. Enter `/model` and select a provider.
2. Select one of its advertised models, then an offered reasoning effort.
3. After logging in or changing the external CLI, run `/model refresh` to
   rediscover providers and reload model metadata.

Unavailable providers remain visible with a reason and preparation instructions.
Selecting an unavailable provider also retries discovery. A fresh installation
starts with Claude when both providers are available, or the sole available
provider. An explicit initial provider selection in the environment takes
precedence; subsequent Session selections are recorded by Unreal.

Model metadata is not a successful inference test. A model listed by a CLI or
cache may still be unavailable to the account.

## Claude through Claude Code

1. Install the official `claude` CLI in the environment running Unreal.
2. Run `claude` in another Bash shell and enter `/login` inside Claude Code.
   Choose your Claude subscription login.
3. Return to Unreal and run `/model refresh`.

The provider ID is `claude`. Claude Code owns its login and conversation. Unreal
starts the CLI, passes conversation input, applies tool approvals, and records
reported tool outcomes as `claude_code` Operations. Claude Code executes its own
built-in tools. If it declines a Bash command itself, Unreal handles that command
through its Bash executor under the same selected permissions and supplies the
result to Claude. Organization-wide settings are not changed.

The model catalog comes from the Claude Code initialization response. Unreal
does not send a separate `list_models` request. Project Instructions and
Claude's conversation compaction are described in [Context](context.md).

## Codex subscription route

1. Install the official Codex CLI and run `codex login` in Bash.
2. Open Codex once if Unreal reports a missing model catalog, so the CLI can
   populate its local metadata.
3. Start Unreal, or run `/model refresh` in an existing Session.

The provider ID is `openai-codex`. Unreal sends model requests through its own
Codex adapter and uses Unreal tools. The external CLI owns OAuth login and
refresh; Unreal does not import a token into its own key store.

Auth-file selection follows an explicit native `--codex-auth-file`, then
`OPENAI_CODEX_AUTH_FILE`, then `$CODEX_HOME/auth.json`, then
`$HOME/.codex/auth.json`. The native OAuth file must be a regular private file.
The ordinary external-token route reads `models_cache.json` beside that file;
it does not start the CLI to enumerate models.

## Docker Sandboxes Codex OAuth

This is an authentication route for an existing Docker Sandbox. Unreal uses
the host-managed subscription through Docker's credential forward proxy.

1. Prepare the Docker host's OpenAI OAuth login using the Sandbox tooling. For
   the Codex kit, the host command is `sbx secret set openai --oauth`; see
   [Docker's Codex setup](https://docs.docker.com/ai/sandboxes/agents/codex/).
2. Enter the Sandbox with its Codex kit configured for OAuth.
3. Install the common Unreal kit as in [Install](install.md), adding
   `--docker-sandbox-oauth` to `bash development.sh` to select this route
   explicitly. Automatic Codex routing can also recognize a valid kit setup.
4. Start Unreal and choose Codex in `/model`.

The resolver validates `SBX_CRED_OPENAI_MODE=oauth`, the kit's nonsecret auth
marker, its selected `sandboxd` TOML provider, and the expected Codex endpoint.
A kit setting named `forced_login_method = "api"` is part of that proxy routing;
it does not make this subscription route an API-key login.

The route needs `HTTPS_PROXY` (or `https_proxy`) to identify Docker's forward
proxy. Unreal accepts `http://host.docker.internal:3128` or
`http://gateway.docker.internal:3128`; conflicting uppercase/lowercase values are
rejected. The forward proxy substitutes the host credential. A transparent
network path alone cannot do this. Unreal does not read the host OAuth token,
copy credentials, or install a certificate. A routing failure does not switch to
another auth source or direct traffic.

For model metadata, this route first reads `models_cache.json`. If unavailable,
it runs the Codex CLI's metadata-only `app-server` initialization and `model/list`
exchange. This does not start a coding turn or execute tools. The last successful
CLI list is saved under the user cache directory at
`unreal-agent/codex-cli-catalog.json`. It can be reused for 24 hours and retained
when a later CLI lookup fails. `/model refresh` requests a fresh lookup when the
local model cache does not already supply the catalog. A slow CLI startup can
therefore leave the previous successful list visible. Authentication is checked
separately from catalog availability.

## Delegated GPT work

The normal Host supplies built-in templates when Codex is registered, permission
modes and their maximum policy are configured, and no explicit `Subagents`
configuration replaces the defaults. These templates can also be used in
existing Sessions. Their Codex model/effort selections are pinned independently
of the parent's current model.

| Role/template | Model and effort | Work |
| --- | --- | --- |
| `lead` | `gpt-6.1-sol`, `max` | Implementation coordination; can start its own workers |
| `worker-low`, `worker-medium`, `worker-high`, `worker-xhigh`, `worker-max` | `gpt-6.1-sol`, the named effort | Implementation and assigned test runs |
| `collector` | `gpt-6-luna`, `medium` | Searches, file/log value extraction, URL/API results, CI status, read-only summaries |

Ask Claude to delegate a task or give implementation work to the lead. Claude
uses the installed `unreal-agent subagent` Bash interface. It hands off work,
ends its turn, and resumes on the child's notice. The parent decides child
approvals. Child modes follow the parent's selected mode within the configured
maximum policy.

Children inherit the Host environment except Codex inline token/account
variables removed by the launcher. This is process inheritance, not an isolation
guarantee. See [Permissions](permissions.md).
