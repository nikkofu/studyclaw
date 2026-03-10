package agentic

import "testing"

func TestPhaseOneTaskParsePatternMatchesPhaseOneConstraints(t *testing.T) {
	if PhaseOneTaskParsePattern.Primary != PatternCustomLogic {
		t.Fatalf("expected custom logic primary, got %+v", PhaseOneTaskParsePattern)
	}
	if len(PhaseOneTaskParsePattern.Supporting) != 2 {
		t.Fatalf("expected supporting patterns, got %+v", PhaseOneTaskParsePattern)
	}
	if PhaseOneTaskParsePattern.Reference != GoogleDesignPatternGuideURL {
		t.Fatalf("expected google guide reference, got %+v", PhaseOneTaskParsePattern)
	}
}

func TestPhaseOneInsightsPatternMatchesPhaseOneConstraints(t *testing.T) {
	if PhaseOneInsightsPattern.Primary != PatternCustomLogic {
		t.Fatalf("expected custom logic primary, got %+v", PhaseOneInsightsPattern)
	}
	if len(PhaseOneInsightsPattern.Supporting) != 1 || PhaseOneInsightsPattern.Supporting[0] != PatternSingleAgent {
		t.Fatalf("expected single-agent supporting pattern, got %+v", PhaseOneInsightsPattern)
	}
	if PhaseOneInsightsPattern.Reference != GoogleDesignPatternGuideURL {
		t.Fatalf("expected google guide reference, got %+v", PhaseOneInsightsPattern)
	}
}
