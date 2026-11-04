package main

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"maps"

	"github.com/unreallabsai/unreal-agent/cmd/internal/agentrunner"
	"github.com/unreallabsai/unreal-agent/harness/host"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/opencodego"
	"github.com/unreallabsai/unreal-agent/harness/permission"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/subagent"
)

// childRuntimeResolver binds a child start to a registered backend. With
// modelFile, a start of a built-in template that names no runtime runs the
// sub-agents' model chosen with /model-sub, and so do a lead's workers;
// a choice that cannot run fails the start, nothing falls back to Codex.
func childRuntimeResolver(c agentrunner.RuntimeConfig, modelFile string) subagent.RuntimeResolver {
	return func(ctx context.Context, owner *host.Session, name string, template subagent.Template, requested *subagent.RuntimeRequest) (subagent.Template, error) {
		// The child follows the Session's mode; only its maximum bounds the start.
		ctx = owner.WithCapabilityPolicy(ctx)
		fail := func(code string) (subagent.Template, error) { return template, &subagent.RuntimeError{Code: code} }
		chosen := false
		if requested == nil && modelFile != "" && template.Default != nil {
			m, e := readSubagentModel(modelFile)
			if e != nil {
				return fail("subagent_model_unreadable")
			}
			if m.Provider == opencodego.ProviderID {
				// The effort is the role's on this model, bound below.
				requested, chosen = &subagent.RuntimeRequest{Provider: m.Provider, Model: m.Model, Effort: template.Default.Effort}, true
			} else if template.Default.Provider != "openai-codex" {
				return fail("provider_unavailable")
			}
		}
		if requested == nil {
			requested = template.Default
		}
		if e := permission.FromContext(ctx).CheckTool("SubagentStart"); e != nil {
			return fail("permission_denied")
		}
		active, _ := owner.RuntimeSelection()
		if active == nil {
			return fail("selection_unavailable")
		}
		choice := *active
		if requested != nil {
			choice = sessionstore.RuntimeSelection{Version: 1, Provider: requested.Provider, Model: requested.Model, Effort: requested.Effort.Normalize()}
		}
		backend, e := c.ProviderRuntime(choice.Provider)
		if e != nil {
			return fail("invalid_provider")
		}
		if choice.Validate() != nil {
			return fail("invalid_selection")
		}
		// Authorize the owning Session before any CLI health/catalog process.
		if e = permission.FromContext(ctx).CheckProcess(); e != nil {
			return fail("permission_denied")
		}
		for _, name := range template.Policy.Tools {
			if e = permission.FromContext(ctx).CheckTool(name); e != nil {
				return fail("permission_denied")
			}
		}
		if c.CheckProvider(ctx, choice.Provider) != nil {
			return fail("provider_unavailable")
		}
		catalog, e := c.Catalog(ctx, choice.Provider)
		if e != nil {
			return fail("provider_unavailable")
		}
		model, e := catalog.FindSelection(choice.Model)
		if e != nil && !catalog.Authoritative && choice.Provider == active.Provider && choice.Model == active.Model && choice.Effort == active.Effort {
			model.ID, model.Name, model.Efforts = active.Model, active.Name, []llm.ReasoningEffort{active.Effort}
			e = nil
		}
		if e != nil {
			return fail("invalid_model")
		}
		if chosen {
			choice.Effort = subagentEffort(name, model)
		}
		if !model.AllowsEffort(choice.Effort) {
			return fail("unsupported_effort")
		}
		raw, e := boundChildRuntime(c, template.Runtime, backend, choice.Model, choice.Effort, chosen)
		if e != nil {
			return template, e
		}
		template.Runtime = raw
		if chosen && len(template.Subagents) > 0 {
			workers := make(map[string]subagent.Template, len(template.Subagents))
			for worker, t := range template.Subagents {
				if t.Runtime, e = boundChildRuntime(c, t.Runtime, backend, choice.Model, subagentEffort(worker, model), true); e != nil {
					return template, e
				}
				workers[worker] = t
			}
			template.Subagents = workers
		}
		return template, nil
	}
}

