# Capability and Permission Enforcement (`harness/permission`)

**Language:** English | [日本語](../../docs/ja/harness/permission/README.md)

`harness/permission` enforces immutable capability policies across tools, filesystem roots, subprocesses, and HTTP requests.

---

## 1. Policy Construction and Attachment

A Host builds an immutable `permission.Policy` for each execution ownership lifetime and attaches it to the execution context:

```go
policy, err := permission.New(permission.Config{
    Tools:          []string{"ViewImage", "SkillUse"},
    ReadRoots:      []string{workspace, skillsDirectory},
    NetworkOrigins: []string{"https://api.openai.com"},
})
// Handle err. Close the policy after all executors have stopped.
ctx = permission.WithPolicy(ctx, policy)
manager := operation.NewLocalOperationManagerWithPolicy(ctx, policy)
```

- **Fail-closed defaults:** A zero `Config` or explicit `nil` policy denies all capabilities. Unrestricted execution requires an explicit `permission.Unrestricted()` policy.
- **No persistence of grants:** Persisting a tool name or Operation record never persists a permission grant; resuming a Session always builds a fresh policy from currently accepted settings.

---

## 2. Concrete Enforcement Boundaries

Permissions are checked at every execution layer:

1. **Coordinator & Operation Manager:** Checked before invoking a Translator and again before starting or resuming an Operation. Expected denials produce a typed `Denial` on a failed Operation rather than crashing the manager.
2. **Filesystem (`ReadFile` / `Create` / `OpenParent`):**
   - Read and write directory roots are independent (`O_RDWR` requires both).
   - Roots are pinned using `os.Root` file descriptors. Descriptor-relative `OpenFile` and `OpenParent` operations prevent symlink races and `..` traversal from escaping the root.
   - `CheckPath` is a preflight helper only; mutation executors re-verify authorization inside their mutation lock using pinned descriptors.
3. **Subprocesses (`StartProcess`):**
   - Checked before opening output files or spawning processes.
   - Because the harness does not implement an OS-level kernel sandbox, launching arbitrary shells, LSP/DAP servers, or sub-agents requires an explicit unrestricted filesystem, network, and process grant (restricted process policies fail closed with `unsupported`).
4. **Network (`RemoteClient` & `Policy.RoundTripper`):**
   - Guards every HTTP request and redirect against exact allowed `scheme://host[:port]` origins (wildcards and URL userinfo are rejected).
   - Denials omit sensitive URLs, paths, headers, and credentials.

---

## 3. Child Session Intersection (`WithPolicy` / `Intersect`)

Attaching a second policy to a context via `WithPolicy` (or `Policy.Intersect`) always computes the **intersection** of capabilities—a child Session or sub-component can narrow permissions, but can never widen its parent's policy.
