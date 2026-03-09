package weeklyinsights

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func generateMock(daysData []map[string]interface{}) Insight {
	totalTasks := 0
	completedTasks := 0

	for _, day := range daysData {
		dayTotal, dayCompleted := countTasks(day["tasks"])
		totalTasks += dayTotal
		completedTasks += dayCompleted
	}

	return Insight{
		Summary:              fmt.Sprintf("Great job this week! You tackled %d tasks and completed %d of them.", totalTasks, completedTasks),
		Strengths:            []string{"Consistent effort", "Ready to tackle new challenges"},
		AreasForImprovement:  []string{"Try to finish pending tasks before starting new ones"},
		PsychologicalInsight: "Your resilience is showing! Keep growing your brain by attempting hard things.",
		RawMetricTotal:       totalTasks,
		RawMetricCompleted:   completedTasks,
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

func buildPrompt(daysData []map[string]interface{}) (string, error) {
	payload, err := json.Marshal(daysData)
	if err != nil {
		return "", fmt.Errorf("marshal weekly data: %w", err)
	}

	return fmt.Sprintf(`
You are an encouraging and perceptive AI companion for a child.
Analyze their task completions over the past 7 days and generate a weekly report.

Task data:
%s

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
		Temperature: 0.4,
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

	if strings.TrimSpace(parsed.Summary) == "" {
		return mock, nil
	}

	parsed.RawMetricTotal = mock.RawMetricTotal
	parsed.RawMetricCompleted = mock.RawMetricCompleted
	parsed.AgenticPattern = weeklyPatternSelection
	return parsed, nil
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
