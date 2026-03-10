package weeklyinsights

import (
	"context"
	"strings"
	"testing"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

func TestGenerateMockCountsTypedDomainTasks(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-06",
			"tasks": []taskboarddomain.Task{
				{TaskID: 1, Subject: "数学", Content: "口算", Completed: true},
				{TaskID: 2, Subject: "英语", Content: "听写", Completed: false},
				{TaskID: 3, Subject: "语文", Content: "背诵", Completed: true},
			},
		},
	}

	insight := generateMock(daysData)

	if insight.RawMetricTotal != 3 {
		t.Fatalf("expected total 3, got %d", insight.RawMetricTotal)
	}
	if insight.RawMetricCompleted != 2 {
		t.Fatalf("expected completed 2, got %d", insight.RawMetricCompleted)
	}
	if len(insight.Strengths) != 3 {
		t.Fatalf("expected 3 strengths, got %+v", insight.Strengths)
	}
	if len(insight.AreasForImprovement) != 3 {
		t.Fatalf("expected 3 improvement areas, got %+v", insight.AreasForImprovement)
	}
	if insight.AgenticPattern.Primary != "custom logic pattern" {
		t.Fatalf("unexpected primary pattern: %+v", insight.AgenticPattern)
	}
}

func TestGenerateNormalizesLLMOutputAndPreservesMetrics(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-06",
			"tasks": []taskboarddomain.Task{
				{TaskID: 1, Subject: "数学", Content: "口算", Completed: true},
				{TaskID: 2, Subject: "英语", Content: "听写", Completed: false},
				{TaskID: 3, Subject: "语文", Content: "背诵", Completed: true},
			},
		},
	}

	service := NewService(stubWeeklyLLMClient{
		response: "```json\n{\n  \"summary\": \"  You kept moving this week.  \",\n  \"strengths\": [\" Focused effort \", \"Focused effort\", \"2. Kept going\", \"- Used multiple study days\"],\n  \"areas_for_improvement\": [\"\", \"Start earlier\", \"Start earlier\", \"3. Finish pending tasks\"],\n  \"psychological_insight\": \"  Progress grows when you return to unfinished work.  \"\n}\n```",
	})

	insight, err := service.Generate(context.Background(), daysData)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if insight.Summary != "You kept moving this week." {
		t.Fatalf("unexpected normalized summary: %q", insight.Summary)
	}
	if insight.RawMetricTotal != 3 || insight.RawMetricCompleted != 2 {
		t.Fatalf("unexpected raw metrics: %+v", insight)
	}
	if len(insight.Strengths) != 3 {
		t.Fatalf("expected normalized strengths to contain 3 items, got %+v", insight.Strengths)
	}
	if insight.Strengths[0] != "Focused effort" || insight.Strengths[1] != "Kept going" {
		t.Fatalf("unexpected normalized strengths: %+v", insight.Strengths)
	}
	if len(insight.AreasForImprovement) != 3 {
		t.Fatalf("expected normalized improvements to contain 3 items, got %+v", insight.AreasForImprovement)
	}
	if insight.AreasForImprovement[0] != "Start earlier" || insight.AreasForImprovement[1] != "Finish pending tasks" {
		t.Fatalf("unexpected normalized improvements: %+v", insight.AreasForImprovement)
	}
	if insight.PsychologicalInsight != "Progress grows when you return to unfinished work." {
		t.Fatalf("unexpected psychological insight: %q", insight.PsychologicalInsight)
	}
	if insight.AgenticPattern.Primary != "custom logic pattern" {
		t.Fatalf("unexpected primary pattern: %+v", insight.AgenticPattern)
	}
}

func TestGenerateFallsBackWhenLLMLeavesFieldsBlank(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-06",
			"tasks": []taskboarddomain.Task{
				{TaskID: 1, Subject: "数学", Content: "口算", Completed: true},
				{TaskID: 2, Subject: "英语", Content: "听写", Completed: false},
			},
		},
	}

	service := NewService(stubWeeklyLLMClient{
		response: `{"summary":" ","strengths":[""],"areas_for_improvement":[],"psychological_insight":" "}`,
	})

	insight, err := service.Generate(context.Background(), daysData)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	fallback := generateMock(daysData)
	if insight.Summary != fallback.Summary {
		t.Fatalf("expected fallback summary %q, got %q", fallback.Summary, insight.Summary)
	}
	if len(insight.Strengths) != 3 || len(insight.AreasForImprovement) != 3 {
		t.Fatalf("expected fallback lists, got strengths=%+v improvements=%+v", insight.Strengths, insight.AreasForImprovement)
	}
	if insight.PsychologicalInsight != fallback.PsychologicalInsight {
		t.Fatalf("expected fallback psychological insight %q, got %q", fallback.PsychologicalInsight, insight.PsychologicalInsight)
	}
}

