package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/unreallabsai/unreal-agent/harness/host"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

// MessageChoice retains only a canonical reference and a safe first line.
// Part distinguishes separate public statements in one model response.
type MessageChoice struct {
	Sequence      sessionstore.Sequence
	Part          int
	Role, Preview string
}

func walkCopyMessages(ctx context.Context, r Reader, id session.ID, through sessionstore.Sequence, visit func(MessageChoice, string)) error {
	walker := copyMessageWalker{internal: map[session.TurnID]bool{}}
	return readHistory(ctx, r, id, 0, through, func(item host.HistoryItem) bool {
		walker.apply(item, visit)
		return true
	})
}

type copyMessageWalker struct {
	internal   map[session.TurnID]bool
	background bool
}

func copyPageWalker(page host.HistoryPage) copyMessageWalker {
	w := copyMessageWalker{internal: map[session.TurnID]bool{}, background: page.Background}
	if c := page.CopyContext; c != nil {
		w.background = c.Background
		for _, turn := range c.InternalTurns {
			w.internal[turn] = true
		}
	}
	return w
}

func (w *copyMessageWalker) apply(item host.HistoryItem, visit func(MessageChoice, string)) {
	emit := func(part int, role, text string) {
		if text == "" {
			return
		}
		first, _, _ := strings.Cut(text, "\n")
		visit(MessageChoice{Sequence: item.Sequence, Part: part, Role: role, Preview: strings.Clone(clipBytes(first, 240))}, text)
	}
	switch data := item.Data.(type) {
	case session.Turn:
		w.internal[data.ID] = data.Type == session.TurnCompaction
	case inbox.Input:
		if data.Kind == inbox.InputExternal {
			w.background = false
			emit(0, "You", publicUserText(data))
		}
		if data.Kind == inbox.InputPeer {
			if peer, err := data.DecodePeerMessage(); err != nil || !quietPeerKind(peer.Kind) {
				w.background = true
			}
		}
	case sessionstore.ModelResponse:
		if w.internal[data.TurnID] || publicAgentText(data) == "" {
			break
		}
		for part, output := range data.Response.Output {
			if message, ok := publicAgentPart(output); ok && (!w.background || forHuman(message.Text)) {
				emit(part, "Agent", exportText(message.Text))
			}
		}
	}
}

type copyPage struct {
	messages []MessageChoice
	before   sessionstore.Sequence
	older    bool
}

// Read from the fixed, exclusive boundary. Only scroll requests move it toward
// the beginning; live appends cannot change an already opened copy picker.
func loadCopyPage(ctx context.Context, r Reader, id session.ID, before sessionstore.Sequence) (copyPage, error) {
	var page copyPage
	collect := func(choice MessageChoice, _ string) { page.messages = append(page.messages, choice) }
	if backward, ok := r.(BackwardReader); ok {
		for {
			v, err := backward.InspectBefore(ctx, id, before, historyPageItems)
			if err != nil {
				return copyPage{}, exportFailure(ctx, err, "history_unavailable")
			}
			w := copyPageWalker(v.History)
			for _, item := range v.History.Items {
				w.apply(item, collect)
			}
			page.older = v.History.Older
			if len(v.History.Items) > 0 {
				page.before = v.History.Items[0].Sequence
			}
			if len(page.messages) > 0 || !page.older {
				break
			}
			before = page.before
		}
	} else if err := walkCopyMessages(ctx, r, id, before-1, collect); err != nil {
		return copyPage{}, exportFailure(ctx, err, "history_unavailable")
	}
	slices.Reverse(page.messages)
	return page, nil
}

// CopyChoices returns the recent page. The picker loads older choices on scroll.
func CopyChoices(ctx context.Context, r Reader, id session.ID, through sessionstore.Sequence) ([]MessageChoice, error) {
	page, err := loadCopyPage(ctx, r, id, through+1)
	return page.messages, err
}

