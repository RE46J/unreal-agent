package tui

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/host"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

// Exercise the real Host projection and JSON transport on a synthetic 38 MB,
// 6000-item Session. Never opens a real Session or calls a provider.
func copyBenchmarkSession(tb testing.TB) (startupClient, sessionstore.Sequence, int) {
	return copySessionFixture(tb, func(add func(sessionstore.ItemKind, any)) {
		var previous session.TurnID
		for i := range 2000 {
			payload, _ := json.Marshal(fmt.Sprintf("copy input %04d", i))
			add(sessionstore.ItemInput, inbox.Input{ID: inbox.ID(fmt.Sprintf("i-%d", i)), Kind: inbox.InputExternal, Payload: payload})
			turn := session.TurnID(fmt.Sprintf("t-%d", i))
			add(sessionstore.ItemTurn, session.Turn{ID: turn, PreviousTurnID: previous, Type: session.TurnRegular})
			add(sessionstore.ItemModelResponse, sessionstore.ModelResponse{TurnID: turn, Response: llm.Response{Output: []llm.Item{{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: fmt.Sprintf("copy response %04d\n", i) + strings.Repeat("ordinary message body\n", 800)}}}}})
			previous = turn
		}
	})
}

func copySessionFixture(tb testing.TB, fill func(func(sessionstore.ItemKind, any))) (startupClient, sessionstore.Sequence, int) {
	tb.Helper()
	dir := tb.TempDir()
	var log strings.Builder
	line := func(kind string, data any) {
		raw, err := json.Marshal(struct {
			Type string `json:"type"`
			Data any    `json:"data"`
		}{kind, data})
		if err != nil {
			tb.Fatal(err)
		}
		log.Write(raw)
		log.WriteByte('\n')
	}
	line("session", struct {
		Version int
		Session session.Session
	}{3, session.Session{ID: "copy-bench", CreatedAt: time.Unix(1700000000, 0).UTC()}})
	var seq sessionstore.Sequence
	add := func(kind sessionstore.ItemKind, data any) {
		seq++
		line("item", struct{ Item sessionstore.Item }{sessionstore.Item{Sequence: seq, RecordedAt: time.Unix(1700000000, 0).UTC(), Kind: kind, Data: data}})
	}
	fill(add)
	if err := os.WriteFile(filepath.Join(dir, "copy-bench.session.jsonl"), []byte(log.String()), 0600); err != nil {
		tb.Fatal(err)
	}
	h, err := host.New(tb.Context(), host.Config{Directory: dir, Build: func(ctx context.Context, _ session.ID) (host.Runtime, error) {
		return host.Runtime{Builder: contextbuilder.NewBuilder(), LLM: startupAdapter{}, Tools: tool.NewRegistry(tool.StaticTranslators{}), Operations: operation.NewLocalOperationManager(ctx)}, nil
	}})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { h.Close() })
	s, err := h.Resume(tb.Context(), host.Options{ID: "copy-bench"})
	if err != nil {
		tb.Fatal(err)
	}
	return startupClient{s}, seq, log.Len()
}

type copyMeasuredClient struct {
	startupClient
	readBytes int
}

func (c *copyMeasuredClient) transport(v host.View, err error) (host.View, error) {
	if err != nil {
		return v, err
	}
	raw, err := json.Marshal(v)
	c.readBytes += len(raw)
	if err != nil {
		return host.View{}, err
	}
	var decoded host.View
	err = json.Unmarshal(raw, &decoded)
	return decoded, err
}

func (c *copyMeasuredClient) Inspect(ctx context.Context, id session.ID, after sessionstore.Sequence, limit int) (host.View, error) {
	return c.transport(c.startupClient.Inspect(ctx, id, after, limit))
}

func (c *copyMeasuredClient) InspectBefore(ctx context.Context, id session.ID, before sessionstore.Sequence, limit int) (host.View, error) {
	return c.transport(c.startupClient.InspectBefore(ctx, id, before, limit))
}

func BenchmarkCopyLargeSession(b *testing.B) {
	client, through, size := copyBenchmarkSession(b)
	for _, action := range []string{"open", "selected", "all"} {
		b.Run(action, func(b *testing.B) {
			r := &copyMeasuredClient{startupClient: client}
			p, err := loadCopyPicker(b.Context(), r, client.owner.ID, through)
			if err != nil {
				b.Fatal(err)
			}
			if action == "selected" {
				p.Selection = 1
			}
			r.readBytes = 0
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if action == "open" {
					p, err = loadCopyPicker(b.Context(), r, client.owner.ID, through)
					if err == nil {
						RenderFrame(idleSnapshot(), UIState{Picker: p, Theme: Theme{Plain: true}}, 120, 32)
					}
				} else {
					var text string
					text, err = copyPickerText(b.Context(), r, client.owner.ID, p)
					if err == nil {
						err = writeClipboard(io.Discard, text)
					}
				}
				if err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(r.readBytes)/float64(b.N), "read-B/op")
			b.ReportMetric(float64(size), "session-B")
		})
	}
}

func BenchmarkClipboardWrite(b *testing.B) {
	for _, size := range []int{16 * 1024, 32 * 1024 * 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			text := strings.Repeat("x", size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := writeClipboard(io.Discard, text); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
