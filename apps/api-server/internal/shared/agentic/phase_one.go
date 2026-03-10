package agentic

const (
	PatternCustomLogic = "custom logic pattern"
	PatternSingleAgent = "single-agent system"
	PatternHumanInLoop = "human-in-the-loop pattern"
	PhaseOneBoundedLLM = "bounded llm summarization"
)

var PhaseOneTaskParsePattern = PatternSelection{
	Primary:    PatternCustomLogic,
	Supporting: []string{PatternSingleAgent, PatternHumanInLoop},
	Why:        "Phase one homework parsing is mostly deterministic normalization with one bounded semantic parsing call and explicit parent review before publishing.",
	Reference:  GoogleDesignPatternGuideURL,
}

var PhaseOneInsightsPattern = PatternSelection{
	Primary:    PatternCustomLogic,
	Supporting: []string{PatternSingleAgent},
	Why:        "Phase one daily, weekly, and monthly encouragement reports must use deterministic metrics first, then one bounded summarization call or template fallback.",
	Reference:  GoogleDesignPatternGuideURL,
}