func CopyMessagesText(ctx context.Context, r Reader, id session.ID, choices []MessageChoice) (string, error) {
	if len(choices) == 0 {
		return "", &ExportError{Code: "no_conversation"}
	}
	type reference struct {
		sequence sessionstore.Sequence
		part     int
	}
	selected := map[reference]bool{}
	var through sessionstore.Sequence
	var sequences []sessionstore.Sequence
	seen := map[sessionstore.Sequence]bool{}
	for _, choice := range choices {
		selected[reference{choice.Sequence, choice.Part}] = true
		through = max(through, choice.Sequence)
		if !seen[choice.Sequence] {
			seen[choice.Sequence] = true
			sequences = append(sequences, choice.Sequence)
		}
	}
	var bodies []string
	collect := func(choice MessageChoice, text string) {
		key := reference{choice.Sequence, choice.Part}
		if !selected[key] {
			return
		}
		delete(selected, key)
		if len(choices) > 1 {
			text = "## " + choice.Role + "\n\n" + text
		}
		bodies = append(bodies, text)
	}
	var err error
	if backward, ok := r.(BackwardReader); ok {
		slices.Sort(sequences)
		for _, sequence := range sequences {
			var v host.View
			v, err = backward.InspectBefore(ctx, id, sequence+1, 1)
			if err != nil {
				break
			}
			w := copyPageWalker(v.History)
			for _, item := range v.History.Items {
				w.apply(item, collect)
			}
		}
	} else {
		err = walkCopyMessages(ctx, r, id, through, collect)
	}
	if err != nil {
		return "", exportFailure(ctx, err, "history_unavailable")
	}
	if len(selected) > 0 {
		return "", &ExportError{Code: "response_unavailable"}
	}
	return strings.Join(bodies, "\n\n"), nil
}

func copyPicker(choices []MessageChoice) *Picker {
	p := &Picker{Kind: "copy", Title: "Copy to the clipboard", Hint: "↑↓ move · Space select · Enter copy", Messages: choices, Checked: map[int]bool{}, Options: []string{"すべて"}}
	for _, choice := range choices {
		p.Options = append(p.Options, fmt.Sprintf("%s  %s", choice.Role, choice.Preview))
	}
	return p
}

func loadCopyPicker(ctx context.Context, r Reader, id session.ID, through sessionstore.Sequence) (*Picker, error) {
	page, err := loadCopyPage(ctx, r, id, through+1)
	p := copyPicker(page.messages)
	p.CopyBefore, p.CopyThrough, p.CopyOlder = page.before, through, page.older
	return p, err
}

func copyPickerText(ctx context.Context, r Reader, id session.ID, p *Picker) (string, error) {
	if p.copyAll() {
		var bodies []string
		err := walkCopyMessages(ctx, r, id, p.CopyThrough, func(choice MessageChoice, text string) {
			bodies = append(bodies, "## "+choice.Role+"\n\n"+text)
		})
		if err != nil {
			return "", exportFailure(ctx, err, "history_unavailable")
		}
		if len(bodies) == 0 {
			return "", &ExportError{Code: "no_conversation"}
		}
		if len(bodies) == 1 {
			_, text, _ := strings.Cut(bodies[0], "\n\n")
			return text, nil
		}
		return strings.Join(bodies, "\n\n"), nil
	}
	return CopyMessagesText(ctx, r, id, p.copySelection())
}

func loadOlderCopyPicker(ctx context.Context, r Reader, id session.ID, p *Picker, delta int) (*Picker, error) {
	page, err := loadCopyPage(ctx, r, id, p.CopyBefore)
	if err != nil {
		return nil, err
	}
	n := *p
	n.Messages = append(slices.Clone(p.Messages), page.messages...)
	n.Options = slices.Clone(p.Options)
	for _, choice := range page.messages {
		n.Options = append(n.Options, fmt.Sprintf("%s  %s", choice.Role, choice.Preview))
	}
	n.CopyBefore, n.CopyOlder = page.before, page.older
	n.move(delta)
	return &n, nil
}

func (p *Picker) copyAll() bool {
	return p.Checked[0] || len(p.Checked) == 0 && p.Selection == 0
}

func (p *Picker) toggleCopy() {
	if p.Checked[p.Selection] {
		delete(p.Checked, p.Selection)
		return
	}
	if p.Selection == 0 {
		clear(p.Checked)
	} else {
		delete(p.Checked, 0)
	}
	p.Checked[p.Selection] = true
}

func (p *Picker) copySelection() []MessageChoice {
	if p.copyAll() {
		return p.Messages
	}
	var selected []MessageChoice
	for i, choice := range p.Messages {
		if p.Checked[i+1] || len(p.Checked) == 0 && p.Selection == i+1 {
			selected = append(selected, choice)
		}
	}
	return selected
}
