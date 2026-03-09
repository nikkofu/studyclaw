package weeklyinsights

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

var weeklyPatternSelection = agentic.PatternSelection{
	Primary:    "single-agent system",
	Supporting: []string{"custom logic pattern"},
	Why:        "Weekly insights require one bounded summarization call after deterministic aggregation, not a multi-agent orchestration loop.",
	Reference:  agentic.GoogleDesignPatternGuideURL,
}

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
	llmClient llm.Client
}

type promptDaySummary struct {
	Date           string `json:"date,omitempty"`
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
	PendingTasks   int    `json:"pending_tasks"`
}

type promptPayload struct {
	TotalTasks            int                `json:"total_tasks"`
	CompletedTasks        int                `json:"completed_tasks"`
	PendingTasks          int                `json:"pending_tasks"`
	ActiveDays            int                `json:"active_days"`
	CompletionRatePercent int                `json:"completion_rate_percent"`
	Days                  []promptDaySummary `json:"days"`
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func buildPromptPayload(daysData []map[string]interface{}) promptPayload {
	days := make([]promptDaySummary, 0, len(daysData))
	totalTasks := 0
	completedTasks := 0
	activeDays := 0

	for _, day := range daysData {
		dayTotal, dayCompleted := countTasks(day["tasks"])
		if dayTotal > 0 {
			activeDays++
		}
		totalTasks += dayTotal
		completedTasks += dayCompleted

		date, _ := day["date"].(string)
		days = append(days, promptDaySummary{
			Date:           strings.TrimSpace(date),
			TotalTasks:     dayTotal,
			CompletedTasks: dayCompleted,
			PendingTasks:   dayTotal - dayCompleted,
		})
	}

	sort.Slice(days, func(i, j int) bool {
		if days[i].Date == days[j].Date {
			return days[i].TotalTasks < days[j].TotalTasks
		}
		if days[i].Date == "" {
			return false
		}
		if days[j].Date == "" {
			return true
		}
		return days[i].Date < days[j].Date
	})

	completionRatePercent := 0
	if totalTasks > 0 {
		completionRatePercent = completedTasks * 100 / totalTasks
	}

	return promptPayload{
		TotalTasks:            totalTasks,
		CompletedTasks:        completedTasks,
		PendingTasks:          totalTasks - completedTasks,
		ActiveDays:            activeDays,
		CompletionRatePercent: completionRatePercent,
		Days:                  days,
	}
}

func generateMock(daysData []map[string]interface{}) Insight {
	payload := buildPromptPayload(daysData)

	return Insight{
		Summary:              buildMockSummary(payload),
		Strengths:            buildMockStrengths(payload),
		AreasForImprovement:  buildMockAreas(payload),
		PsychologicalInsight: buildMockPsychologicalInsight(payload),
		RawMetricTotal:       payload.TotalTasks,
		RawMetricCompleted:   payload.CompletedTasks,
		AgenticPattern:       weeklyPatternSelection,
	}
}

func countTasks(rawTasks any) (int, int) {
	switch typed := rawTasks.(type) {
	case []interface{}:
		completed := 0
		for _, item := range typed {
			if taskCompleted(item) {
				completed++
			}
		}
		return len(typed), completed
	case []map[string]interface{}:
		completed := 0
		for _, item := range typed {
			if taskCompleted(item) {
				completed++
			}
		}
		return len(typed), completed
	}

	value := reflect.ValueOf(rawTasks)
	if !value.IsValid() {
		return 0, 0
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, 0
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return 0, 0
	}

	completed := 0
	for index := 0; index < value.Len(); index++ {
		if taskCompleted(value.Index(index).Interface()) {
			completed++
		}
	}
	return value.Len(), completed
}

func taskCompleted(task any) bool {
	switch typed := task.(type) {
	case map[string]interface{}:
		completed, _ := typed["completed"].(bool)
		return completed
	}

	value := reflect.ValueOf(task)
	if !value.IsValid() {
		return false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return false
	}

	completedField := value.FieldByName("Completed")
	if !completedField.IsValid() || completedField.Kind() != reflect.Bool {
		return false
	}
	return completedField.Bool()
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

func buildMockSummary(payload promptPayload) string {
	switch {
	case payload.TotalTasks == 0:
		return "This week was light, which is a good chance to build a steady study rhythm for next week."
	case payload.CompletedTasks == payload.TotalTasks:
		return fmt.Sprintf("You completed all %d tasks this week, which shows steady follow-through from start to finish.", payload.TotalTasks)
	default:
		return fmt.Sprintf("You worked on %d tasks this week and completed %d of them, which means your effort is moving things forward.", payload.TotalTasks, payload.CompletedTasks)
	}
}

func buildMockStrengths(payload promptPayload) []string {
	candidates := []string{
		"You kept showing up for your homework instead of giving up.",
		"You made progress one task at a time.",
		"You are building a more reliable study routine.",
	}

	if payload.ActiveDays >= 3 {
		candidates = append([]string{"You spread your effort across multiple days instead of rushing everything at once."}, candidates...)
	}
	if payload.CompletionRatePercent >= 80 {
		candidates = append([]string{"You followed through on most of the work that was assigned."}, candidates...)
	}
	if payload.CompletedTasks == payload.TotalTasks && payload.TotalTasks > 0 {
		candidates = append([]string{"You finished the full list, which shows strong task completion habits."}, candidates...)
	}
	if payload.TotalTasks == 0 {
		candidates = append([]string{
			"You kept the week calm and ready for the next round of work.",
			"You had room to reset your routine without extra pressure.",
		}, candidates...)
	}

	return normalizeInsightList(candidates, nil)
}

func buildMockAreas(payload promptPayload) []string {
	candidates := []string{
		"Start the harder tasks a little earlier so they feel smaller.",
		"Finish older pending work before adding too many new tasks.",
		"Break bigger assignments into smaller steps and clear them in order.",
	}

	if payload.TotalTasks == 0 {
		candidates = []string{
			"Set up a simple homework rhythm before the next busy week starts.",
			"Pick one small study goal early so momentum is easier to build.",
			"Check the task list each day even when the workload is light.",
		}
	}
	if payload.CompletionRatePercent >= 80 && payload.TotalTasks > 0 {
		candidates = []string{
			"Keep using the same steady pace so strong weeks become a habit.",
			"Notice which study time helped you stay focused and repeat it.",
			"Use the same step-by-step approach on the next harder assignment.",
		}
	}

	return normalizeInsightList(candidates, nil)
}

func buildMockPsychologicalInsight(payload promptPayload) string {
	switch {
	case payload.TotalTasks == 0:
		return "A lighter week can still strengthen confidence when you use it to protect your routine."
	case payload.CompletionRatePercent >= 80:
		return "Your progress came from steady follow-through, which is a stronger habit than waiting to feel perfectly ready."
	default:
		return "Each finished task is evidence that persistence grows when you keep moving one step at a time."
	}
}

func normalizeInsightWithFallback(parsed Insight, fallback Insight) Insight {
	summary := trimInsightText(parsed.Summary)
	if summary == "" {
		summary = fallback.Summary
	}

	psychologicalInsight := trimInsightText(parsed.PsychologicalInsight)
	if psychologicalInsight == "" {
		psychologicalInsight = fallback.PsychologicalInsight
	}

	return Insight{
		Summary:              summary,
		Strengths:            normalizeInsightList(parsed.Strengths, fallback.Strengths),
		AreasForImprovement:  normalizeInsightList(parsed.AreasForImprovement, fallback.AreasForImprovement),
		PsychologicalInsight: psychologicalInsight,
		RawMetricTotal:       fallback.RawMetricTotal,
		RawMetricCompleted:   fallback.RawMetricCompleted,
		AgenticPattern:       weeklyPatternSelection,
	}
}

func buildPrompt(daysData []map[string]interface{}) (string, error) {
	payload, err := json.Marshal(buildPromptPayload(daysData))
	if err != nil {
		return "", fmt.Errorf("marshal weekly data: %w", err)
	}

	return fmt.Sprintf(`
You are an encouraging and perceptive AI companion for a child.
Analyze the deterministic weekly study metrics below and generate a weekly report.

Weekly metrics:
%s

Rules:
1. Use only the metrics and daily counts above. Do not invent subjects, habits, or events that are not supported by the data.
2. Keep the wording simple, concrete, and encouraging for a primary school child.
3. Return exactly 3 strengths and exactly 3 areas_for_improvement.

Return JSON only with this exact structure:
{
  "summary": "one friendly summary sentence or short paragraph addressed to the child",
  "strengths": ["strength 1", "strength 2", "strength 3"],
  "areas_for_improvement": ["improvement 1", "improvement 2", "improvement 3"],
  "psychological_insight": "one growth-mindset observation"
}
`, string(payload)), nil
}

func (s *Service) Generate(ctx context.Context, daysData []map[string]interface{}) (Insight, error) {
	mock := generateMock(daysData)
	if s.llmClient == nil {
		return mock, nil
	}

	prompt, err := buildPrompt(daysData)
	if err != nil {
		return mock, nil
	}

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_WEEKLY_MODEL_NAME",
		Temperature: 0.2,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a supportive weekly reflection agent. Return valid JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return mock, nil
	}

	var parsed Insight
	if err := json.Unmarshal([]byte(stripWeeklyJSONFence(resultText)), &parsed); err != nil {
		return mock, nil
	}

	return normalizeInsightWithFallback(parsed, mock), nil
}

func stripWeeklyJSONFence(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
	}
	return strings.TrimSpace(trimmed)
}
