# Distribution

The supported kit target is Linux amd64 with CGO/Tree-sitter enabled. Claude and
Codex share one common kit; external CLIs retain authentication ownership.

## Local checks and CI

Run `make prepush` before committing; `make prepush BASE=<rev>` also includes
committed changes since that base. It checks formatting, vet, affected Go packages
and transitive importers, and Python script tests. Non-Go inputs select all Go
tests without race. Harbor changes add its checks/unit tests and require the
`uv` version pinned in [ci.yml](../.github/workflows/ci.yml).

Normal CI runs `make test check build` and Python tests on Ubuntu 24.04. It runs
the full Go race suite. A separate Harbor job checks, builds, and smokes the
adapter, then builds and checks the Docker runner image. Scheduled/manual
[fuzzing](../.github/workflows/fuzz.yml) is separate; normal tests still exercise
saved seed corpora. A skipped subscription-dependent test is not real inference
evidence.

## Reproduce a snapshot and common kit

Use Go from [go.mod](../go.mod), Python 3.12+, a C compiler, and GoReleaser 2.18.2.
Run from the repository root on Linux amd64:

1. Check the snapshot scripts and install the pinned build tool into a new
   temporary directory:

   ```bash
   make test-snapshot
   unreal_release_tools=$(mktemp -d /tmp/unreal-release-tools.XXXXXXXX)
   GOBIN="$unreal_release_tools" go install -ldflags='-X main.version=2.18.2' github.com/goreleaser/goreleaser/v2@v2.18.2
   "$unreal_release_tools/goreleaser" --version
   ```

2. Build a snapshot:

   ```bash
   make snapshot GORELEASER="$unreal_release_tools/goreleaser"
   ```

3. Package the verified archive. Enter the full archive path printed by
   `snapshot verified:` and its full source revision from `SNAPSHOT.json`:

   ```bash
   read -r -e -p 'Verified snapshot archive: ' unreal_snapshot
   read -r -p 'Full source revision: ' unreal_revision
   python3 scripts/common_development_bundle.py --snapshot "$unreal_snapshot" --revision "$unreal_revision"
   ```

The [snapshot wrapper](../scripts/snapshot.py) reserves a fresh private
`dist/snapshots/snapshot-*/` directory. It runs `goreleaser check` and
`goreleaser release --snapshot --skip=publish` with a private config copy that
changes only the output path. It does not overwrite earlier snapshots.
[.goreleaser.yaml](../.goreleaser.yaml) disables publishing and its pre-build hook
rejects non-snapshot runs.

The snapshot is built from the current worktree, including uncommitted sources.
`SNAPSHOT.json` records revision, dirty state, target, and build metadata. The
wrapper checks archive inventory, executable permissions, checksums, CGO/VCS
metadata, help commands, and packaged Host attach/detach fixtures. These checks
use synthetic data and do not send real model requests.

The [common kit builder](../scripts/common_development_bundle.py) uses that
snapshot and adds installation, wrapper, doctor, and support scripts. Output is:

- `unreal-agent-linux-amd64-REV.tar.gz`
- `unreal-agent-linux-amd64-REV.tar.gz.sha256`

`REV` is the first seven characters of the source revision. The kit also contains
`KIT.json`, `SNAPSHOT.json`, internal `SHA256SUMS`, its README, and license.

## Executables and installed behavior

| Executable | Responsibility |
| --- | --- |
| `unreal` | Native create/attach/resume launcher in the snapshot; the installed common-kit command wraps it |
| `unreal-agent` | Host `serve`/`attach` and the subagent command interface |
| `unreal-agent-runner` | Headless JSON request / JSONL session runner for development/evaluation |
| `unreal-agent-auth` | Dedicated authentication control CLI |

The snapshot builds `unreal` and `unreal-agent` as regular binaries from the same
main package. Keep both names. The common installer publishes wrapper commands
in `~/.local/bin/` and uses per-workspace config/state. Native discovery checks
auth and catalogs without inference. Fresh setup defaults to automatic Codex
routing and Claude enabled; updates preserve explicit choices.

On the next normal workspace launch after a changed build/provider profile,
the wrapper restarts the same instance's Host and retains its Sessions/settings.
It does not create a new Session just because the build changed. Availability
changes alone are not profile changes. Explicit `--instance`/`--config` remain
operator destinations. Installed version pruning keeps current, previous, and
live-process versions. See [Install](install.md) and [Architecture](architecture.md).

## Release workflow

The [release workflow](../.github/workflows/release.yml) runs for `v*` tags in
`TomyFovos/unreal-agent`. It checks that the tagged commit is an ancestor of
`main` and waits for that exact commit's CI to succeed. It installs the pinned
GoReleaser, builds the snapshot/common kit, verifies its checksum, and passes
the archive and checksum to the publish job.

The build job has read-only contents/actions access. Only the publish job has
contents write access. It creates a GitHub Release with the common archive and
checksum. Tags containing `-` are prereleases; tags without it are regular
releases. The workflow does not publish a Docker Hub image or a runner-only kit.

Push, PR, merge, tag, and Release actions need prior user approval for the
specific commit. A successful local build or test does not grant that approval.

## Before distribution

1. Confirm local checks and the exact revision's CI result. Distinguish skipped
   login-dependent tests from passed tests.
2. Verify the common kit on WSL with a separate install destination: install it,
   start the terminal, and complete a real subscription model call. Preserve
   external login ownership and real Session data.
3. Exercise the affected user behavior, including reconnect/resume and retained
   permissions when those paths changed. For file changes, inspect actual bytes
   and canonical outcomes.
4. Report the tested kit/revision, environment, real inference result, and any
   untested route. Only then hand out the kit with [Install](install.md).

Synthetic fixtures and `doctor.sh` check packaging and local behavior. They do
not establish live provider availability. Building a kit alone is not a completed
install/start/inference acceptance check.
