# Contributing

**Language:** English | [日本語](docs/ja/CONTRIBUTING.md)

This repository is a personal-use fork of [unreallabsai/unreal-agent](https://github.com/unreallabsai/unreal-agent), maintained for day-to-day development with Claude and Codex.

- **Issues:** Questions, bug reports, and feature requests are welcome via GitHub Issues.
- **Pull Requests:** External pull requests are not reviewed at this time.

## Prerequisites

- **Go:** Version specified in [`go.mod`](go.mod)
- **C Compiler (CGO):** Required for Tree-sitter AST tools (`gcc` on Linux/WSL, Xcode Command Line Tools on macOS)
- **Python 3:** Python 3.12+ (used by build, snapshot, and verification scripts)
- **Git**

The official [distribution kit](docs/distribution.md) target is **Linux amd64**.

## Build and Test Workflow

### 1. Build Binaries

From the repository root:

```bash
make build
```

This compiles four binaries into `bin/`:
- `bin/unreal` (symlink to `unreal-agent`, launches or attaches to sessions)
- `bin/unreal-agent` (Host server, attach client, and sub-agent CLI)
- `bin/unreal-agent-runner` (headless evaluation runner)
- `bin/unreal-agent-auth` (credential management CLI)

### 2. Run Focused Tests During Development

Run tests for the specific packages you modify. Keep existing assertions, cancellation behavior, and failure semantics intact. Do not treat a skipped test as a pass.

### 3. Run Pre-Commit Checks (`make prepush`)

Before every commit, run:

```bash
make prepush
# Or include committed changes since a base revision:
make prepush BASE=<rev>
```

What `make prepush` runs:
1. `gofmt -l cmd harness internal` and `go vet ./...`
2. Go tests for modified packages and all transitive importers (or `./...` if non-Go files changed)
3. Python unit tests under `scripts/` (`test_*.py`)
4. Harbor adapter checks and tests if files under `benchmarks/harbor/` changed (requires the `uv` version pinned in [`.github/workflows/ci.yml`](.github/workflows/ci.yml))

### 4. Full Race Suite at Milestones

Before completing a milestone, run the full Go test suite with the race detector:

```bash
go test -race -count=1 -p 1 ./...
```

CI runs `make test check build` (including the race detector) on pushes to `main`, pull requests, and manual runs, alongside separate jobs for Harbor, Docker smoke tests, and distribution scripts.

## Architectural Rules to Preserve

When modifying the codebase, keep these core boundaries intact:
- **Host & Session ownership:** The background Host owns Session state, writer locks, and canonical history.
- **Canonical Operations & Receipts:** Translators are pure; side effects run through the Operation Manager with durable receipts.
- **Permission enforcement:** Checked at concrete execution boundaries (`permission.Policy` and interactive permission modes).
- **Provider runtime:** No automatic fallback to another provider or to an API key.
- **Context Engine & Project Instructions:** Deterministic projection and snapshot rules.
- **LSP & DAP isolation:** Language server and debugger lifecycles remain separate from inference and TUI rendering.

## Related Developer Documentation

- [Architecture](docs/architecture.md)
- [Context & Project Instructions](docs/context.md)
- [Tools & File Mutation](docs/tools.md)
- [Distribution](docs/distribution.md)
- [Headless Runner](cmd/unreal-agent-runner/README.md)
- [Harbor Benchmark Adapter](benchmarks/harbor/README.md)
