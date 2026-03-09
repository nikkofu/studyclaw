// Package weeklyinsights implements the weekly insight generation workflow.
//
// Agentic design pattern selection:
//   - Primary: single-agent system
//   - Supporting: custom logic pattern
//
// The service performs deterministic data aggregation in Go, then asks one
// bounded LLM call to produce a child-friendly summary. There is no need for a
// multi-agent coordinator because the task is a single summarization problem.
package weeklyinsights
