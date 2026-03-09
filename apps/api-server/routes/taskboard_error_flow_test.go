package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type routeErrorResponse struct {
	Error     string                 `json:"error"`
	ErrorCode string                 `json:"error_code"`
	Details   map[string]interface{} `json:"details"`
}

type weeklyStatsResponse struct {
	Message  string `json:"message"`
	RawStats []struct {
		Date string `json:"date"`
	} `json:"raw_stats"`
}

func decodeRouteErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) routeErrorResponse {
	t.Helper()

	var payload routeErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	return payload
}

func performRawJSONRequest(t *testing.T, router http.Handler, method, target, payload string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func detailString(t *testing.T, payload routeErrorResponse, key string) string {
	t.Helper()

	value, ok := payload.Details[key]
	if !ok {
		t.Fatalf("expected details[%q] to exist", key)
	}

	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("expected details[%q] to be a string, got %T", key, value)
	}

	return stringValue
}

func detailStringSlice(t *testing.T, payload routeErrorResponse, key string) []string {
	t.Helper()

	value, ok := payload.Details[key]
	if !ok {
		t.Fatalf("expected details[%q] to exist", key)
	}

	items, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected details[%q] to be an array, got %T", key, value)
	}

	converted := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("expected details[%q] items to be strings, got %T", key, item)
		}
		converted = append(converted, text)
	}

	return converted
}

func containsAll(values []string, expected ...string) bool {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	for _, value := range expected {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

func TestTaskCreateRejectsMissingRequiredFields(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()
	recorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks", map[string]interface{}{
		"family_id": 306,
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected POST /tasks to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", payload.ErrorCode)
	}
	fields := detailStringSlice(t, payload, "fields")
	if !containsAll(fields, "assignee_id", "content") {
		t.Fatalf("expected missing assignee_id/content fields, got %+v", fields)
	}
}

func TestTaskCreateRejectsMalformedJSON(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()
	recorder := performRawJSONRequest(t, router, http.MethodPost, "/api/v1/tasks", `{"family_id":306,"assignee_id":1,`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected malformed POST /tasks to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "invalid_json" {
		t.Fatalf("expected invalid_json, got %s", payload.ErrorCode)
	}
}

func TestListTasksRejectsInvalidDate(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()
	recorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-02-30", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected GET /tasks to return 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeRouteErrorResponse(t, recorder)
	if payload.ErrorCode != "invalid_date" {
		t.Fatalf("expected invalid_date, got %s", payload.ErrorCode)
	}
	if detailString(t, payload, "field") != "date" {
		t.Fatalf("expected invalid field date, got %+v", payload.Details)
	}
}

func TestTaskStatusHandlesNotFoundAndDuplicateUpdates(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local)
	seedMarch6DemoTasks(t, familyID, userID, date)

	router := SetupRouter()

	notFoundRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       99,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing task update to return 404, got %d: %s", notFoundRecorder.Code, notFoundRecorder.Body.String())
	}

	notFoundPayload := decodeRouteErrorResponse(t, notFoundRecorder)
	if notFoundPayload.ErrorCode != "task_not_found" {
		t.Fatalf("expected task_not_found, got %s", notFoundPayload.ErrorCode)
	}

	firstRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected first item update to return 200, got %d: %s", firstRecorder.Code, firstRecorder.Body.String())
	}

	firstPayload := decodeTaskBoardResponse(t, firstRecorder)
	if firstPayload.UpdatedCount != 1 {
		t.Fatalf("expected first item update count 1, got %d", firstPayload.UpdatedCount)
	}

	duplicateRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if duplicateRecorder.Code != http.StatusConflict {
		t.Fatalf("expected duplicate item update to return 409, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}

	duplicatePayload := decodeRouteErrorResponse(t, duplicateRecorder)
	if duplicatePayload.ErrorCode != "status_unchanged" {
		t.Fatalf("expected status_unchanged, got %s", duplicatePayload.ErrorCode)
	}
	if detailString(t, duplicatePayload, "status") != "completed" {
		t.Fatalf("expected duplicate status completed, got %+v", duplicatePayload.Details)
	}

	allRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/all", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if allRecorder.Code != http.StatusOK {
		t.Fatalf("expected bulk update to return 200, got %d: %s", allRecorder.Code, allRecorder.Body.String())
	}

	allPayload := decodeTaskBoardResponse(t, allRecorder)
	if allPayload.UpdatedCount != 8 {
		t.Fatalf("expected bulk update count 8, got %d", allPayload.UpdatedCount)
	}

	duplicateAllRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/all", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if duplicateAllRecorder.Code != http.StatusConflict {
		t.Fatalf("expected duplicate bulk update to return 409, got %d: %s", duplicateAllRecorder.Code, duplicateAllRecorder.Body.String())
	}

	duplicateAllPayload := decodeRouteErrorResponse(t, duplicateAllRecorder)
	if duplicateAllPayload.ErrorCode != "status_unchanged" {
		t.Fatalf("expected duplicate bulk status_unchanged, got %s", duplicateAllPayload.ErrorCode)
	}
}

