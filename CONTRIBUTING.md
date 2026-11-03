# Contributing

This repository is a fork of
[unreallabsai/unreal-agent](https://github.com/unreallabsai/unreal-agent),
maintained for personal day-to-day development with Claude and Codex.

GitHub issues with questions, feature requests, and bug reports are welcome.
Pull requests are not reviewed at this time.

## Development environment

Use the Go version in [go.mod](go.mod), Python 3, Git, and a C compiler. CI uses
Python 3.12. AST tools need CGO and a working native compiler; Linux/WSL builds
can use GCC. macOS source builds need the Xcode Command Line Tools. The
[distribution](docs/distribution.md) target is Linux amd64.

1. From the repository root, build with `make build`. The executables go into
   `bin/`; `bin/unreal` is the source-built launcher.
2. Start with a focused test for the package you changed. Keep assertions,
   cancellation, and failure semantics intact. A skipped test is not a pass.
3. Before committing, run `make prepush`. Use `make prepush BASE=<rev>` when
   the check must include committed changes since a chosen base revision.

`make prepush` checks Go formatting and vet, selects changed Go packages and
transitive importers (including test imports), and runs Python script tests.
Non-Go inputs select the full Go suite without race. Harbor changes also run
its checks and unit tests; they require the `uv` version pinned in
[CI](.github/workflows/ci.yml).

CI runs `make test check build` on pushes to `main`, pull requests, and manual
dispatch. This includes the full Go race suite. Separate jobs cover Harbor,
Docker smoke, and distribution scripts. Local focused checks and `prepush` do
not replace CI's full race run. For a local milestone requiring the full race
suite with limited concurrency, use:

```bash
go test -race -count=1 -p 1 ./...
```

Keep Host/Session ownership, canonical Operations and receipts, Permission,
Registry/Translator, provider runtime, Context Engine, Project Instructions,
TUI/Orchestration, export, and LSP/DAP boundaries intact. Add only behavior the
requested change needs. Do not add provider or API-key fallback.

See [Architecture](docs/architecture.md), [Context](docs/context.md),
[Tools](docs/tools.md), and [Distribution](docs/distribution.md). The headless
[runner](cmd/unreal-agent-runner/README.md) and
[Harbor adapter](benchmarks/harbor/README.md) are development/evaluation tools.