func TestBuildPromptUsesDeterministicWeeklyMetrics(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-07",
			"tasks": []map[string]interface{}{
				{"completed": true},
			},
		},
		{
			"date": "2026-03-06",
			"tasks": []map[string]interface{}{
				{"completed": true},
				{"completed": false},
			},
		},
	}

	prompt, err := buildPrompt(daysData)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertContains(t, prompt, `"total_tasks":3`)
	assertContains(t, prompt, `"completed_tasks":2`)
	assertContains(t, prompt, `"completion_rate_percent":66`)
	assertContains(t, prompt, `"active_days":2`)

	date6 := strings.Index(prompt, `"label":"2026-03-06"`)
	date7 := strings.Index(prompt, `"label":"2026-03-07"`)
	if date6 == -1 || date7 == -1 || date6 >= date7 {
		t.Fatalf("expected prompt dates to be sorted, got prompt %q", prompt)
	}
}

func TestGenerateHandlesExtremeInputs(t *testing.T) {
	testCases := []struct {
		name             string
		daysData         []map[string]interface{}
		expectedTotal    int
		expectedComplete int
		expectedSummary  string
	}{
		{
			name:             "empty data",
			daysData:         nil,
			expectedTotal:    0,
			expectedComplete: 0,
			expectedSummary:  "This week was light, which is a good chance to build a steady study rhythm for next week.",
		},
		{
			name: "single day mixed completion",
			daysData: []map[string]interface{}{
				{
					"date": "2026-03-08",
					"tasks": []map[string]interface{}{
						{"completed": true},
						{"completed": false},
					},
				},
			},
			expectedTotal:    2,
			expectedComplete: 1,
			expectedSummary:  "You worked on 2 tasks this week and completed 1 of them, which means your effort is moving things forward.",
		},
		{
			name: "all completed",
			daysData: []map[string]interface{}{
				{
					"date": "2026-03-06",
					"tasks": []map[string]interface{}{
						{"completed": true},
						{"completed": true},
						{"completed": true},
					},
				},
			},
			expectedTotal:    3,
			expectedComplete: 3,
			expectedSummary:  "You completed all 3 tasks this week, which shows steady follow-through from start to finish.",
		},
		{
			name: "all incomplete",
			daysData: []map[string]interface{}{
				{
					"date": "2026-03-06",
					"tasks": []map[string]interface{}{
						{"completed": false},
						{"completed": false},
						{"completed": false},
					},
				},
			},
			expectedTotal:    3,
			expectedComplete: 0,
			expectedSummary:  "You worked on 3 tasks this week and completed 0 of them, which means your effort is moving things forward.",
		},
	}

	service := NewService(nil)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			insight, err := service.Generate(context.Background(), tc.daysData)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if insight.RawMetricTotal != tc.expectedTotal || insight.RawMetricCompleted != tc.expectedComplete {
				t.Fatalf("unexpected metrics: %+v", insight)
			}
			if insight.Summary != tc.expectedSummary {
				t.Fatalf("expected summary %q, got %q", tc.expectedSummary, insight.Summary)
			}
			if len(insight.Strengths) != 3 {
				t.Fatalf("expected 3 strengths, got %+v", insight.Strengths)
			}
			if len(insight.AreasForImprovement) != 3 {
				t.Fatalf("expected 3 improvements, got %+v", insight.AreasForImprovement)
			}
			if strings.TrimSpace(insight.PsychologicalInsight) == "" {
				t.Fatalf("expected psychological insight to be populated: %+v", insight)
			}
		})
	}
}

func assertContains(t *testing.T, text string, expected string) {
	t.Helper()

	if !strings.Contains(text, expected) {
		t.Fatalf("expected %q to contain %q", text, expected)
	}
}

type stubWeeklyLLMClient struct {
	response string
	err      error
}

func (s stubWeeklyLLMClient) Generate(_ context.Context, _ llm.GenerateRequest) (string, error) {
	return s.response, s.err
}
