# Debug Adapter Protocol Runtime (`harness/dap`)

**Language:** English | [日本語](../../docs/ja/harness/dap/README.md)

`harness/dap` manages Debug Adapter Protocol (DAP) subprocesses and exposes structured debugging operations to the agent runtime.

Protocol specification: [Debug Adapter Protocol Overview](https://microsoft.github.io/debug-adapter-protocol/overview)

---

## 1. Host Composition & Runner Configuration

### Host Integration

1. Create `dap.NewManager` with the Host permission context and static `AdapterConfig` entries.
2. Register the manager as an `operation.RemoteJobHandler`, along with `tool/dap.Definition()` and `New(manager.Generation())` in the tool registry.
3. Wrap session tool factories via `agentrunner.DebugTools` so each Session gets an isolated debugger generation that closes cleanly when the Session ends.

### Runner Environment Variable (`UNREAL_HARNESS_DAP_ADAPTERS`)

Pass a JSON array (up to 64 KiB) of trusted adapter configurations:

```json
[{"id":"lldb","path":"/opt/bin/lldb-dap","directory":"/workspace","arguments":[],"allowed_attach_pids":[]}]
```

- **Trusted Host config vs. Model arguments:** Adapter executable paths, base arguments, `launch_fields`, `attach_fields`, and `allowed_attach_pids` come strictly from trusted Host configuration—never from model tool-call arguments.
- **Session metadata:** Only a SHA-256 digest of the configuration is persisted in Session metadata.

---

## 2. Supported Commands & Limits

### DAP Actions

`launch`, `attach`, `setBreakpoints`, `threads`, `stackTrace`, `scopes`, `variables`, `evaluate`, `continue`, `pause`, `next`, `stepIn`, `stepOut`, `inspect`, `disconnect`.

- **Epoch validation (`stop_epoch`):** Stack frame and variable references are bound to the current `stop_epoch` and are rejected once execution resumes.
- **No automatic replay on restart:** Plans include a logical session ID and the Host owner `generation`. After a Host restart, previous-generation debug plans become `expired`—debug targets are never automatically relaunched, reattached, or stepped on recovery.

### Resource Bounds

- **Request timeout:** 30 seconds per command
- **Concurrent sessions:** Up to 8 logical debug sessions per owner
- **Result size caps:** Max 100 items, 2 KiB per text field, 32 KiB structured result, 1 MiB wire frame

---

## 3. Permissions and Process Ownership

- **Permissions:** Requires explicit unrestricted process/filesystem/network permission (since no OS sandbox is built into the harness). `attach` also requires the target PID to be listed in `allowed_attach_pids`, and `evaluate` requires the `DAP.Evaluate` capability.
- **Safe disconnect behavior:**
  - **Launched targets (`launch`):** Closing or disconnecting sends `terminateDebuggee: true` and cleans up the adapter process group.
  - **Attached targets (`attach`):** Closing or disconnecting sends `terminateDebuggee: false` and never sends kill signals directly to the attached PID.
