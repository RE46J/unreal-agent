// Command unreal-agent-runner executes one JSON request and writes persisted session items as JSONL.
package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/unreallabsai/unreal-agent/cmd/internal/agentrunner"
)

func main() {
	os.Exit(run(os.Stdin))
}

func run(input io.Reader) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return agentrunner.RunMain(
		ctx, os.Args[1:], os.Getenv, os.Environ,
		input, os.Stdout, os.Stderr,
		agentrunner.Config{
			Name:         "unreal-agent-runner",
			ParseRequest: parseRequest,
			Providers:    agentrunner.DefaultProviders(),
		},
	)
}
