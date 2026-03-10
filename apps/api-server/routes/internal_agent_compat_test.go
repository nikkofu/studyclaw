package routes

import (
	"encoding/json"
	"net/http"
	"testing"
)

type internalWeeklyResponse struct {
	Summary            string `json:"summary"`
	RawMetricTotal     int    `json:"raw_metric_total"`
	RawMetricCompleted int    `json:"raw_metric_completed"`
	AgenticPattern     struct {
		Primary string `json:"primary"`
	} `json:"agentic_pattern"`
}

type internalParseResponse struct {
	Status     string `json:"status"`
	ParserMode string `json:"parser_mode"`
	Analysis   struct {
		TaskCount        int `json:"task_count"`
		NeedsReviewCount int `json:"needs_review_count"`
	} `json:"analysis"`
	Data []struct {
		Subject     string `json:"subject"`
		GroupTitle  string `json:"group_title"`
		Title       string `json:"title"`
		NeedsReview bool   `json:"needs_review"`
	} `json:"data"`
}

func TestInternalParseEndpointCompatibility(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()
	recorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/internal/parse", map[string]interface{}{
		"raw_text": routeSampleGroupMessage,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected internal parse to return 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var parsePayload internalParseResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &parsePayload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if parsePayload.ParserMode != "rule_fallback" {
		t.Fatalf("expected rule_fallback parser mode, got %s", parsePayload.ParserMode)
	}
	if len(parsePayload.Data) != 9 {
		t.Fatalf("expected 9 parsed tasks, got %d", len(parsePayload.Data))
	}
}

func TestInternalWeeklyEndpointCompatibility(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_WEEKLY_MODEL_NAME", "")

	router := SetupRouter()
	recorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/internal/analyze/weekly", map[string]interface{}{
		"days_data": []map[string]interface{}{
			{
				"date": "2026-03-10",
				"tasks": []map[string]interface{}{
					{"completed": true},
					{"completed": false},
					{"completed": true},
				},
			},
		},
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected internal weekly endpoint to return 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload internalWeeklyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if payload.RawMetricTotal != 3 || payload.RawMetricCompleted != 2 {
		t.Fatalf("unexpected weekly metrics: %+v", payload)
	}
	if payload.AgenticPattern.Primary == "" {
		t.Fatalf("unexpected agentic pattern: %+v", payload.AgenticPattern)
	}
}
