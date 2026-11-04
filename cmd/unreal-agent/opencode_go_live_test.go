//go:build linux || darwin

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/host"
	"github.com/unreallabsai/unreal-agent/harness/host/gateway"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/opencodego"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

// The live Host check starts a normal Host on the user's stored key, reads
// the real model list, switches the Session to OpenCode Go and sends one turn
// (docs/opencode-go.md, U13). It runs only with UNREAL_LIVE_OPENCODE_GO=1.
// The test never reads the key; the Host does, for each request.
func TestLiveOpenCodeGoThroughHost(t *testing.T) {
	if os.Getenv("UNREAL_LIVE_OPENCODE_GO") != "1" {
		t.Skip("set UNREAL_LIVE_OPENCODE_GO=1 to send one turn through a Host with the stored key")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(home, ".local", "state", "unreal-agent", "credentials")
	f, cli := normalFixture(t, true, true)
	l := launcher{start: f.start, attach: func(ctx context.Context, c *gateway.Client, id session.ID) error {
		out, err := modelExchange(ctx, c.Extension, modelRequest{Action: "model.catalog", ID: id, Provider: opencodego.ProviderID, Refresh: true})
		if err != nil || out.Catalog == nil || !out.Catalog.Available || len(out.Catalog.Models) != 1 {
			t.Fatal("real model list", err)
		}
		model := out.Catalog.Models[0]
		t.Logf("model list: %s %q efforts %v default %s context %d", model.ID, model.Name, model.Efforts, model.DefaultEffort, model.ContextWindow)
		v, err := c.Inspect(ctx, id, 0, 128)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("permission mode: %s", v.Permissions.Mode)
		stopMultiSession(t, c, v)
		v, err = c.Inspect(ctx, id, 0, 128)
		if err != nil {
			t.Fatal(err)
		}
		choice := sessionstore.RuntimeSelection{Version: 1, RequestID: "live-opencode-go", Provider: opencodego.ProviderID, Model: model.ID, Effort: llm.ReasoningEffortLow}
		if _, err = modelExchange(ctx, c.Extension, modelRequest{Action: "model.select", ID: id, Generation: v.Generation, Selection: choice}); err != nil {
			t.Fatal("switch to OpenCode Go", err)
		}
		resumed, err := c.Open(ctx, host.Resume, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Submit(ctx, id, resumed.Generation, inbox.Input{ID: "live-turn", Kind: inbox.InputExternal, Payload: []byte(`"Reply with the single word: ok"`)}); err != nil {
			t.Fatal(err)
		}
		end, text := liveAwait(t, c, id, 3*time.Minute)
		if end.Failure != "" {
			t.Fatal("turn failed:", end.Failure)
		}
		if selection := multiReport(t, end).Selection; selection == nil || selection.Provider != opencodego.ProviderID {
			t.Fatal("the turn did not run on OpenCode Go")
		}
		var usage llm.Usage
		for _, item := range end.History.Items {
			if r, ok := item.Data.(sessionstore.ModelResponse); ok {
				usage = r.Response.Usage
			}
		}
		t.Logf("reply %q, input %d, cached %d, output %d tokens", text, usage.InputTokens, usage.CachedInputTokens, usage.OutputTokens)
		if !strings.Contains(strings.ToLower(text), "ok") {
			t.Error("unexpected reply", text)
		}
		if f.calls.Load() != 0 || len(cli.Calls(t)) != 0 {
			t.Fatal("the turn went to Codex or Claude")
		}
		return nil
	}}
	if err := l.run(t.Context(), []string{"--credential-directory", store}, io.Discard); err != nil {
		t.Fatal(err)
	}
}
