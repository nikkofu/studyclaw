package progressinsights

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

type reportFixture struct {
	ID                            string `json:"id"`
	Why                           string `json:"why"`
	Stats                         Stats  `json:"stats"`
	ExpectedSummary               string `json:"expected_summary"`
	ExpectedTotal                 int    `json:"expected_total"`
	ExpectedCompleted             int    `json:"expected_completed"`
	ExpectedCompletionRatePercent int    `json:"expected_completion_rate_percent"`
}

func loadReportFixtures(t *testing.T) []reportFixture {
	t.Helper()

	path := filepath.Join("testdata", "report_cases.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report fixtures %s: %v", path, err)
	}

	var fixtures []reportFixture
	if err := json.Unmarshal(content, &fixtures); err != nil {
		t.Fatalf("decode report fixtures %s: %v", path, err)
	}
	if len(fixtures) == 0 {
		t.Fatalf("expected report fixtures in %s", path)
	}

	return fixtures
}

func TestBuildStatsFromDaysAggregatesSubjectsAndTimeline(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-09",
			"tasks": []taskboarddomain.Task{
				{TaskID: 1, Subject: "数学", Content: "口算", Completed: true},
				{TaskID: 2, Subject: "英语", Content: "听写", Completed: false},
			},
		},
		{
			"date": "2026-03-10",
			"tasks": []map[string]interface{}{
				{"subject": "数学", "completed": false},
				{"subject": "语文", "completed": true},
			},
		},
	}

	stats := BuildStatsFromDays(ReportTypeWeekly, "2026-W11", daysData, 3)

	if stats.ReportType != ReportTypeWeekly {
		t.Fatalf("expected weekly report type, got %+v", stats)
	}
	if stats.TotalTasks != 4 || stats.CompletedTasks != 2 {
		t.Fatalf("unexpected totals: %+v", stats)
	}
	if stats.CompletionRatePercent != 50 {
		t.Fatalf("expected completion rate 50, got %+v", stats)
	}
	if stats.PointsDelta != 3 {
		t.Fatalf("expected points delta 3, got %+v", stats)
	}
	if len(stats.SubjectBreakdown) != 3 {
		t.Fatalf("expected 3 subject breakdown items, got %+v", stats.SubjectBreakdown)
	}
	if stats.SubjectBreakdown[0].Subject != "数学" || stats.SubjectBreakdown[0].TotalTasks != 2 {
		t.Fatalf("expected sorted subject breakdown, got %+v", stats.SubjectBreakdown)
	}
	if len(stats.Timeline) != 2 || stats.Timeline[0].Label != "2026-03-09" {
		t.Fatalf("expected sorted timeline, got %+v", stats.Timeline)
	}
}

