package coordinator

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

type cancelFailureRemoteHandler struct{ updates chan operation.Operation }

func (*cancelFailureRemoteHandler) RemoteJobPlanType() operation.RemoteJobPlanType {
	return "cancel-failure"
}
func (*cancelFailureRemoteHandler) RemoteJobPlanVersion() operation.RemoteJobPlanVersion { return 1 }
func (*cancelFailureRemoteHandler) AddRemoteJob(operation.Operation) error               { return nil }
func (*cancelFailureRemoteHandler) CancelRemoteJob(operation.ID, string) error {
	return errors.New("remote cancel unavailable")
}
func (handler *cancelFailureRemoteHandler) RemoteJobUpdates() <-chan operation.Operation {
	return handler.updates
}

func TestCoordinatorRemoteCancelFailureKeepsSessionAvailable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		run := newStopTestRun(t, 1)
		spec, err := operation.NewRemoteJobSpec(operation.RemoteJobPlan{
			Type: "cancel-failure", Version: 1, Data: jsontext.Value(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		op := run.store.resume.Operations[0]
		op.Type, op.Version, op.State = spec.Type, spec.Version, spec.State
		op.Status, op.MaxOutputLength = operation.StatusReady, spec.MaxOutputLength
		run.store.resume.Operations[0] = op
		status := run.store.items[2].Data.(sessionstore.ToolCallStatus)
		status.Operations[0] = op
		run.store.items[2].Data = status
		handler := &cancelFailureRemoteHandler{updates: make(chan operation.Operation)}
		run.current.dependencies.Operations = operation.NewLocalOperationManager(t.Context(), handler)
		run.start(t)
		run.input(t, stopInput(t, "interrupt", inbox.InterruptAgent))
		run.assertRunning(t)
		if len(run.calls) != 0 || !run.current.held() {
			t.Fatal("remote cancel failure released the user interruption")
		}
		last := run.store.savedOperations[len(run.store.savedOperations)-1]
		state, err := operation.DecodeRemoteJobState(last)
		if err != nil || last.Status != operation.StatusCanceled || !strings.Contains(state.TerminalError, "remote cancel unavailable") || !strings.Contains(state.TerminalError, "outcome is unknown") {
			t.Fatalf("cancellation lost its receipt: operation=%#v state=%#v error=%v", last, state, err)
		}
		run.input(t, externalEvent(t, 1, "continue", "continue working"))
		if len(run.calls) != 1 {
			t.Fatal("remote cancel failure stopped the Session")
		}
		assertStopResult(t, run.calls[0].request, "call-0", string(operation.StatusCanceled))
		run.respond(t, 0, textResponse("Still available."))
		run.input(t, stopInput(t, "stop", inbox.StopWhenIdle))
		run.assertStopped(t)
	})
}

func TestCoordinatorStopCancelsShellProcess(t *testing.T) {
	store, registry := independentToolCalls(t, 1)
	spec, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh", Command: "exec sleep 30"}, t.TempDir(), 64)
	if err != nil {
		t.Fatal(err)
	}
	value := store.resume.Operations[0]
	value.Status, value.State = operation.StatusReady, spec.State
	value.Type, value.Version = spec.Type, spec.Version
	value.MaxOutputLength = spec.MaxOutputLength
	store.resume.Operations[0] = value
	status := store.items[2].Data.(sessionstore.ToolCallStatus)
	status.Operations[0] = value
	store.items[2].Data = status
	inputs := newTestInbox(t)
	stop := stopInput(t, "stop", inbox.StopHard)
	var processGroupID int
	store.onSaveOperation = func(update operation.Operation) {
		var state operation.ShellState
		if err := json.Unmarshal(update.State, &state); err != nil {
			store.saveOperationErr = err
			return
		}
		if processGroupID == 0 && state.ProcessGroupID > 1 {
			processGroupID = state.ProcessGroupID
			submitTestInput(t, inputs, stop)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	adapter := &fakeAdapter{respond: func(context.Context, llm.Request) (llm.Response, error) {
		return textResponse("Stopped."), nil
	}}
	current := New(Dependencies{
		SessionID: "session-1", Inbox: inputs, Restored: store.resume, Sessions: store,
		ContextBuilder: contextbuilder.NewBuilder(), LLM: adapter, Tools: registry,
		Operations: operation.NewLocalOperationManager(ctx),
	})
	c := current.(*coordinator)
	if err := c.loadHistory(ctx); err != nil {
		t.Fatal(err)
	}
	for _, op := range store.resume.Operations {
		c.addOperationToLocalState(op)
	}
	if err := c.runLoop(ctx); err != nil {
		t.Fatal(err)
	}
	if processGroupID <= 1 {
		t.Fatal("shell process never started")
	}
	if err := syscall.Kill(-processGroupID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("process group %d after stop: %v", processGroupID, err)
	}
	last := store.appendedStatuses[len(store.appendedStatuses)-1]
	if last.Operations[0].Status != operation.StatusCanceled {
		t.Fatalf("persisted operation status = %s", last.Operations[0].Status)
	}
	requests := adapter.requestSnapshot()
	if len(requests) != 0 {
		t.Fatal("hard stop requested a model response")
	}
}

func TestCoordinatorStopAfterToolCommitSettlesOperation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		run := newToolGraceTestRun(t)
		store := persistTestRun(t, run)
		restoreTestRun(t, run, store)
		run.current.dependencies.Operations = operation.NewLocalOperationManager(t.Context())
		stopQueued := false
		observer := store.AddObserver(func(_ session.ID, item sessionstore.Item) {
			if item.Kind != sessionstore.ItemToolCallStatus || stopQueued {
				return
			}
			status := item.Data.(sessionstore.ToolCallStatus)
			if len(status.Operations) == 0 {
				return
			}
			stopQueued = true
			submitTestInput(t, run.inputs, stopInput(t, "stop", inbox.StopHard))
		})
		defer store.RemoveObserver(observer)
		run.start(t)
		run.input(t, externalEvent(t, 0, "input", "run tool"))
		run.respond(t, 0, toolGraceResponse("A"))
		run.assertStopped(t)
		if !stopQueued || len(run.calls) != 1 {
			t.Fatal("stop was not queued at the commit boundary or started another turn")
		}
		page, err := store.Items(t.Context(), "session-1", sessionstore.BeforeFirst, 100)
		if err != nil {
			t.Fatal(err)
		}
		last := page.Items[len(page.Items)-1]
		if last.Kind != sessionstore.ItemToolCallStatus {
			t.Fatalf("last item = %s, want settled tool call", last.Kind)
		}
		status := last.Data.(sessionstore.ToolCallStatus)
		if status.CallID != "A" || len(status.Operations) != 1 || status.Operations[0].Status != operation.StatusCompleted {
			t.Fatalf("persisted tool result = %#v, want completed value operation", status)
		}
		restored, err := store.Resume(t.Context(), "session-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(restored.Operations) != 0 {
			t.Fatal("stop left an operation unsettled in the session store")
		}
	})
}
