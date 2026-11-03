package operation_test

import (
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/operation"
)

type testRemoteJobHandler struct {
	adds          chan operation.Operation
	cancellations chan testRemoteJobCancellation
	updates       chan operation.Operation
	planType      operation.RemoteJobPlanType
	planVersion   operation.RemoteJobPlanVersion
	addErr        error
	cancelErr     error
}

type testRemoteJobCancellation struct {
	id     operation.ID
	reason string
}

func (handler *testRemoteJobHandler) RemoteJobPlanType() operation.RemoteJobPlanType {
	return handler.planType
}

func (handler *testRemoteJobHandler) RemoteJobPlanVersion() operation.RemoteJobPlanVersion {
	return handler.planVersion
}

func (handler *testRemoteJobHandler) AddRemoteJob(current operation.Operation) error {
	if handler.addErr != nil {
		return handler.addErr
	}
	handler.adds <- current
	return nil
}

func (handler *testRemoteJobHandler) CancelRemoteJob(id operation.ID, reason string) error {
	if handler.cancelErr != nil {
		return handler.cancelErr
	}
	handler.cancellations <- testRemoteJobCancellation{id: id, reason: reason}
	return nil
}

func (handler *testRemoteJobHandler) RemoteJobUpdates() <-chan operation.Operation {
	return handler.updates
}

func TestLocalOperationManagerRoutesRemoteJobs(t *testing.T) {
	handler := newTestRemoteJobHandler()
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "remote-job")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	if added := <-handler.adds; added.ID != current.ID {
		t.Fatalf("added operation ID = %q", added.ID)
	}

	state, err := operation.DecodeRemoteJobState(current)
	if err != nil {
		t.Fatal(err)
	}
	state.TerminalResult = `{"content":[]}`
	completed, err := operation.UpdateRemoteJob(current, state, operation.StatusCompleted)
	if err != nil {
		t.Fatal(err)
	}
	handler.updates <- *completed.Operation
	if update := receiveRemoteJobUpdate(t, manager.Updates()); update.Status != operation.StatusCompleted {
		t.Fatalf("status = %q", update.Status)
	}

	second := newTestRemoteJobOperation(t, "remote-cancel")
	if err := manager.Add(second); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	if err := manager.Cancel(second.ID, "requested"); err != nil {
		t.Fatal(err)
	}
	if cancellation := <-handler.cancellations; cancellation.id != second.ID || cancellation.reason != "requested" {
		t.Fatalf("cancellation = %#v", cancellation)
	}
}

func TestLocalOperationManagerRejectsInvalidRemoteJobUpdates(t *testing.T) {
	handler := newTestRemoteJobHandler()
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "remote-invalid")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	invalid := current
	invalid.Status = operation.StatusAwaiting
	invalid.State = jsontext.Value(`{`)
	handler.updates <- invalid

	failed := receiveRemoteJobUpdate(t, manager.Updates())
	state, err := operation.DecodeRemoteJobState(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != operation.StatusFailed ||
		!strings.Contains(state.TerminalError, "validate remote job handler update") {
		t.Fatalf("operation = %#v, state = %#v", failed, state)
	}
	if cancellation := <-handler.cancellations; cancellation.id != current.ID {
		t.Fatalf("cancellation = %#v", cancellation)
	}
}

func TestLocalOperationManagerReportsRemoteStartFailure(t *testing.T) {
	for _, scenario := range []string{"unsupported plan", "ambiguous plan", "add error"} {
		t.Run(scenario, func(t *testing.T) {
			handler := newTestRemoteJobHandler()
			var handlers []operation.RemoteJobHandler
			want := "does not support remote job plan"
			switch scenario {
			case "ambiguous plan":
				handlers = []operation.RemoteJobHandler{handler, newTestRemoteJobHandler()}
				want = "multiple handlers"
			case "add error":
				handler.addErr = errors.New("cannot add")
				handlers = []operation.RemoteJobHandler{handler}
				want = handler.addErr.Error()
			}
			manager := operation.NewLocalOperationManager(t.Context(), handlers...)
			current := newTestRemoteJobOperation(t, "start-error")
			if err := manager.Add(current); err != nil {
				t.Fatalf("operation failure escaped Add: %v", err)
			}
			failed := receiveRemoteJobUpdate(t, manager.Updates())
			state, err := operation.DecodeRemoteJobState(failed)
			if err != nil || failed.ID != current.ID || failed.Status != operation.StatusFailed || !strings.Contains(state.TerminalError, want) {
				t.Fatalf("failure update = %#v, state = %#v, error = %v", failed, state, err)
			}
			if err := manager.Add(current); err != nil {
				t.Fatal(err)
			}
			assertManagerRunsValueAfterFailure(t, manager)
		})
	}
}

