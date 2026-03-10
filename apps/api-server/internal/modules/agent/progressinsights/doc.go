// Package progressinsights implements StudyClaw's phase-one daily, weekly, and
// monthly encouragement reports.
//
// Agentic design pattern selection:
//   - Primary: custom logic pattern
//   - Supporting: single-agent system
//
// The workflow is deterministic first: task totals, completion rates, active
// days, points changes, and subject breakdowns come from precomputed stats. The
// LLM is only used for one bounded summarization step and must never rewrite the
// metric values.
package progressinsights
