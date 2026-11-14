package claudeagent

import "strings"

// claudeSubagentModelEnv names the model Claude Code's own Agent sub-agents
// run on; Claude Code counts "inherit" as unset.
const claudeSubagentModelEnv = "CLAUDE_CODE_SUBAGENT_MODEL"

// agentModelEnv names the model chosen with /model-sub for Claude Code's
// Agent sub-agents. Without a choice, or while Unreal's own children run, the
// environment is left exactly as it was.
func agentModelEnv(env []string, s SubagentCLI) []string {
	if s.Binary != "" || s.AgentModel == nil {
		return env
	}
	if model := s.AgentModel(); model != "" {
		return append(env, claudeSubagentModelEnv+"="+model)
	}
	return env
}

// claudeAgentModelRule is upstream's ask for Claude's own haiku/sonnet choice;
// Claude's per-invocation Agent model wins over the environment.
const claudeAgentModelRule = `to the Agent tool with model set to "haiku" for the simplest and "sonnet" for moderate ones, and review`

// claudeDelegationPromptFor is the delegation prompt for the model it runs:
// with a model chosen with /model-sub the Agent runs that model, so the prompt
// names it; otherwise upstream's text stays byte for byte.
func claudeDelegationPromptFor(s SubagentCLI) string {
	if s.AgentModel == nil {
		return claudeDelegationPrompt
	}
	model := s.AgentModel()
	if model == "" {
		return claudeDelegationPrompt
	}
	return strings.Replace(claudeDelegationPrompt, claudeAgentModelRule, "to the Agent tool, which runs them on "+model+", and review", 1)
}
