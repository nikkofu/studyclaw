package routes

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPingSmokeEndpoint(t *testing.T) {
	router := SetupRouter()

	recorder := performJSONRequest(t, router, http.MethodGet, "/ping", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected ping to return 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if payload["message"] != "pong" {
		t.Fatalf("expected ping message pong, got %+v", payload)
	}
}

func TestAuthLoginValidationUsesSharedErrorContract(t *testing.T) {
	router := SetupRouter()

	recorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected auth login validation to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", payload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, payload, "fields"), "phone", "password") {
		t.Fatalf("expected missing auth fields phone/password, got %+v", payload.Details)
	}
}

func TestPointsValidationUsesSharedErrorContract(t *testing.T) {
	router := SetupRouter()

	recorder := performRawJSONRequest(t, router, http.MethodPost, "/api/v1/points/update", `{"user_id":1,`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected points validation to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "invalid_json" {
		t.Fatalf("expected invalid_json, got %s", payload.ErrorCode)
	}
}

func TestInternalHandlersValidationUseSharedErrorContract(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")
	t.Setenv("LLM_WEEKLY_MODEL_NAME", "")

	router := SetupRouter()

	parseRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/internal/parse", map[string]interface{}{})
	if parseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected internal parse validation to return 400, got %d: %s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parsePayload := decodeRouteErrorResponse(t, parseRecorder)
	if parsePayload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", parsePayload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, parsePayload, "fields"), "raw_text") {
		t.Fatalf("expected missing raw_text field, got %+v", parsePayload.Details)
	}

	weeklyRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/internal/analyze/weekly", map[string]interface{}{})
	if weeklyRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected internal weekly validation to return 400, got %d: %s", weeklyRecorder.Code, weeklyRecorder.Body.String())
	}

	weeklyPayload := decodeRouteErrorResponse(t, weeklyRecorder)
	if weeklyPayload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", weeklyPayload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, weeklyPayload, "fields"), "days_data") {
		t.Fatalf("expected missing days_data field, got %+v", weeklyPayload.Details)
	}
}

func TestRecitationAnalysisValidationUsesSharedErrorContract(t *testing.T) {
	router := SetupRouter()

	recorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/recitation/analyze", map[string]interface{}{})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected recitation analyze validation to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", payload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, payload, "fields"), "transcript") {
		t.Fatalf("expected missing transcript field, got %+v", payload.Details)
	}
}