func TestLocalOperationManagerIsolatesRemoteCancellationError(t *testing.T) {
	handler := newTestRemoteJobHandler()
	handler.cancelErr = errors.New("cannot cancel")
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "cancel-error")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	// Preserve the latest remote checkpoint when cancellation cannot be confirmed.
	state, err := operation.DecodeRemoteJobState(current)
	if err != nil {
		t.Fatal(err)
	}
	state.Handle = jsontext.Value(`{"job":"remote-handle"}`)
	awaiting, err := operation.UpdateRemoteJob(current, state, operation.StatusAwaiting)
	if err != nil {
		t.Fatal(err)
	}
	handler.updates <- *awaiting.Operation
	if update := receiveRemoteJobUpdate(t, manager.Updates()); update.Status != operation.StatusAwaiting {
		t.Fatalf("awaiting update = %#v", update)
	}
	if err := manager.Cancel(current.ID, "requested"); err != nil {
		t.Fatalf("operation failure escaped Cancel: %v", err)
	}
	failed := receiveRemoteJobUpdate(t, manager.Updates())
	failedState, err := operation.DecodeRemoteJobState(failed)
	if err != nil || failed.ID != current.ID || failed.Status != operation.StatusFailed || string(failedState.Handle) != string(state.Handle) || !strings.Contains(failedState.TerminalError, "cannot cancel") || !strings.Contains(failedState.TerminalError, "outcome is unknown") {
		t.Fatalf("failure update = %#v, state = %#v, error = %v", failed, failedState, err)
	}
	if err := manager.Cancel(current.ID, "repeat"); err != nil {
		t.Fatal(err)
	}
	// Late progress cannot revive a locally terminal operation.
	handler.updates <- *awaiting.Operation
	assertManagerRunsValueAfterFailure(t, manager)
}

func assertManagerRunsValueAfterFailure(t *testing.T, manager *operation.LocalOperationManager) {
	t.Helper()
	spec, err := operation.NewValueSpec(jsontext.Value(`"sibling result"`))
	if err != nil {
		t.Fatal(err)
	}
	current := operation.Operation{ID: "sibling", Type: spec.Type, Version: spec.Version, Status: operation.StatusReady, State: spec.State}
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	if update := receiveRemoteJobUpdate(t, manager.Updates()); update.ID != current.ID || update.Status != operation.StatusCompleted {
		t.Fatalf("sibling completion = %#v", update)
	}
}

func TestLocalOperationManagerFailsJobsWhenRemoteHandlerStops(t *testing.T) {
	handler := newTestRemoteJobHandler()
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "handler-stopped")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	close(handler.updates)

	failed := receiveRemoteJobUpdate(t, manager.Updates())
	state, err := operation.DecodeRemoteJobState(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != operation.StatusFailed ||
		!strings.Contains(state.TerminalError, "remote job handler 0 stopped") {
		t.Fatalf("operation = %#v, state = %#v", failed, state)
	}
	later := newTestRemoteJobOperation(t, "handler-already-stopped")
	if err := manager.Add(later); err != nil {
		t.Fatalf("operation failure escaped Add: %v", err)
	}
	laterFailure := receiveRemoteJobUpdate(t, manager.Updates())
	laterState, err := operation.DecodeRemoteJobState(laterFailure)
	if err != nil || laterFailure.ID != later.ID || laterFailure.Status != operation.StatusFailed || !strings.Contains(laterState.TerminalError, "remote job handler 0 stopped") {
		t.Fatalf("later failure = %#v, state = %#v, error = %v", laterFailure, laterState, err)
	}
	select {
	case added := <-handler.adds:
		t.Fatalf("stopped handler accepted operation %#v", added)
	default:
	}
}

