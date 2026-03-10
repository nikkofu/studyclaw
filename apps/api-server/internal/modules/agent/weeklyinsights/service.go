package weeklyinsights

import (
	"context"

	"github.com/nikkofu/studyclaw/api-server/internal/modules/agent/progressinsights"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
	"github.com/nikkofu/studyclaw/api-server/internal/shared/agentic"
)

var weeklyPatternSelection = agentic.PhaseOneInsightsPattern

type Insight struct {
	Summary              string                   `json:"summary"`
	Strengths            []string                 `json:"strengths"`
	AreasForImprovement  []string                 `json:"areas_for_improvement"`
	PsychologicalInsight string                   `json:"psychological_insight"`
	RawMetricTotal       int                      `json:"raw_metric_total"`
	RawMetricCompleted   int                      `json:"raw_metric_completed"`
	AgenticPattern       agentic.PatternSelection `json:"agentic_pattern"`
}

type Service struct {
	delegate *progressinsights.Service
}

func NewService(llmClient llm.Client) *Service {
	return &Service{delegate: progressinsights.NewService(llmClient)}
}

func buildWeeklyStats(daysData []map[string]interface{}) progressinsights.Stats {
	return progressinsights.BuildStatsFromDays(progressinsights.ReportTypeWeekly, "", daysData, 0)
}

func convertReport(report progressinsights.Report) Insight {
	return Insight{
		Summary:              report.Summary,
		Strengths:            report.Strengths,
		AreasForImprovement:  report.AreasForImprovement,
		PsychologicalInsight: report.PsychologicalInsight,
		RawMetricTotal:       report.RawMetricTotal,
		RawMetricCompleted:   report.RawMetricCompleted,
		AgenticPattern:       weeklyPatternSelection,
	}
}

func generateMock(daysData []map[string]interface{}) Insight {
	return convertReport(progressinsights.BuildTemplateReport(buildWeeklyStats(daysData)))
}

func buildPrompt(daysData []map[string]interface{}) (string, error) {
	return progressinsights.BuildPrompt(buildWeeklyStats(daysData))
}

func (s *Service) Generate(ctx context.Context, daysData []map[string]interface{}) (Insight, error) {
	report, err := s.delegate.Generate(ctx, buildWeeklyStats(daysData))
	if err != nil {
		return Insight{}, err
	}
	return convertReport(report), nil
}