func TestTaskGroupStatusRejectsDuplicateUpdate(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local)
	seedMarch6DemoTasks(t, familyID, userID, date)

	router := SetupRouter()

	firstRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"subject":       "英语",
		"group_title":   "预习M1U2",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected first group update to return 200, got %d: %s", firstRecorder.Code, firstRecorder.Body.String())
	}

	firstPayload := decodeTaskBoardResponse(t, firstRecorder)
	if firstPayload.UpdatedCount != 3 {
		t.Fatalf("expected first group updated_count 3, got %d", firstPayload.UpdatedCount)
	}

	duplicateRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"subject":       "英语",
		"group_title":   "预习M1U2",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if duplicateRecorder.Code != http.StatusConflict {
		t.Fatalf("expected duplicate group update to return 409, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}

	duplicatePayload := decodeRouteErrorResponse(t, duplicateRecorder)
	if duplicatePayload.ErrorCode != "status_unchanged" {
		t.Fatalf("expected status_unchanged, got %s", duplicatePayload.ErrorCode)
	}
	if detailString(t, duplicatePayload, "status") != "completed" {
		t.Fatalf("expected duplicate group status completed, got %+v", duplicatePayload.Details)
	}
	if detailString(t, duplicatePayload, "subject") != "英语" {
		t.Fatalf("expected duplicate group subject 英语, got %+v", duplicatePayload.Details)
	}
	if detailString(t, duplicatePayload, "group_title") != "预习M1U2" {
		t.Fatalf("expected duplicate group title 预习M1U2, got %+v", duplicatePayload.Details)
	}
}

func TestEmptyTaskBoardStatusUpdatesReturnNotFound(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()

	itemRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if itemRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected empty-board item update to return 404, got %d: %s", itemRecorder.Code, itemRecorder.Body.String())
	}

	itemPayload := decodeRouteErrorResponse(t, itemRecorder)
	if itemPayload.ErrorCode != "task_not_found" {
		t.Fatalf("expected task_not_found, got %s", itemPayload.ErrorCode)
	}

	groupRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"subject":       "数学",
		"group_title":   "校本P14-15",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if groupRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected empty-board group update to return 404, got %d: %s", groupRecorder.Code, groupRecorder.Body.String())
	}

	groupPayload := decodeRouteErrorResponse(t, groupRecorder)
	if groupPayload.ErrorCode != "task_group_not_found" {
		t.Fatalf("expected task_group_not_found, got %s", groupPayload.ErrorCode)
	}

	allRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/all", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if allRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected empty-board all update to return 404, got %d: %s", allRecorder.Code, allRecorder.Body.String())
	}

	allPayload := decodeRouteErrorResponse(t, allRecorder)
	if allPayload.ErrorCode != "task_not_found" {
		t.Fatalf("expected task_not_found, got %s", allPayload.ErrorCode)
	}
}

