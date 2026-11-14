package claudeagent

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/permission"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

// Claude Code's own Agent sub-agents show their tool calls in the progress
// stream, indented under the Agent call, while their text and their calls stay
// out of the Session.
func TestSubagentCallsShowInTheProgressStream(t *testing.T) {
	for _, tc := range []struct{ name, in, chosen string }{
		{"the call's own model", "haiku", "sonnet"},
		{"the model chosen with /model-sub", "", "opus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, "agent_progress", "CLAUDEAGENT_AGENT_MODEL="+tc.in)
			h.client.config.Subagents = SubagentCLI{AgentModel: func() string { return tc.chosen }}
			ctx, _, _ := withMode(t, permission.FullAccess)
			response, err := h.respond(ctx)
			if err != nil {
				t.Fatal(err)
			}
			model := tc.in
			if model == "" {
				model = tc.chosen
			}
			progress := h.progress.String()
			for _, want := range []string{
				"\n› Claude Code Agent · " + model + ": Review the parser\n",
				"\n" + subagentLinePrefix + "Claude Code Read: /tmp/parser.go\n",
			} {
				if !strings.Contains(progress, want) {
					t.Fatalf("progress lacks %q: %q", want, progress)
				}
			}
			if strings.Contains(progress, "Look at the parser and report the bugs you find.") || strings.Contains(progress, "I will read the parser now.") {
				t.Fatalf("progress shows the Agent input or the sub-agent's text: %q", progress)
			}
			if len(h.responses) != 1 {
				t.Fatalf("rounds = %d, want 1", len(h.responses))
			}
			recorded := h.responses[0].Output
			if len(recorded) != 1 {
				t.Fatalf("round = %+v, want the Agent call alone", recorded)
			}
			if call := recorded[0].Data.(llm.ToolCall); call.CallID != "toolu_agent" || call.Name != tool.ClaudeCodeName {
				t.Fatalf("recorded call = %+v", call)
			}
			if len(response.Output) != 1 || response.Output[0].Data.(llm.Message).Text != "Done." {
				t.Fatalf("final response = %+v", response.Output)
			}
			// The sub-agent's call and text never reach Unreal's history.
			for _, items := range [][]llm.Item{recorded, response.Output} {
				if records(items, "toolu_read") || records(items, "I will read the parser now.") || records(items, "/tmp/parser.go") {
					t.Fatalf("the sub-agent was recorded in the Session: %+v", items)
				}
			}
		})
	}
}

// records reports whether a recorded item carries the text or call id.
func records(items []llm.Item, needle string) bool {
	for _, item := range items {
		switch data := item.Data.(type) {
		case llm.Message:
			if strings.Contains(data.Text, needle) {
				return true
			}
		case llm.ToolCall:
			if data.CallID == needle || strings.Contains(data.Arguments, needle) {
				return true
			}
		}
	}
	return false
}

// The Agent call's subject names the model and the task, never the JSON input.
func TestAgentCallSubjectNamesTheModelAndTheTask(t *testing.T) {
	chosen := func() string { return "sonnet" }
	for _, tc := range []struct {
		name, input string
		chosen      func() string
		want        string
	}{
		{"the call's own model wins", `{"model":"haiku","description":"Review the parser"}`, chosen, "Claude Code Agent · haiku: Review the parser"},
		{"the chosen model when the call names none", `{"description":"Review the parser"}`, chosen, "Claude Code Agent · sonnet: Review the parser"},
		{"no model leaves no model part", `{"description":"Review the parser"}`, func() string { return "" }, "Claude Code Agent: Review the parser"},
		{"no choice at all", `{"description":"Review the parser"}`, nil, "Claude Code Agent: Review the parser"},
		{"no description", `{"model":"haiku","prompt":"the whole task"}`, nil, "Claude Code Agent · haiku"},
		{"nothing known", `{"prompt":"the whole task"}`, nil, "Claude Code Agent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := agentCallSubject(jsontext.Value(tc.input), tc.chosen); got != tc.want {
				t.Fatalf("subject = %q, want %q", got, tc.want)
			}
		})
	}
	// The subject is one clipped line, like every other progress subject.
	long := `{"model":"haiku","description":"` + strings.Repeat("long task ", 100) + `"}`
	got := agentCallSubject(jsontext.Value(long), nil)
	if len(got) > 512 || !strings.HasSuffix(got, "…") || !strings.HasPrefix(got, "Claude Code Agent · haiku: long task") || strings.ContainsAny(got, "\r\n\t") {
		t.Fatalf("clipped subject = %q (%d bytes)", got, len(got))
	}
}

// The chosen model is read only when there is a choice.
func TestAgentModelIsNilSafe(t *testing.T) {
	if model := (SubagentCLI{}).agentModel(); model != "" {
		t.Fatalf("model = %q without a choice", model)
	}
	if model := (SubagentCLI{AgentModel: func() string { return "opus" }}).agentModel(); model != "opus" {
		t.Fatalf("model = %q, want the choice", model)
	}
}
