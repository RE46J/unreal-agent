package coordinator

import (
	"strings"
	"testing"
	"testing/synctest"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

// Both an initial owner batch and an unanswered restored input can wake the
// first request without receiving anything from the event-loop select.
func startupInputRun(t *testing.T, restored bool) *stopTestRun {
	t.Helper()
	run := newStopTestRun(t, 0)
	input := externalEvent(t, 0, "task", "initial task")
	if restored {
		run.store.items = []sessionstore.Item{storedItem(1, sessionstore.ItemInput, input)}
	} else {
		run.store.items = nil
		run.current.dependencies.InitialInputs = []inbox.Input{input}
	}
	return run
}

func TestCoordinatorStartupAppliesQueuedSettingsAndInputBeforeFirstRequest(t *testing.T) {
	for _, restored := range []bool{false, true} {
		t.Run(map[bool]string{false: "initial", true: "restored"}[restored], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				run := startupInputRun(t, restored)
				submitTestInput(t, run.inputs, settingsInput(t, "settings", inbox.Settings{ReasoningEffort: llm.ReasoningEffortHigh}))
				submitTestInput(t, run.inputs, externalEvent(t, 1, "queued", "queued instruction"))
				synctest.Wait()
				run.start(t)
				if len(run.calls) != 1 || run.calls[0].request.Model.ReasoningEffort != llm.ReasoningEffortHigh {
					t.Fatal("first request did not use queued settings")
				}
				dump := requestDump(t, run.calls[0].request)
				if !strings.Contains(dump, "initial task") || !strings.Contains(dump, "queued instruction") {
					t.Fatal("first request omitted already queued input")
				}
				run.respond(t, 0, textResponse("Done."))
				if len(run.calls) != 1 {
					t.Fatal("already queued input caused an unnecessary second request")
				}
				run.input(t, stopInput(t, "stop", inbox.StopWhenIdle))
				run.assertStopped(t)
			})
		})
	}
}

func TestCoordinatorStartupAppliesQueuedInterruptAndStopBeforeFirstRequest(t *testing.T) {
	for _, restored := range []bool{false, true} {
		for _, mode := range []inbox.ControlMode{inbox.InterruptAgent, inbox.StopHard, inbox.StopWhenIdle} {
			t.Run(map[bool]string{false: "initial/", true: "restored/"}[restored]+string(mode), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					run := startupInputRun(t, restored)
					submitTestInput(t, run.inputs, stopInput(t, "queued-control", mode))
					synctest.Wait()
					run.start(t)
					if mode == inbox.StopWhenIdle {
						if len(run.calls) != 1 {
							t.Fatal("when-idle stop did not deliver pending input")
						}
						run.respond(t, 0, textResponse("Done."))
						run.assertStopped(t)
						return
					}
					if len(run.calls) != 0 {
						t.Fatal("queued interruption allowed an initial model request")
					}
					if mode == inbox.StopHard {
						run.assertStopped(t)
						return
					}
					run.assertRunning(t)
					if !run.current.held() {
						t.Fatal("queued interrupt did not hold the model")
					}
					run.input(t, externalEvent(t, 1, "continue", "continue now"))
					if len(run.calls) != 1 {
						t.Fatal("user input did not release the queued interruption")
					}
					run.respond(t, 0, textResponse("Done."))
					run.input(t, stopInput(t, "stop", inbox.StopWhenIdle))
					run.assertStopped(t)
				})
			})
		}
	}
}

func TestCoordinatorStartupAppliesQueuedOperationResultBeforeFirstRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		run := newStopTestRun(t, 1)
		run.current.dependencies.InitialInputs = []inbox.Input{externalEvent(t, 1, "progress", "check progress")}
		completed := run.store.resume.Operations[0]
		completed.Status = operation.StatusCompleted
		run.operations.updates <- completed
		run.start(t)
		if len(run.calls) != 1 {
			t.Fatal("queued result did not produce exactly one initial request")
		}
		assertStopResult(t, run.calls[0].request, "call-0", string(operation.StatusCompleted))
		run.respond(t, 0, textResponse("Done."))
		if len(run.calls) != 1 {
			t.Fatal("queued result caused an unnecessary second request")
		}
		run.input(t, stopInput(t, "stop", inbox.StopWhenIdle))
		run.assertStopped(t)
	})
}

func TestCoordinatorStartupQueuedResultKeepsFullBatchBarrier(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		run := newStopTestRun(t, 2)
		messagesUseValueOperations(t, run)
		messagesRequireBatch(run)
		run.current.dependencies.InitialInputs = []inbox.Input{externalEvent(t, 1, "progress", "check progress")}
		completed := run.store.resume.Operations[0]
		completed.Status = operation.StatusCompleted
		run.operations.updates <- completed
		run.start(t)
		if len(run.calls) != 0 {
			t.Fatal("queued partial result bypassed the batch barrier at startup")
		}
		run.update(t, 1, operation.StatusCompleted)
		if len(run.calls) != 1 {
			t.Fatal("complete batch did not release exactly one request")
		}
		for _, call := range []string{"call-0", "call-1"} {
			if result := messagesResult(t, run.calls[0].request, call); result.State != llm.ToolResultSucceeded {
				t.Fatalf("%s state = %v", call, result.State)
			}
		}
		run.respond(t, 0, textResponse("Done."))
		run.input(t, stopInput(t, "stop", inbox.StopWhenIdle))
		run.assertStopped(t)
	})
}
