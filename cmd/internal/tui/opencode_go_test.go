package tui

import (
	"strings"
	"testing"
)

// /login explains how to save the OpenCode Go key in another shell; the TUI
// itself never asks for it (TestLoginShowsSubscriptionInstructionsWithoutRequestingKeys).
func TestLoginShowsOpenCodeGoKeyCommand(t *testing.T) {
	command := "pbpaste | /opt/unreal/unreal-agent-auth login --store ~/.local/state/unreal-agent/credentials --provider opencode-go --id primary --key-stdin"
	for _, given := range []string{command, ""} {
		f, keys, out := newFake(), make(chan Key, 8), &screenObserver{}
		done := make(chan error, 1)
		go func() {
			done <- Run(t.Context(), Config{Client: f, ID: "s", Keys: keys, Output: out, Theme: &Theme{Plain: true}, Size: func() (int, int) { return 260, 40 }, OpenCodeGoKeyCommand: given})
		}()
		waitFor(t, func() bool { return strings.Contains(out.text(), "入力待ち") })
		keys <- Key{Text: "/login"}
		keys <- Key{Name: "enter"}
		waitFor(t, func() bool { return strings.Contains(out.text(), "Extra Usage") })
		text := out.text()
		for _, want := range []string{"codex login", "claude を起動", "OpenCode Go", "キーは画面に出ません", "/model refresh", "docs/opencode-go.md"} {
			if !strings.Contains(text, want) {
				t.Fatal(want, text)
			}
		}
		if strings.Contains(text, command) != (given != "") || given == "" && strings.Contains(text, "pbpaste |") {
			t.Fatal("key command", given != "", text)
		}
		for _, gone := range []string{"PROVIDER ID", "API key", "store key", "private:", "pkill"} {
			if strings.Contains(text, gone) {
				t.Fatal(gone, text)
			}
		}
		keys <- Key{Name: "detach"}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		f.mu.Lock()
		if f.logins != 0 || len(f.inputs) != 0 {
			t.Fatal("login requested a key or wrote conversation")
		}
		f.mu.Unlock()
	}
}
