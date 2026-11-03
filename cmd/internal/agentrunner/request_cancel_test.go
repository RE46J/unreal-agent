package agentrunner

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/synctest"
)

func TestRunCancelsWhileParsingRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		input, writer := io.Pipe()
		defer input.Close()
		defer writer.Close()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		started, finished := make(chan struct{}), make(chan struct{})
		config := testConfig(&fakeClient{})
		parse := config.ParseRequest
		config.ParseRequest = func(input io.Reader) (Request, ToolFactory, error) {
			close(started)
			defer close(finished)
			return parse(input)
		}
		result := make(chan error, 1)
		go func() {
			result <- Run(ctx, nil, func(string) string { return "" }, func() []string { return nil }, input, io.Discard, io.Discard, config)
		}()
		<-started
		synctest.Wait()
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want cancellation", err)
		}
		// Run must not close a caller-owned reader. Let the parser finish and
		// ensure that sending its abandoned result cannot strand a goroutine.
		if _, err := io.WriteString(writer, `{"prompt":"hello"}`); err != nil {
			t.Fatal(err)
		}
		writer.Close()
		<-finished
		synctest.Wait()
	})
}

func TestParseRequestContextPreservesResult(t *testing.T) {
	for _, raw := range []string{`{"prompt":"hello"}`, `{"prompt":`} {
		t.Run(raw, func(t *testing.T) {
			want, _, wantErr := parseTestRequest(strings.NewReader(raw))
			got, factory, err := parseRequestContext(t.Context(), strings.NewReader(raw), parseTestRequest)
			if (err == nil) != (wantErr == nil) {
				t.Fatalf("parse error = %v, want %v", err, wantErr)
			}
			if err != nil && err.Error() != wantErr.Error() {
				t.Fatalf("parse error = %v, want %v", err, wantErr)
			}
			if want.Prompt != nil && (got.Prompt == nil || *got.Prompt != *want.Prompt) {
				t.Fatalf("prompt = %v, want %q", got.Prompt, *want.Prompt)
			}
			if factory == nil {
				t.Fatal("parser tool factory was lost")
			}
		})
	}
}
