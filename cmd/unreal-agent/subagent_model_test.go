package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/cmd/internal/agentrunner"
	"github.com/unreallabsai/unreal-agent/harness/credential"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/opencodego"
	"github.com/unreallabsai/unreal-agent/harness/modelcatalog"
	"github.com/unreallabsai/unreal-agent/harness/permission"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

// /model-sub's choice is one private file; no file is Codex, and anything
// else than Codex or an OpenCode Go model is refused rather than guessed.
func TestSubagentModelFileRoundTrip(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "subagent-model.json")
	if m, err := readSubagentModel(path); err != nil || m != (subagentModel{}) {
		t.Fatal("a missing file is not Codex", m, err)
	}
	for _, want := range []subagentModel{{Version: 1, Provider: opencodego.ProviderID, Model: "deepseek-v4.1-flash"}, {Version: 1, Provider: "openai-codex"}} {
		if err := writeSubagentModel(path, want); err != nil {
			t.Fatal(err)
		}
		if got, err := readSubagentModel(path); err != nil || got != want {
			t.Fatal("round trip", got, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("the file is not private", info, err)
	}
	if entries, _ := os.ReadDir(directory); len(entries) != 1 {
		t.Fatal("a temporary file was left behind", entries)
	}
	for _, invalid := range []string{`{`, `{"Version":2,"Provider":"openai-codex"}`, `{"Version":1,"Provider":"opencode-go"}`, `{"Version":1,"Provider":"openai-codex","Model":"gpt-6.1-sol"}`, `{"Version":1,"Provider":"claude","Model":"opus"}`, `{"Version":1,"Provider":"openai-codex","Extra":true}`} {
		if err := os.WriteFile(path, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if m, err := readSubagentModel(path); err == nil || m != (subagentModel{}) {
			t.Fatal("accepted", invalid)
		}
	}
	if m, err := readSubagentModel(""); err != nil || m != (subagentModel{}) {
		t.Fatal("no file configured", m, err)
	}
}

// P5: on Flash (low, high, max) the lead and worker-max run max, worker-low
// and the collector low, and the workers between high, the default.
func TestSubagentEffortFollowsP5(t *testing.T) {
	flash := modelcatalog.Model{ID: "deepseek-v4.1-flash", Efforts: []llm.ReasoningEffort{"low", "high", "max"}, DefaultEffort: "high"}
	want := map[string]llm.ReasoningEffort{"lead": "max", "worker-max": "max", "worker-xhigh": "high", "worker-high": "high", "worker-medium": "high", "worker-low": "low", "collector": "low"}
	for name, effort := range want {
		if got := subagentEffort(name, flash); got != effort {
			t.Errorf("%s runs %s on Flash, want %s", name, got, effort)
		}
	}
	// A model with the worker's own effort runs it.
	other := modelcatalog.Model{Efforts: []llm.ReasoningEffort{"low", "medium", "high", "ultra"}, DefaultEffort: "high"}
	if subagentEffort("worker-medium", other) != "medium" || subagentEffort("lead", other) != "high" || subagentEffort("collector", other) != "low" {
		t.Fatal("efforts on another model")
	}
}

// The Codex prompts stay as they were; on OpenCode Go they name the chosen
// model instead of the Codex models.
func TestSubagentPromptsNameTheChosenModel(t *testing.T) {
	if !strings.Contains(leadPrompt, "collector template (gpt-6-luna) and never to a gpt-6.1-sol worker") || subagentPromptFor(leadPrompt, "openai-codex", "gpt-6.1-sol") != leadPrompt {
		t.Fatal("the Codex lead prompt changed")
	}
	retargeted := subagentPromptFor(leadPrompt, opencodego.ProviderID, "deepseek-v4.1-flash")
	if !strings.Contains(retargeted, "collector template (deepseek-v4.1-flash at low effort) and never to a higher-effort worker") || strings.Contains(retargeted, "gpt-6") {
		t.Fatal("lead prompt on OpenCode Go", retargeted)
	}
	for _, prompt := range []string{workerPrompt, collectorPrompt} {
		if subagentPromptFor(prompt, opencodego.ProviderID, "deepseek-v4.1-flash") != prompt {
			t.Fatal("a prompt without model names changed")
		}
	}
	path := filepath.Join(t.TempDir(), "subagent-model.json")
	models := subagentModels(path)
	if m := models(); m.Family != "" {
		t.Fatal("Claude is told of another model without a choice", m)
	}
	if err := writeSubagentModel(path, subagentModel{Version: 1, Provider: opencodego.ProviderID, Model: "deepseek-v4.1-flash"}); err != nil {
		t.Fatal(err)
	}
	if m := models(); m.Family != "OpenCode Go" || m.Lead != "deepseek-v4.1-flash" || m.Collector != "" {
		t.Fatal("Claude's prompt does not name the chosen model", m)
	}
}

// Without Codex the templates exist only when OpenCode Go is the chosen
// model, and then run it.
func TestDefaultSubagentsOnOpenCodeGoWithoutCodex(t *testing.T) {
	workspace := t.TempDir()
	cfg := codexServeConfiguration(workspace, "https://codex.invalid")
	cfg.Runtime.Provider.Provider = "claude"
	maximum := permission.Config{Tools: append(tool.StaticNames(), tool.ClaudeCodeName), FilesystemUnrestricted: true, NetworkUnrestricted: true, ProcessMode: permission.ProcessUnrestricted}
	cfg.PermissionModes = &permission.ModesConfig{Enabled: true, Maximum: &maximum}
	if defaultSubagents(cfg, subagentModel{}) != nil || defaultSubagents(cfg, subagentModel{Version: 1, Provider: "openai-codex"}) != nil {
		t.Fatal("templates without Codex or a chosen model")
	}
	templates := defaultSubagents(cfg, subagentModel{Version: 1, Provider: opencodego.ProviderID, Model: "deepseek-v4.1-flash"})
	lead, ok := templates["lead"]
	if !ok || len(templates) != 2+len(workerEfforts) || len(lead.Workers) != 1+len(workerEfforts) {
		t.Fatal("templates", len(templates))
	}
	for name, template := range templates {
		s := template.Runtime.Provider
		if s.Provider != opencodego.ProviderID || s.Model.ID != "deepseek-v4.1-flash" || s.Endpoint != opencodego.BaseURL || s.Auth != openCodeGoKey {
			t.Fatalf("%s: %+v", name, s)
		}
	}
	// A registered route is used as registered.
	route := openCodeGoRoute("deepseek-v4.1-flash")
	route.Endpoint = "http://127.0.0.1:9/inference/go/openai/v1"
	cfg.Providers = map[string]agentrunner.ProviderRuntime{opencodego.ProviderID: {Provider: route}}
	if lead := defaultSubagents(cfg, subagentModel{Version: 1, Provider: opencodego.ProviderID, Model: "deepseek-v4.1-flash"})["lead"]; lead.Runtime.Provider.Endpoint != route.Endpoint {
		t.Fatal("the registered route was not used", lead.Runtime.Provider)
	}
}

// A start on the chosen model records the lead's workers rebound too;
// recovery accepts exactly that, and refuses a worker changed otherwise.
func TestSubagentTemplateValidatorAcceptsReboundWorkers(t *testing.T) {
	workspace := t.TempDir()
	cfg := codexServeConfiguration(workspace, "https://codex.invalid")
	route := openCodeGoRoute("deepseek-v4.1-flash")
	cfg.Providers = map[string]agentrunner.ProviderRuntime{opencodego.ProviderID: {Provider: route, ReasoningEffort: "high"}}
	cfg.Permissions = permission.Config{Tools: []string{"read", "grep", "glob"}, ReadRoots: []string{workspace}}
	maximum := permission.Config{Tools: append(tool.StaticNames(), childToolNames...), FilesystemUnrestricted: true, NetworkUnrestricted: true, ProcessMode: permission.ProcessUnrestricted}
	cfg.PermissionModes = &permission.ModesConfig{Enabled: true, Maximum: &maximum}
	registry, err := registeredProviders(cfg.Runtime, cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	resolve := credential.ResolverFunc(func(context.Context, credential.Reference) (credential.Material, error) {
		return credential.Material{}, nil
	})
	parent, err := permission.New(maximum)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	c, templates, err := withParentSubagents(agentrunner.RuntimeConfig{Identity: cfg.Runtime, SessionDirectory: t.TempDir(), Providers: registry, Credentials: resolve, Backends: cfg.Providers}, defaultSubagents(cfg, subagentModel{}), parent, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	validate := childTemplateValidator(c)
	base := templates["lead"]
	backend, err := c.ProviderRuntime(opencodego.ProviderID)
	if err != nil {
		t.Fatal(err)
	}
	flash := modelcatalog.Model{ID: "deepseek-v4.1-flash", Efforts: []llm.ReasoningEffort{"low", "high", "max"}, DefaultEffort: "high"}
	chosen := base
	if chosen.Runtime, err = boundChildRuntime(c, base.Runtime, backend, flash.ID, subagentEffort("lead", flash), true); err != nil {
		t.Fatal(err)
	}
	chosen.Subagents = maps.Clone(base.Subagents)
	for name, worker := range base.Subagents {
		if worker.Runtime, err = boundChildRuntime(c, worker.Runtime, backend, flash.ID, subagentEffort(name, flash), true); err != nil {
			t.Fatal(err)
		}
		chosen.Subagents[name] = worker
	}
	if err = validate(base, chosen); err != nil {
		t.Fatal("recovery refused the chosen model's start", err)
	}
	for name, worker := range chosen.Subagents {
		s := sessionstore.SelectionFromConfiguration(worker.Runtime)
		if s == nil || s.Provider != opencodego.ProviderID || s.Effort != subagentEffort(name, flash) {
			t.Fatal("worker", name, s)
		}
	}
	// The Codex start is accepted as before.
	unchanged := base
	if unchanged.Runtime, err = boundChildRuntime(c, base.Runtime, mustProvider(t, c, "openai-codex"), leadModel, leadEffort, false); err != nil {
		t.Fatal(err)
	}
	if err = validate(base, unchanged); err != nil {
		t.Fatal("recovery refused a Codex start", err)
	}
	// A worker with another prompt, or a renamed worker, is refused.
	tampered := chosen
	tampered.Subagents = maps.Clone(chosen.Subagents)
	worker := tampered.Subagents["worker-high"]
	var identity agentrunner.RuntimeIdentity
	if err = json.Unmarshal(worker.Runtime, &identity); err != nil {
		t.Fatal(err)
	}
	identity.SystemPrompt = "Do something else."
	if worker.Runtime, err = json.Marshal(identity); err != nil {
		t.Fatal(err)
	}
	tampered.Subagents["worker-high"] = worker
	if validate(base, tampered) == nil {
		t.Fatal("recovery accepted a worker with another prompt")
	}
	renamed := chosen
	renamed.Subagents = maps.Clone(chosen.Subagents)
	renamed.Subagents["worker-extra"] = renamed.Subagents["worker-high"]
	delete(renamed.Subagents, "worker-high")
	if validate(base, renamed) == nil {
		t.Fatal("recovery accepted a renamed worker")
	}
	if names := slices.Sorted(maps.Keys(chosen.Subagents)); !slices.Equal(names, slices.Sorted(maps.Keys(base.Subagents))) {
		t.Fatal("workers", names)
	}
}

func mustProvider(t *testing.T, c agentrunner.RuntimeConfig, id string) agentrunner.ProviderRuntime {
	t.Helper()
	p, err := c.ProviderRuntime(id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A child reads OpenCode Go's key from the launcher's store as its Host does,
// never creates the store, and an explicit store comes first.
func TestChildCredentialsReadTheDefaultStore(t *testing.T) {
	store := filepath.Join(privateCLIDirectory(t), "credentials")
	resolver, err := childCredentials("", store, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = resolver.Resolve(t.Context(), openCodeGoKey); !hasCredentialCode(err, "store_required") {
		t.Fatal("a missing store", err)
	}
	if _, err = os.Lstat(store); !os.IsNotExist(err) {
		t.Fatal("the child created the store", err)
	}
	storeKey(t, store, openCodeGoKey, "opencode-go-key-sensitive")
	m, err := resolver.Resolve(t.Context(), openCodeGoKey)
	if err != nil || m.Token.Reveal() != "opencode-go-key-sensitive" {
		t.Fatal("the saved key was not read", err)
	}
	explicit := filepath.Join(privateCLIDirectory(t), "credentials")
	storeKey(t, explicit, openCodeGoKey, "explicit-key-sensitive")
	if resolver, err = childCredentials(explicit, store, ""); err != nil {
		t.Fatal(err)
	}
	if m, err = resolver.Resolve(t.Context(), openCodeGoKey); err != nil || m.Token.Reveal() != "explicit-key-sensitive" {
		t.Fatal("the explicit store was not used", err)
	}
	if _, err = childCredentials("", "relative/credentials", ""); err == nil {
		t.Fatal("a relative store was accepted")
	}
}

func hasCredentialCode(err error, code string) bool {
	var e *credential.Error
	return errors.As(err, &e) && e.Code == code
}