func TestLocalOperationManagerIsolatesStoppedRemoteHandler(t *testing.T) {
	first := newTestRemoteJobHandler()
	second := newTestRemoteJobHandler()
	second.planType = "other"
	manager := operation.NewLocalOperationManager(t.Context(), first, second)
	firstJob := newTestRemoteJobOperation(t, "first-handler")
	secondJob := newTestRemoteJobOperationForPlan(t, "second-handler", "other")
	if err := manager.Add(firstJob); err != nil {
		t.Fatal(err)
	}
	<-first.adds
	if err := manager.Add(secondJob); err != nil {
		t.Fatal(err)
	}
	<-second.adds
	close(first.updates)
	if failed := receiveRemoteJobUpdate(t, manager.Updates()); failed.ID != firstJob.ID || failed.Status != operation.StatusFailed {
		t.Fatalf("failed update = %#v", failed)
	}
	state, err := operation.DecodeRemoteJobState(secondJob)
	if err != nil {
		t.Fatal(err)
	}
	state.TerminalResult = `{}`
	completed, err := operation.UpdateRemoteJob(secondJob, state, operation.StatusCompleted)
	if err != nil {
		t.Fatal(err)
	}
	second.updates <- *completed.Operation
	if update := receiveRemoteJobUpdate(t, manager.Updates()); update.ID != secondJob.ID || update.Status != operation.StatusCompleted {
		t.Fatalf("completed update = %#v", update)
	}
}

func TestLocalOperationManagerRejectsUpdateFromWrongRemoteHandler(t *testing.T) {
	first := newTestRemoteJobHandler()
	second := newTestRemoteJobHandler()
	second.planType = "other"
	manager := operation.NewLocalOperationManager(t.Context(), first, second)
	current := newTestRemoteJobOperation(t, "wrong-handler")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-first.adds
	state, err := operation.DecodeRemoteJobState(current)
	if err != nil {
		t.Fatal(err)
	}
	awaiting, err := operation.UpdateRemoteJob(current, state, operation.StatusAwaiting)
	if err != nil {
		t.Fatal(err)
	}
	second.updates <- *awaiting.Operation

	failed := receiveRemoteJobUpdate(t, manager.Updates())
	failedState, err := operation.DecodeRemoteJobState(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != operation.StatusFailed ||
		!strings.Contains(failedState.TerminalError, "wrong handler") {
		t.Fatalf("operation = %#v, state = %#v", failed, failedState)
	}
	if cancellation := <-first.cancellations; cancellation.id != current.ID {
		t.Fatalf("cancellation = %#v", cancellation)
	}
}

func TestLocalOperationManagerValidatesRemoteJobUpdateIdentity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*operation.Operation)
		want   string
	}{
		{
			name: "output limit", want: "changed output limit",
			mutate: func(update *operation.Operation) { update.MaxOutputLength++ },
		},
		{
			name: "type", want: "type or version",
			mutate: func(update *operation.Operation) { update.Type = operation.TypeShell },
		},
		{
			name: "version", want: "type or version",
			mutate: func(update *operation.Operation) { update.Version++ },
		},
		{
			name: "status", want: "invalid status",
			mutate: func(update *operation.Operation) { update.Status = "unknown" },
		},
		{
			name: "plan", want: "changed plan",
			mutate: func(update *operation.Operation) {
				state, err := operation.DecodeRemoteJobState(*update)
				if err != nil {
					t.Fatal(err)
				}
				state.Plan.Data = jsontext.Value(`{"changed":true}`)
				step, err := operation.UpdateRemoteJob(*update, state, operation.StatusAwaiting)
				if err != nil {
					t.Fatal(err)
				}
				*update = *step.Operation
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestRemoteJobHandler()
			manager := operation.NewLocalOperationManager(t.Context(), handler)
			current := newTestRemoteJobOperation(t, operation.ID("invalid-"+test.name))
			if err := manager.Add(current); err != nil {
				t.Fatal(err)
			}
			<-handler.adds
			update := current
			update.Status = operation.StatusAwaiting
			test.mutate(&update)
			handler.updates <- update

			failed := receiveRemoteJobUpdate(t, manager.Updates())
			state, err := operation.DecodeRemoteJobState(failed)
			if err != nil {
				t.Fatal(err)
			}
			if failed.Status != operation.StatusFailed || failed.MaxOutputLength != current.MaxOutputLength ||
				!strings.Contains(state.TerminalError, test.want) {
				t.Fatalf("operation = %#v, state = %#v", failed, state)
			}
			if cancellation := <-handler.cancellations; cancellation.id != current.ID {
				t.Fatalf("cancellation = %#v", cancellation)
			}
		})
	}
}