func TestTaskAndStatsQueryValidation(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_WEEKLY_MODEL_NAME", "")

	router := SetupRouter()

	missingTaskRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306", nil)
	if missingTaskRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing task query params to return 400, got %d: %s", missingTaskRecorder.Code, missingTaskRecorder.Body.String())
	}

	missingTaskPayload := decodeRouteErrorResponse(t, missingTaskRecorder)
	if missingTaskPayload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", missingTaskPayload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, missingTaskPayload, "fields"), "user_id") {
		t.Fatalf("expected missing task fields to include user_id, got %+v", missingTaskPayload.Details)
	}

	invalidTaskRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=bad&user_id=1", nil)
	if invalidTaskRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid task query params to return 400, got %d: %s", invalidTaskRecorder.Code, invalidTaskRecorder.Body.String())
	}

	invalidTaskPayload := decodeRouteErrorResponse(t, invalidTaskRecorder)
	if invalidTaskPayload.ErrorCode != "invalid_query_parameter" {
		t.Fatalf("expected invalid_query_parameter, got %s", invalidTaskPayload.ErrorCode)
	}
	if detailString(t, invalidTaskPayload, "field") != "family_id" {
		t.Fatalf("expected invalid task field family_id, got %+v", invalidTaskPayload.Details)
	}

	invalidStatsRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/weekly?family_id=306&user_id=bad", nil)
	if invalidStatsRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid stats query params to return 400, got %d: %s", invalidStatsRecorder.Code, invalidStatsRecorder.Body.String())
	}

	invalidStatsPayload := decodeRouteErrorResponse(t, invalidStatsRecorder)
	if invalidStatsPayload.ErrorCode != "invalid_query_parameter" {
		t.Fatalf("expected invalid_query_parameter, got %s", invalidStatsPayload.ErrorCode)
	}
	if detailString(t, invalidStatsPayload, "field") != "user_id" {
		t.Fatalf("expected invalid stats field user_id, got %+v", invalidStatsPayload.Details)
	}
}

func TestWeeklyStatsValidationAndAnchoredWindow(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_WEEKLY_MODEL_NAME", "")

	router := SetupRouter()

	missingRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/weekly?family_id=306", nil)
	if missingRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing stats params to return 400, got %d: %s", missingRecorder.Code, missingRecorder.Body.String())
	}

	missingPayload := decodeRouteErrorResponse(t, missingRecorder)
	if missingPayload.ErrorCode != "missing_required_fields" {
		t.Fatalf("expected missing_required_fields, got %s", missingPayload.ErrorCode)
	}
	if !containsAll(detailStringSlice(t, missingPayload, "fields"), "user_id") {
		t.Fatalf("expected user_id in missing fields, got %+v", missingPayload.Details)
	}

	invalidDateRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/weekly?family_id=306&user_id=1&end_date=2026-02-30", nil)
	if invalidDateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid stats end_date to return 400, got %d: %s", invalidDateRecorder.Code, invalidDateRecorder.Body.String())
	}

	invalidDatePayload := decodeRouteErrorResponse(t, invalidDateRecorder)
	if invalidDatePayload.ErrorCode != "invalid_date" {
		t.Fatalf("expected invalid_date, got %s", invalidDatePayload.ErrorCode)
	}
	if detailString(t, invalidDatePayload, "field") != "end_date" {
		t.Fatalf("expected invalid field end_date, got %+v", invalidDatePayload.Details)
	}

	seedMarch6DemoTasks(t, 306, 1, time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local))

	windowRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/weekly?family_id=306&user_id=1&end_date=2026-03-06", nil)
	if windowRecorder.Code != http.StatusOK {
		t.Fatalf("expected anchored stats request to return 200, got %d: %s", windowRecorder.Code, windowRecorder.Body.String())
	}

	var payload weeklyStatsResponse
	if err := json.Unmarshal(windowRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(payload.RawStats) != 1 || payload.RawStats[0].Date != "2026-03-06" {
		t.Fatalf("expected one anchored raw_stats entry for 2026-03-06, got %+v", payload.RawStats)
	}
}