func TestGenerateNormalizesLLMOutputAndPreservesDeterministicMetrics(t *testing.T) {
	service := NewService(stubInsightsLLMClient{
		response: "```json\n{\n  \"summary\": \"  You stayed steady this month.  \",\n  \"strengths\": [\" Focused effort \", \"Focused effort\", \"2. Kept showing up\"],\n  \"areas_for_improvement\": [\"\", \"Start a little earlier\", \"3. Finish older tasks first\"],\n  \"psychological_insight\": \"  Confidence grows when you return to unfinished work.  \"\n}\n```",
	})

	report, err := service.Generate(context.Background(), Stats{
		ReportType:     ReportTypeMonthly,
		PeriodLabel:    "2026-03",
		TotalTasks:     10,
		CompletedTasks: 7,
		PointsDelta:    4,
		SubjectBreakdown: []SubjectMetric{
			{Subject: "数学", TotalTasks: 4, CompletedTasks: 3},
			{Subject: "英语", TotalTasks: 6, CompletedTasks: 4},
		},
		Timeline: []BucketMetric{
			{Label: "2026-03-10", TotalTasks: 5, CompletedTasks: 4},
			{Label: "2026-03-20", TotalTasks: 5, CompletedTasks: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if report.Summary != "You stayed steady this month." {
		t.Fatalf("unexpected summary %q", report.Summary)
	}
	if report.RawMetricTotal != 10 || report.RawMetricCompleted != 7 || report.RawPointsDelta != 4 {
		t.Fatalf("expected deterministic metrics to be preserved, got %+v", report)
	}
	if report.CompletionRatePercent != 70 {
		t.Fatalf("expected completion rate 70, got %+v", report)
	}
	if len(report.Strengths) != 3 || report.Strengths[0] != "Focused effort" {
		t.Fatalf("unexpected normalized strengths %+v", report.Strengths)
	}
	if len(report.AreasForImprovement) != 3 || report.AreasForImprovement[0] != "Start a little earlier" {
		t.Fatalf("unexpected normalized improvements %+v", report.AreasForImprovement)
	}
	if report.PsychologicalInsight != "Confidence grows when you return to unfinished work." {
		t.Fatalf("unexpected psychological insight %q", report.PsychologicalInsight)
	}
	if report.AgenticPattern.Primary != "custom logic pattern" {
		t.Fatalf("unexpected pattern %+v", report.AgenticPattern)
	}
}

func TestBuildPromptUsesDeterministicStatsPayload(t *testing.T) {
	prompt, err := BuildPrompt(Stats{
		ReportType:     ReportTypeDaily,
		PeriodLabel:    "2026-03-10",
		TotalTasks:     4,
		CompletedTasks: 3,
		PointsDelta:    2,
		SubjectBreakdown: []SubjectMetric{
			{Subject: "数学", TotalTasks: 2, CompletedTasks: 1},
			{Subject: "英语", TotalTasks: 2, CompletedTasks: 2},
		},
		Timeline: []BucketMetric{
			{Label: "2026-03-10", TotalTasks: 4, CompletedTasks: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertContains(t, prompt, "deterministic daily study metrics")
	assertContains(t, prompt, `"report_type":"daily"`)
	assertContains(t, prompt, `"total_tasks":4`)
	assertContains(t, prompt, `"completed_tasks":3`)
	assertContains(t, prompt, `"points_delta":2`)
	assertContains(t, prompt, "Do not rewrite or modify any statistic")
}

func TestGenerateTemplateReportFixtures(t *testing.T) {
	service := NewService(nil)
	for _, fixture := range loadReportFixtures(t) {
		t.Run(fixture.ID, func(t *testing.T) {
			report, err := service.Generate(context.Background(), fixture.Stats)
			if err != nil {
				t.Fatalf("%s: expected nil error, got %v", fixture.Why, err)
			}
			if report.Summary != fixture.ExpectedSummary {
				t.Fatalf("%s: expected summary %q, got %q", fixture.Why, fixture.ExpectedSummary, report.Summary)
			}
			if report.RawMetricTotal != fixture.ExpectedTotal || report.RawMetricCompleted != fixture.ExpectedCompleted {
				t.Fatalf("%s: unexpected metrics %+v", fixture.Why, report)
			}
			if report.CompletionRatePercent != fixture.ExpectedCompletionRatePercent {
				t.Fatalf("%s: expected completion rate %d, got %+v", fixture.Why, fixture.ExpectedCompletionRatePercent, report)
			}
			if len(report.Strengths) != 3 || len(report.AreasForImprovement) != 3 {
				t.Fatalf("%s: expected complete template lists, got %+v", fixture.Why, report)
			}
			if strings.TrimSpace(report.PsychologicalInsight) == "" {
				t.Fatalf("%s: expected psychological insight to be populated", fixture.Why)
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

type stubInsightsLLMClient struct {
	response string
	err      error
}

func (s stubInsightsLLMClient) Generate(_ context.Context, _ llm.GenerateRequest) (string, error) {
	return s.response, s.err
}