func TestLocalOperationManagerRejectsRemoteJobStatusRegression(t *testing.T) {
	handler := newTestRemoteJobHandler()
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "status-regression")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	state, err := operation.DecodeRemoteJobState(current)
	if err != nil {
		t.Fatal(err)
	}
	awaiting, err := operation.UpdateRemoteJob(current, state, operation.StatusAwaiting)
	if err != nil {
		t.Fatal(err)
	}
	handler.updates <- *awaiting.Operation
	if update := receiveRemoteJobUpdate(t, manager.Updates()); update.Status != operation.StatusAwaiting {
		t.Fatalf("status = %q", update.Status)
	}
	regressed, err := operation.UpdateRemoteJob(*awaiting.Operation, state, operation.StatusReady)
	if err != nil {
		t.Fatal(err)
	}
	handler.updates <- *regressed.Operation
	failed := receiveRemoteJobUpdate(t, manager.Updates())
	failedState, err := operation.DecodeRemoteJobState(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != operation.StatusFailed ||
		!strings.Contains(failedState.TerminalError, "invalid status transition") {
		t.Fatalf("operation = %#v, state = %#v", failed, failedState)
	}
	if cancellation := <-handler.cancellations; cancellation.id != current.ID {
		t.Fatalf("cancellation = %#v", cancellation)
	}
}

func TestLocalOperationManagerReportsRemoteCancellationFailureAfterInvalidUpdate(t *testing.T) {
	handler := newTestRemoteJobHandler()
	handler.cancelErr = errors.New("cannot cancel")
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "invalid-cancel-error")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	invalid := current
	invalid.Status = operation.StatusAwaiting
	invalid.State = jsontext.Value(`{`)
	handler.updates <- invalid

	failed := receiveRemoteJobUpdate(t, manager.Updates())
	state, err := operation.DecodeRemoteJobState(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != operation.StatusFailed ||
		!strings.Contains(state.TerminalError, "validate remote job handler update") ||
		!strings.Contains(state.TerminalError, "cannot cancel") {
		t.Fatalf("operation = %#v, state = %#v", failed, state)
	}
}

func TestLocalOperationManagerAddsRemoteJobOnce(t *testing.T) {
	handler := newTestRemoteJobHandler()
	manager := operation.NewLocalOperationManager(t.Context(), handler)
	current := newTestRemoteJobOperation(t, "idempotent")
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	<-handler.adds
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	select {
	case duplicate := <-handler.adds:
		t.Fatalf("duplicate add = %#v", duplicate)
	default:
	}
}

func newTestRemoteJobHandler() *testRemoteJobHandler {
	return &testRemoteJobHandler{
		adds:          make(chan operation.Operation, 1),
		cancellations: make(chan testRemoteJobCancellation, 1),
		updates:       make(chan operation.Operation, 1),
		planType:      "test",
		planVersion:   1,
	}
}

func newTestRemoteJobOperation(t *testing.T, id operation.ID) operation.Operation {
	t.Helper()
	return newTestRemoteJobOperationForPlan(t, id, "test")
}

func newTestRemoteJobOperationForPlan(
	t *testing.T,
	id operation.ID,
	planType operation.RemoteJobPlanType,
) operation.Operation {
	t.Helper()
	spec, err := operation.NewRemoteJobSpec(operation.RemoteJobPlan{
		Type: planType, Version: 1, Data: jsontext.Value(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return operation.Operation{
		ID: id, Type: spec.Type, Version: spec.Version, Status: operation.StatusReady, MaxOutputLength: spec.MaxOutputLength, State: spec.State,
	}
}

func receiveRemoteJobUpdate(t *testing.T, updates <-chan operation.Operation) operation.Operation {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case update := <-updates:
		return update
	case <-timer.C:
		t.Fatal("timed out waiting for remote job update")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	return operation.Operation{}
}
