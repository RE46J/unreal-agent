package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

type readyInput struct {
	io.Reader
	notified bool
}

func (input *readyInput) Read(buffer []byte) (int, error) {
	if !input.notified {
		input.notified = true
		fmt.Fprintln(os.Stdout, "waiting for request")
	}
	return input.Reader.Read(buffer)
}

func TestRunnerSignalHelperProcess(t *testing.T) {
	if os.Getenv("UNREAL_RUNNER_SIGNAL_HELPER") != "1" {
		return
	}
	os.Args = []string{"unreal-agent-runner"}
	os.Exit(run(&readyInput{Reader: os.Stdin}))
}

func TestRunnerSignalsCancelInputWait(t *testing.T) {
	for _, signal := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(signal.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunnerSignalHelperProcess$")
			command.Env = append(os.Environ(), "UNREAL_RUNNER_SIGNAL_HELPER=1")
			input, err := command.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			output, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			command.Stderr = &stderr
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { command.Process.Kill() })
			ready, err := bufio.NewReader(output).ReadString('\n')
			if err != nil || ready != "waiting for request\n" {
				t.Fatalf("runner readiness = %q, %v; stderr: %s", ready, err, stderr.String())
			}
			if err := command.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			// Keep stdin open: signal handling must cancel the parse wait.
			err = command.Wait()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 130 {
				t.Fatalf("runner exit = %v, want 130; stderr: %s", err, stderr.String())
			}
		})
	}
}
