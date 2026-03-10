package progressinsights

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
	"github.com/nikkofu/studyclaw/api-server/internal/shared/agentic"
)

const (
	ReportTypeDaily   = "daily"
	ReportTypeWeekly  = "weekly"
	ReportTypeMonthly = "monthly"
)

var reportPatternSelection = agentic.PhaseOneInsightsPattern

type SubjectMetric struct {
	Subject        string `json:"subject"`
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
	PendingTasks   int    `json:"pending_tasks"`
}

type BucketMetric struct {
	Label          string `json:"label,omitempty"`
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
	PendingTasks   int    `json:"pending_tasks"`
}

type Stats struct {
	ReportType            string          `json:"report_type"`
	PeriodLabel           string          `json:"period_label,omitempty"`
	TotalTasks            int             `json:"total_tasks"`
	CompletedTasks        int             `json:"completed_tasks"`
	PendingTasks          int             `json:"pending_tasks"`
	CompletionRatePercent int             `json:"completion_rate_percent"`
	ActiveDays            int             `json:"active_days"`
	PointsDelta           int             `json:"points_delta"`
	SubjectBreakdown      []SubjectMetric `json:"subject_breakdown,omitempty"`
	Timeline              []BucketMetric  `json:"timeline,omitempty"`
}

type Report struct {
	ReportType            string                   `json:"report_type"`
	PeriodLabel           string                   `json:"period_label,omitempty"`
	Summary               string                   `json:"summary"`
	Strengths             []string                 `json:"strengths"`
	AreasForImprovement   []string                 `json:"areas_for_improvement"`
	PsychologicalInsight  string                   `json:"psychological_insight"`
	RawMetricTotal        int                      `json:"raw_metric_total"`
	RawMetricCompleted    int                      `json:"raw_metric_completed"`
	CompletionRatePercent int                      `json:"completion_rate_percent"`
	RawPointsDelta        int                      `json:"raw_points_delta"`
	ActiveDays            int                      `json:"active_days"`
	SubjectBreakdown      []SubjectMetric          `json:"subject_breakdown,omitempty"`
	AgenticPattern        agentic.PatternSelection `json:"agentic_pattern"`
}

type Service struct {
	llmClient llm.Client
}

type llmReport struct {
	Summary              string   `json:"summary"`
	Strengths            []string `json:"strengths"`
	AreasForImprovement  []string `json:"areas_for_improvement"`
	PsychologicalInsight string   `json:"psychological_insight"`
}

type taskSnapshot struct {
	Subject   string
	Completed bool
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func canonicalReportType(reportType string) string {
	switch strings.ToLower(strings.TrimSpace(reportType)) {
	case ReportTypeDaily:
		return ReportTypeDaily
	case ReportTypeMonthly:
		return ReportTypeMonthly
	default:
		return ReportTypeWeekly
	}
}

func normalizeStats(stats Stats) Stats {
	stats.ReportType = canonicalReportType(stats.ReportType)
	stats.PeriodLabel = strings.TrimSpace(stats.PeriodLabel)
	if stats.TotalTasks < 0 {
		stats.TotalTasks = 0
	}
	if stats.CompletedTasks < 0 {
		stats.CompletedTasks = 0
	}
	stats.PendingTasks = stats.TotalTasks - stats.CompletedTasks
	if stats.PendingTasks < 0 {
		stats.PendingTasks = 0
	}
	if stats.CompletionRatePercent == 0 && stats.TotalTasks > 0 {
		stats.CompletionRatePercent = stats.CompletedTasks * 100 / stats.TotalTasks
	}
	if stats.TotalTasks == 0 {
		stats.CompletionRatePercent = 0
	}

	subjects := make([]SubjectMetric, 0, len(stats.SubjectBreakdown))
	for _, subject := range stats.SubjectBreakdown {
		trimmedSubject := strings.TrimSpace(subject.Subject)
		if trimmedSubject == "" {
			continue
		}
		if subject.TotalTasks < 0 {
			subject.TotalTasks = 0
		}
		if subject.CompletedTasks < 0 {
			subject.CompletedTasks = 0
		}
		subject.PendingTasks = subject.TotalTasks - subject.CompletedTasks
		if subject.PendingTasks < 0 {
			subject.PendingTasks = 0
		}
		subject.Subject = trimmedSubject
		subjects = append(subjects, subject)
	}
	sort.Slice(subjects, func(i, j int) bool {
		if subjects[i].TotalTasks == subjects[j].TotalTasks {
			return subjects[i].Subject < subjects[j].Subject
		}
		return subjects[i].TotalTasks > subjects[j].TotalTasks
	})
	stats.SubjectBreakdown = subjects

	timeline := make([]BucketMetric, 0, len(stats.Timeline))
	for _, bucket := range stats.Timeline {
		if bucket.TotalTasks < 0 {
			bucket.TotalTasks = 0
		}
		if bucket.CompletedTasks < 0 {
			bucket.CompletedTasks = 0
		}
		bucket.PendingTasks = bucket.TotalTasks - bucket.CompletedTasks
		if bucket.PendingTasks < 0 {
			bucket.PendingTasks = 0
		}
		bucket.Label = strings.TrimSpace(bucket.Label)
		timeline = append(timeline, bucket)
	}
	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].Label == timeline[j].Label {
			return timeline[i].TotalTasks < timeline[j].TotalTasks
		}
		if timeline[i].Label == "" {
			return false
		}
		if timeline[j].Label == "" {
			return true
		}
		return timeline[i].Label < timeline[j].Label
	})
	stats.Timeline = timeline

	if stats.ActiveDays == 0 {
		for _, bucket := range stats.Timeline {
			if bucket.TotalTasks > 0 {
				stats.ActiveDays++
			}
		}
		if stats.ActiveDays == 0 && stats.TotalTasks > 0 && stats.ReportType == ReportTypeDaily {
			stats.ActiveDays = 1
		}
	}

	return stats
}