// boundChildRuntime is a template's runtime on a registered backend, model
// and effort; chosen gives a built-in prompt for that model.
func boundChildRuntime(c agentrunner.RuntimeConfig, runtime jsontext.Value, backend agentrunner.ProviderRuntime, model string, effort llm.ReasoningEffort, chosen bool) (jsontext.Value, error) {
	var identity agentrunner.RuntimeIdentity
	if json.Unmarshal(runtime, &identity) != nil {
		return nil, &subagent.RuntimeError{Code: "invalid_template"}
	}
	identity.Provider = backend.Provider
	// Keep the exact requested/applied selection. A catalog row may match a
	// resolved version as well as its alias; inheritance must not silently
	// replace a pinned version with the alias (or break stable-ID retries).
	identity.Provider.Model.ID = model
	identity.ReasoningEffort = effort
	identity.Claude = backend.Claude
	if chosen {
		identity.SystemPrompt = subagentPromptFor(identity.SystemPrompt, backend.Provider.Provider, model)
	}
	registry, e := c.Providers.WithModel(identity.Provider)
	if e != nil {
		return nil, &subagent.RuntimeError{Code: "invalid_model"}
	}
	_, raw, e := agentrunner.NewRuntimeFactory(agentrunner.RuntimeConfig{Identity: identity, SessionDirectory: c.SessionDirectory, Providers: registry, Credentials: c.Credentials})
	if e != nil {
		return nil, &subagent.RuntimeError{Code: "incompatible_runtime"}
	}
	return raw, nil
}

// Recovery validates the recorded configuration, never re-inherits the parent's
// current selection or replaces the child's catalog choice.
func childTemplateValidator(c agentrunner.RuntimeConfig) func(subagent.Template, subagent.Template) error {
	return func(base, chosen subagent.Template) error {
		fail := func() error { return &subagent.RuntimeError{Code: "configuration_mismatch"} }
		if validChildRuntime(c, base.Runtime, chosen.Runtime) != nil {
			return fail()
		}
		base.Runtime = chosen.Runtime
		// A start on the sub-agents' model binds the lead's workers too.
		if len(base.Subagents) > 0 && len(chosen.Subagents) == len(base.Subagents) {
			workers := maps.Clone(base.Subagents)
			for name, worker := range chosen.Subagents {
				if t, ok := workers[name]; ok && validChildRuntime(c, t.Runtime, worker.Runtime) == nil {
					t.Runtime = worker.Runtime
					workers[name] = t
				}
			}
			base.Subagents = workers
		}
		// Stored plans come back canonicalized, nested worker runtimes too.
		old, _ := json.Marshal(base)
		now, _ := json.Marshal(chosen)
		was, is := jsontext.Value(old), jsontext.Value(now)
		if was.Canonicalize() != nil || is.Canonicalize() != nil || !bytes.Equal(was, is) {
			return fail()
		}
		return nil
	}
}

// validChildRuntime reports whether recorded is base bound to a registered
// backend, as the resolver binds it.
func validChildRuntime(c agentrunner.RuntimeConfig, base, recorded jsontext.Value) error {
	mismatch := errors.New("child runtime mismatch")
	var a, b agentrunner.RuntimeIdentity
	if json.Unmarshal(base, &a) != nil || json.Unmarshal(recorded, &b) != nil {
		return mismatch
	}
	backend, e := c.ProviderRuntime(b.Provider.Provider)
	if e != nil {
		return mismatch
	}
	registered := backend.Provider
	registered.Model.ID = b.Provider.Model.ID
	registry, e := c.Providers.WithModel(registered)
	if e != nil {
		return mismatch
	}
	registered, e = registry.Resolve(registered)
	if e != nil {
		return mismatch
	}
	expected, _ := json.Marshal(registered)
	actual, _ := json.Marshal(b.Provider)
	if !bytes.Equal(expected, actual) {
		return mismatch
	}
	a.Provider = registered
	a.ReasoningEffort = b.ReasoningEffort
	want := recorded.Clone()
	if want.Canonicalize() != nil {
		return mismatch
	}
	// The template's own prompt, or the built-in one for the sub-agents' model.
	for _, prompt := range []string{a.SystemPrompt, subagentPromptFor(a.SystemPrompt, registered.Provider, registered.Model.ID)} {
		a.SystemPrompt = prompt
		// The same factory normalization used at creation fills catalog display
		// names and resolved operator defaults. It performs no auth/inference and
		// cannot add a permission or replace the recorded model on recovery.
		_, expectedRuntime, e := agentrunner.NewRuntimeFactory(agentrunner.RuntimeConfig{Identity: a, SessionDirectory: c.SessionDirectory, Providers: registry, Credentials: c.Credentials})
		if e == nil && expectedRuntime.Canonicalize() == nil && bytes.Equal(expectedRuntime, want) {
			return nil
		}
	}
	return mismatch
}
