# Harbor Evaluation Adapter

**Language:** English | [日本語](../../docs/ja/benchmarks/harbor/README.md)

This package integrates `unreal-agent-runner` with **Harbor 0.22.0** for automated benchmark evaluations (including Terminal-Bench).

## Prerequisites

- Go (version from root [`go.mod`](../../go.mod))
- Python 3.12+ and `uv`
- Docker or Modal

---

## 1. Setup and Build

From the repository root:

```sh
make -C benchmarks/harbor sync
make -C benchmarks/harbor build REVISION=HEAD
```

This compiles the committed source for Linux amd64 into `bin/harbor/<short-commit>/` alongside a revision and SHA-256 checksum manifest. Existing bundle directories are never overwritten (set `BUNDLE=<new-dir>` if you need to rebuild into a new path).

---

## 2. Running Evaluations

### Local Task Run

Pass an absolute path to your built bundle:

```sh
export OPENAI_API_KEY=...
uv run --project benchmarks/harbor --locked harbor run \
  -p /absolute/path/to/task \
  -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-5.4 \
  --ak bundle="$PWD/bin/harbor/<short-commit>" \
  --ak thinking_level=high
```

### Supported Models & Options

- **`thinking_level`:** `low`, `medium`, `high` (default), `xhigh`, or `max`
- **Providers:**
  - `openai/<model>` with `OPENAI_API_KEY`
  - `openrouter/<model>` with `OPENROUTER_API_KEY` (automatically enables 1-hour prompt caching with session affinity)
  - `fireworks_ai/<model>` with `FIREWORKS_AI_API_KEY`
- Use Harbor's `--ae` flag to pass additional environment variables.

### Running Terminal-Bench 4.0 on Modal

With Modal credentials configured:

```sh
uv run --project benchmarks/harbor --locked --extra modal harbor run \
  -d terminal-bench/terminal-bench@4.0.0 \
  -e modal -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-6-astra \
  --ak bundle="$PWD/bin/harbor/<short-commit>" --ak thinking_level=max \
  -k 5 -n 40 --job-name tb4-unreal-agent
```

View results with:

```sh
harbor view jobs/<job-name>
```

---

## 3. Output Format and Tools

- **Enabled tools:** `Bash` and `ViewImage` (returns images within 2000×2000 px and <5 MB base64, saved under `agent/images/` with format and coordinate-scaling metadata).
- **Artifacts:** Logs and session files are written under `agent/`. `agent/trajectory.json` records ATIF observations (grouped by tool call with sequence and timing metadata) and token usage totals.

---

## 4. Validation & Smoke Tests

```sh
make -C benchmarks/harbor test check
make -C benchmarks/harbor smoke BUNDLE=../../bin/harbor/<short-commit>
```

The smoke test runs inside Docker against a local test model.

Reference: [Harbor Agents Documentation](https://www.harborframework.com/docs/agents)
