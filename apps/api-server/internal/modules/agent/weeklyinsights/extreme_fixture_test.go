package weeklyinsights

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type extremeCase struct {
	ID                 string                   `json:"id"`
	Why                string                   `json:"why"`
	RegressionFocus    string                   `json:"regression_focus"`
	ShouldNotBreakWhen string                   `json:"should_not_break_when"`
	DaysData           []map[string]interface{} `json:"days_data"`
	ExpectedTotal      int                      `json:"expected_total"`
	ExpectedCompleted  int                      `json:"expected_completed"`
	ExpectedSummary    string                   `json:"expected_summary"`
}

func loadExtremeCases(t *testing.T) []extremeCase {
	t.Helper()

	path := filepath.Join("testdata", "extreme_cases.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read extreme cases %s: %v", path, err)
	}

	var cases []extremeCase
	if err := json.Unmarshal(content, &cases); err != nil {
		t.Fatalf("decode extreme cases %s: %v", path, err)
	}
	if len(cases) == 0 {
		t.Fatalf("expected extreme cases in %s", path)
	}

	return cases
}

func TestGenerateExtremeFixtures(t *testing.T) {
	service := NewService(nil)
	for _, tc := range loadExtremeCases(t) {
		t.Run(tc.ID, func(t *testing.T) {
			if tc.Why == "" || tc.RegressionFocus == "" || tc.ShouldNotBreakWhen == "" {
				t.Fatalf("expected weekly extreme case metadata for %s: %+v", tc.ID, tc)
			}
			insight, err := service.Generate(context.Background(), tc.DaysData)
			if err != nil {
				t.Fatalf("%s: expected nil error, got %v", tc.Why, err)
			}
			if insight.RawMetricTotal != tc.ExpectedTotal || insight.RawMetricCompleted != tc.ExpectedCompleted {
				t.Fatalf("%s: unexpected metrics %+v", tc.Why, insight)
			}
			if insight.Summary != tc.ExpectedSummary {
				t.Fatalf("%s: expected summary %q, got %q", tc.Why, tc.ExpectedSummary, insight.Summary)
			}
			if len(insight.Strengths) != 3 {
				t.Fatalf("%s: expected 3 strengths, got %+v", tc.Why, insight.Strengths)
			}
			if len(insight.AreasForImprovement) != 3 {
				t.Fatalf("%s: expected 3 improvements, got %+v", tc.Why, insight.AreasForImprovement)
			}
			if strings.TrimSpace(insight.PsychologicalInsight) == "" {
				t.Fatalf("%s: expected psychological insight to be populated: %+v", tc.Why, insight)
			}
		})
	}
}