func BuildStatsFromDays(reportType, periodLabel string, daysData []map[string]interface{}, pointsDelta int) Stats {
	subjects := make(map[string]SubjectMetric)
	timeline := make([]BucketMetric, 0, len(daysData))
	totalTasks := 0
	completedTasks := 0
	activeDays := 0

	for _, day := range daysData {
		snapshots := extractTaskSnapshots(day["tasks"])
		dayTotal := len(snapshots)
		dayCompleted := 0
		for _, snapshot := range snapshots {
			if snapshot.Completed {
				dayCompleted++
			}
			subject := strings.TrimSpace(snapshot.Subject)
			if subject == "" {
				continue
			}
			metric := subjects[subject]
			metric.Subject = subject
			metric.TotalTasks++
			if snapshot.Completed {
				metric.CompletedTasks++
			}
			subjects[subject] = metric
		}
		if dayTotal > 0 {
			activeDays++
		}
		totalTasks += dayTotal
		completedTasks += dayCompleted

		label, _ := day["date"].(string)
		timeline = append(timeline, BucketMetric{
			Label:          strings.TrimSpace(label),
			TotalTasks:     dayTotal,
			CompletedTasks: dayCompleted,
			PendingTasks:   dayTotal - dayCompleted,
		})
	}

	subjectBreakdown := make([]SubjectMetric, 0, len(subjects))
	for _, metric := range subjects {
		metric.PendingTasks = metric.TotalTasks - metric.CompletedTasks
		subjectBreakdown = append(subjectBreakdown, metric)
	}

	return normalizeStats(Stats{
		ReportType:       reportType,
		PeriodLabel:      periodLabel,
		TotalTasks:       totalTasks,
		CompletedTasks:   completedTasks,
		PendingTasks:     totalTasks - completedTasks,
		ActiveDays:       activeDays,
		PointsDelta:      pointsDelta,
		SubjectBreakdown: subjectBreakdown,
		Timeline:         timeline,
	})
}

func extractTaskSnapshots(rawTasks any) []taskSnapshot {
	switch typed := rawTasks.(type) {
	case []interface{}:
		snapshots := make([]taskSnapshot, 0, len(typed))
		for _, item := range typed {
			snapshots = append(snapshots, toTaskSnapshot(item))
		}
		return snapshots
	case []map[string]interface{}:
		snapshots := make([]taskSnapshot, 0, len(typed))
		for _, item := range typed {
			snapshots = append(snapshots, toTaskSnapshot(item))
		}
		return snapshots
	}

	value := reflect.ValueOf(rawTasks)
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil
	}

	snapshots := make([]taskSnapshot, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		snapshots = append(snapshots, toTaskSnapshot(value.Index(index).Interface()))
	}
	return snapshots
}

