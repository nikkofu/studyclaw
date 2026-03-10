## Go-Agent Group

You are working inside the Go agent lane.

### Owned scope
- This directory and its children
- `../../platform/llm/`
- `../../shared/agentic/`
- `docs/04_AGENTIC_DESIGN.md` when agentic design needs updating

### Goal
- Improve task parsing and phase-one report quality
- Keep LLM usage bounded and explicit
- Follow Google agentic design patterns strictly

### Do not do
- Do not modify `taskboard` persistence logic
- Do not redesign HTTP response shapes without coordinating with the Go-API lane
- Do not edit Flutter or Parent Web source

### Default policy
- Prefer deterministic logic first
- Add tests for every parser or insight behavior change
- Preserve response field semantics unless there is a deliberate contract change
