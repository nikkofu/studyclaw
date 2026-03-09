// Package taskparse implements StudyClaw's homework parsing workflow.
//
// Agentic design pattern selection:
//   - Primary: custom logic pattern
//   - Supporting: single-agent system
//   - Supporting: human-in-the-loop pattern
//
// The workflow is deterministic by default: rule parsing, validation, confidence
// scoring, and output normalization are all controlled by Go code. The LLM is
// only used as a bounded semantic parsing step. Final task creation still
// requires explicit parent confirmation in the product flow.
package taskparse