func toTaskSnapshot(task any) taskSnapshot {
	switch typed := task.(type) {
	case map[string]interface{}:
		completed, _ := typed["completed"].(bool)
		subject, _ := typed["subject"].(string)
		return taskSnapshot{
			Subject:   strings.TrimSpace(subject),
			Completed: completed,
		}
	}

	value := reflect.ValueOf(task)
	if !value.IsValid() {
		return taskSnapshot{}
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return taskSnapshot{}
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return taskSnapshot{}
	}

	snapshot := taskSnapshot{}
	if subjectField := value.FieldByName("Subject"); subjectField.IsValid() && subjectField.Kind() == reflect.String {
		snapshot.Subject = strings.TrimSpace(subjectField.String())
	}
	if completedField := value.FieldByName("Completed"); completedField.IsValid() && completedField.Kind() == reflect.Bool {
		snapshot.Completed = completedField.Bool()
	}
	return snapshot
}

func appendUniqueText(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func trimInsightText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	value = strings.TrimSpace(strings.TrimLeft(value, "-•0123456789.、)）"))
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func normalizeInsightList(values []string, fallback []string) []string {
	normalized := make([]string, 0, 3)
	for _, value := range values {
		trimmed := trimInsightText(value)
		if trimmed == "" {
			continue
		}
		normalized = appendUniqueText(normalized, trimmed)
		if len(normalized) == 3 {
			return normalized
		}
	}
	for _, value := range fallback {
		trimmed := trimInsightText(value)
		if trimmed == "" {
			continue
		}
		normalized = appendUniqueText(normalized, trimmed)
		if len(normalized) == 3 {
			return normalized
		}
	}
	return normalized
}

func reportPeriodName(reportType string) string {
	switch canonicalReportType(reportType) {
	case ReportTypeDaily:
		return "today"
	case ReportTypeMonthly:
		return "this month"
	default:
		return "this week"
	}
}

func reportPromptName(reportType string) string {
	switch canonicalReportType(reportType) {
	case ReportTypeDaily:
		return "daily"
	case ReportTypeMonthly:
		return "monthly"
	default:
		return "weekly"
	}
}

func buildTemplateSummary(stats Stats) string {
	switch canonicalReportType(stats.ReportType) {
	case ReportTypeDaily:
		switch {
		case stats.TotalTasks == 0:
			return "Today was a lighter study day, which gives you room to reset and start strong tomorrow."
		case stats.CompletedTasks == stats.TotalTasks:
			return fmt.Sprintf("You completed all %d tasks today, which shows steady focus from start to finish.", stats.TotalTasks)
		default:
			return fmt.Sprintf("You finished %d of %d tasks today, and every completed step helped your day move forward.", stats.CompletedTasks, stats.TotalTasks)
		}
	case ReportTypeMonthly:
		switch {
		case stats.TotalTasks == 0:
			return "This month was lighter, which gives you a good chance to build a calmer routine for the next month."
		case stats.CompletedTasks == stats.TotalTasks:
			return fmt.Sprintf("You completed all %d tasks this month, which shows patient and steady effort over time.", stats.TotalTasks)
		default:
			return fmt.Sprintf("You worked on %d tasks this month and completed %d of them, which shows your routine is growing more consistent.", stats.TotalTasks, stats.CompletedTasks)
		}
	default:
		switch {
		case stats.TotalTasks == 0:
			return "This week was light, which is a good chance to build a steady study rhythm for next week."
		case stats.CompletedTasks == stats.TotalTasks:
			return fmt.Sprintf("You completed all %d tasks this week, which shows steady follow-through from start to finish.", stats.TotalTasks)
		default:
			return fmt.Sprintf("You worked on %d tasks this week and completed %d of them, which means your effort is moving things forward.", stats.TotalTasks, stats.CompletedTasks)
		}
	}
}

func buildTemplateStrengths(stats Stats) []string {
	period := reportPeriodName(stats.ReportType)
	candidates := []string{
		fmt.Sprintf("You kept showing up for your work %s instead of giving up.", period),
		"You made progress one task at a time.",
		"You are building a more reliable study routine.",
	}
	if stats.ActiveDays >= 3 && canonicalReportType(stats.ReportType) != ReportTypeDaily {
		candidates = append([]string{"You spread your effort across multiple days instead of rushing everything at once."}, candidates...)
	}
	if stats.CompletionRatePercent >= 80 && stats.TotalTasks > 0 {
		candidates = append([]string{"You followed through on most of the work that was assigned."}, candidates...)
	}
	if stats.CompletedTasks == stats.TotalTasks && stats.TotalTasks > 0 {
		candidates = append([]string{"You finished the full list, which shows strong task completion habits."}, candidates...)
	}
	if stats.PointsDelta > 0 {
		candidates = append([]string{fmt.Sprintf("Your points moved up by %d, which shows your effort was noticed.", stats.PointsDelta)}, candidates...)
	}
	if stats.TotalTasks == 0 {
		candidates = []string{
			fmt.Sprintf("You kept %s calm and ready for the next round of work.", period),
			"You had room to reset your routine without extra pressure.",
			"You are ready to restart with a clear plan.",
		}
	}
	return normalizeInsightList(candidates, nil)
}

func buildTemplateAreas(stats Stats) []string {
	switch canonicalReportType(stats.ReportType) {
	case ReportTypeDaily:
		if stats.TotalTasks == 0 {
			return normalizeInsightList([]string{
				"Set one small study goal for tomorrow so starting feels easy.",
				"Choose a simple homework time and protect it.",
				"Check tomorrow's task list early so the day feels clearer.",
			}, nil)
		}
		if stats.CompletionRatePercent >= 80 {
			return normalizeInsightList([]string{
				"Keep the same steady pace tomorrow so strong days become a habit.",
				"Start the first task early while your energy is fresh.",
				"Use the same step-by-step method on the next harder task.",
			}, nil)
		}
		return normalizeInsightList([]string{
			"Start the first task a little earlier so the day feels lighter.",
			"Finish older pending work before adding too many new steps.",
			"Break bigger tasks into smaller actions and clear them in order.",
		}, nil)
	case ReportTypeMonthly:
		if stats.TotalTasks == 0 {
			return normalizeInsightList([]string{
				"Set one simple study rhythm for the next month before the busy days arrive.",
				"Choose a regular review time each week so momentum grows naturally.",
				"Check in on your task list often, even in lighter weeks.",
			}, nil)
		}
		if stats.CompletionRatePercent >= 80 {
			return normalizeInsightList([]string{
				"Keep the same monthly rhythm so consistency stays strong across longer stretches.",
				"Notice which study times helped you finish tasks and repeat them next month.",
				"Carry the same calm routine into the next group of harder assignments.",
			}, nil)
		}
		return normalizeInsightList([]string{
			"Split bigger monthly goals into smaller weekly checkpoints.",
			"Finish older pending work before it grows into a larger pile.",
			"Keep a steadier routine across the month instead of waiting for the last few days.",
		}, nil)
	default:
		if stats.TotalTasks == 0 {
			return normalizeInsightList([]string{
				"Set up a simple homework rhythm before the next busy week starts.",
				"Pick one small study goal early so momentum is easier to build.",
				"Check the task list each day even when the workload is light.",
			}, nil)
		}
		if stats.CompletionRatePercent >= 80 {
			return normalizeInsightList([]string{
				"Keep using the same steady pace so strong weeks become a habit.",
				"Notice which study time helped you stay focused and repeat it.",
				"Use the same step-by-step approach on the next harder assignment.",
			}, nil)
		}
		return normalizeInsightList([]string{
			"Start the harder tasks a little earlier so they feel smaller.",
			"Finish older pending work before adding too many new tasks.",
			"Break bigger assignments into smaller steps and clear them in order.",
		}, nil)
	}
}

func buildTemplatePsychologicalInsight(stats Stats) string {
	switch canonicalReportType(stats.ReportType) {
	case ReportTypeDaily:
		switch {
		case stats.TotalTasks == 0:
			return "A lighter day can still build confidence when you use it to reset and get ready for tomorrow."
		case stats.CompletionRatePercent >= 80:
			return "Today's progress shows that calm follow-through is often stronger than waiting to feel perfectly ready."
		default:
			return "Every finished task today is proof that steady effort grows when you keep moving one step at a time."
		}
	case ReportTypeMonthly:
		switch {
		case stats.TotalTasks == 0:
			return "A lighter month can still strengthen confidence when you use it to protect your routine."
		case stats.CompletionRatePercent >= 80:
			return "This month's progress shows that consistency grows from many small follow-through moments, not one perfect day."
		default:
			return "Long-term confidence grows when you keep returning to unfinished work instead of treating setbacks as the end."
		}
	default:
		switch {
		case stats.TotalTasks == 0:
			return "A lighter week can still strengthen confidence when you use it to protect your routine."
		case stats.CompletionRatePercent >= 80:
			return "Your progress came from steady follow-through, which is a stronger habit than waiting to feel perfectly ready."
		default:
			return "Each finished task is evidence that persistence grows when you keep moving one step at a time."
		}
	}
}

func buildTemplateReport(stats Stats) Report {
	stats = normalizeStats(stats)
	return Report{
		ReportType:            stats.ReportType,
		PeriodLabel:           stats.PeriodLabel,
		Summary:               buildTemplateSummary(stats),
		Strengths:             buildTemplateStrengths(stats),
		AreasForImprovement:   buildTemplateAreas(stats),
		PsychologicalInsight:  buildTemplatePsychologicalInsight(stats),
		RawMetricTotal:        stats.TotalTasks,
		RawMetricCompleted:    stats.CompletedTasks,
		CompletionRatePercent: stats.CompletionRatePercent,
		RawPointsDelta:        stats.PointsDelta,
		ActiveDays:            stats.ActiveDays,
		SubjectBreakdown:      stats.SubjectBreakdown,
		AgenticPattern:        reportPatternSelection,
	}
}

func BuildTemplateReport(stats Stats) Report {
	return buildTemplateReport(stats)
}

func normalizeReportWithFallback(parsed llmReport, fallback Report) Report {
	summary := trimInsightText(parsed.Summary)
	if summary == "" {
		summary = fallback.Summary
	}
	psychologicalInsight := trimInsightText(parsed.PsychologicalInsight)
	if psychologicalInsight == "" {
		psychologicalInsight = fallback.PsychologicalInsight
	}

	return Report{
		ReportType:            fallback.ReportType,
		PeriodLabel:           fallback.PeriodLabel,
		Summary:               summary,
		Strengths:             normalizeInsightList(parsed.Strengths, fallback.Strengths),
		AreasForImprovement:   normalizeInsightList(parsed.AreasForImprovement, fallback.AreasForImprovement),
		PsychologicalInsight:  psychologicalInsight,
		RawMetricTotal:        fallback.RawMetricTotal,
		RawMetricCompleted:    fallback.RawMetricCompleted,
		CompletionRatePercent: fallback.CompletionRatePercent,
		RawPointsDelta:        fallback.RawPointsDelta,
		ActiveDays:            fallback.ActiveDays,
		SubjectBreakdown:      fallback.SubjectBreakdown,
		AgenticPattern:        reportPatternSelection,
	}
}

func BuildPrompt(stats Stats) (string, error) {
	stats = normalizeStats(stats)
	payload, err := json.Marshal(stats)
	if err != nil {
		return "", fmt.Errorf("marshal %s insight stats: %w", stats.ReportType, err)
	}

	return fmt.Sprintf(`
You are an encouraging and supportive AI companion for a child.
Analyze the deterministic %s study metrics below and generate a positive %s report.

Deterministic metrics:
%s

Rules:
1. Use only the metrics above. Do not invent counts, subjects, points, habits, or events.
2. Keep the wording simple, concrete, and supportive for a primary school child.
3. Never scold, shame, or use negative criticism. Do not describe the child as failing, falling behind, or doing poorly.
4. Return exactly 3 strengths and exactly 3 areas_for_improvement.
5. Do not rewrite or modify any statistic. Your primary job is to explain the provided numbers, not to recalculate them.
6. Each areas_for_improvement item must be a forward-looking next-step suggestion, not a critique of past behavior.
7. Focus your summary on the achievements and the progress made. Do not emphasize what was missing or incomplete.
8. Your interpretation must be based strictly on the provided JSON data. Do not infer activities, moods, or external context not present in the metrics.

Return JSON only with this exact structure:
{
  "summary": "one friendly summary sentence or short paragraph addressed to the child",
  "strengths": ["strength 1", "strength 2", "strength 3"],
  "areas_for_improvement": ["improvement 1", "improvement 2", "improvement 3"],
  "psychological_insight": "one growth-mindset observation"
}
`, reportPromptName(stats.ReportType), reportPromptName(stats.ReportType), string(payload)), nil
}

func stripJSONFence(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
	}
	return strings.TrimSpace(trimmed)
}

func (s *Service) Generate(ctx context.Context, stats Stats) (Report, error) {
	normalizedStats := normalizeStats(stats)
	fallback := buildTemplateReport(normalizedStats)
	if s.llmClient == nil {
		return fallback, nil
	}

	prompt, err := BuildPrompt(normalizedStats)
	if err != nil {
		return fallback, nil
	}

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_INSIGHTS_MODEL_NAME",
		Temperature: 0.2,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a supportive study reflection agent. Return valid JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return fallback, nil
	}

	var parsed llmReport
	if err := json.Unmarshal([]byte(stripJSONFence(resultText)), &parsed); err != nil {
		return fallback, nil
	}

	return normalizeReportWithFallback(parsed, fallback), nil
}
