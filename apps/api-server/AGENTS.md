## Backend Scope

This subtree is owned only by Go backend groups.

### Primary rule
- Work in Go only.
- Do not modify `apps/pad-app` or `apps/parent-web` from this subtree.

### Internal split
- `Go-API` lane owns:
  - `cmd/`
  - `config/`
  - `internal/app/`
  - `internal/interfaces/http/`
  - `internal/modules/taskboard/`
  - `routes/`
- `Go-Agent` lane owns:
  - `internal/modules/agent/`
  - `internal/platform/llm/`
  - `internal/shared/agentic/`

### Coordination rules
- Keep public HTTP contracts backward compatible by default.
- If a change would affect Flutter or Parent Web, update backend tests first.
- Do not edit generated caches or runtime data directories.
