package main

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/claudeagent"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/opencodego"
	"github.com/unreallabsai/unreal-agent/harness/modelcatalog"
)

// subagentModel is the sub-agents' model chosen with /model-sub (U15, U16).
// The launcher keeps it in one file under STATE, so every conversation shares
// it, and each child start reads it. No file, or Codex, starts the built-in
// templates as built.
type subagentModel struct {
	Version  int
	Provider string
	Model    string `json:",omitzero"`
}

func readSubagentModel(path string) (subagentModel, error) {
	var m subagentModel
	if path == "" {
		return m, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if json.Unmarshal(data, &m, json.RejectUnknownMembers(true)) != nil || m.Version != 1 ||
		!(m.Provider == "openai-codex" && m.Model == "" || m.Provider == opencodego.ProviderID && m.Model != "") {
		return subagentModel{}, errors.New("invalid sub-agent model file")
	}
	return m, nil
}

// writeSubagentModel replaces the file whole, so a child start never reads a
// partial choice.
func writeSubagentModel(path string, m subagentModel) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".subagent-model-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// subagentEffort is the effort a built-in template runs at on the sub-agents'
// model (P5): the lead and worker-max at the model's highest, worker-low and
// the collector at its lowest, another worker at its own effort when the
// model has it and at the model's default otherwise.
func subagentEffort(name string, m modelcatalog.Model) llm.ReasoningEffort {
	var offered []llm.ReasoningEffort
	for _, e := range workerEfforts {
		if slices.Contains(m.Efforts, e) {
			offered = append(offered, e)
		}
	}
	if len(offered) == 0 {
		return m.DefaultEffort
	}
	switch name {
	case "lead", "worker-max":
		return offered[len(offered)-1]
	case "worker-low", "collector":
		return offered[0]
	}
	if e := llm.ReasoningEffort(strings.TrimPrefix(name, "worker-")); slices.Contains(offered, e) {
		return e
	}
	return m.DefaultEffort
}

// codexCollectorRule names the Codex models in the lead's prompt.
const codexCollectorRule = "collector template (" + collectorModel + ") and never to a " + leadModel + " worker"

// subagentPromptFor is a built-in prompt for the model it runs: on OpenCode
// Go every template runs the one chosen model, the collector at low effort.
func subagentPromptFor(prompt, provider, model string) string {
	if provider != opencodego.ProviderID {
		return prompt
	}
	return strings.Replace(prompt, codexCollectorRule, "collector template ("+model+" at low effort) and never to a higher-effort worker", 1)
}

// subagentModels names the sub-agents' model for Claude's prompt, from the
// file at each turn.
func subagentModels(path string) func() claudeagent.SubagentModels {
	return func() claudeagent.SubagentModels {
		m, err := readSubagentModel(path)
		if err != nil || m.Provider != opencodego.ProviderID {
			return claudeagent.SubagentModels{}
		}
		return claudeagent.SubagentModels{Family: modelcatalog.ProviderName(m.Provider), Lead: m.Model}
	}
}
