// Package weeklyinsights implements the compatibility wrapper for the weekly
// encouragement report workflow.
//
// Agentic design pattern selection:
//   - Primary: custom logic pattern
//   - Supporting: single-agent system
//
// The weekly service delegates to the shared phase-one report generator:
// deterministic metrics are aggregated first, then one bounded LLM call or
// template fallback produces a supportive weekly summary. The model never owns
// statistics or task state.
package weeklyinsights
