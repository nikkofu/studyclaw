package agentic

const GoogleDesignPatternGuideURL = "https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system"

type PatternSelection struct {
	Primary    string   `json:"primary"`
	Supporting []string `json:"supporting,omitempty"`
	Why        string   `json:"why"`
	Reference  string   `json:"reference,omitempty"`
}
