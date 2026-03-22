package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func decodeObjectResponse(t *testing.T, recorderBody []byte) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(recorderBody, &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	return payload
}

func assertHasKeys(t *testing.T, payload map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected key %q in payload, got %+v", key, payload)
		}
	}
}

func TestHotTaskFlagsOff_PayloadUnchanged(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")
	t.Setenv("hot_task_launch_v1", "")
	t.Setenv("hot_task_resume_v1", "")
	t.Setenv("hot_task_rewards_v1", "")

	router := SetupRouter()

	draftRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/drafts/parse", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"source_text":   routeSampleGroupMessage,
	})
	if draftRecorder.Code != http.StatusCreated {
		t.Fatalf("expected draft parse to return 201, got %d: %s", draftRecorder.Code, draftRecorder.Body.String())
	}
	draftPayload := decodeObjectResponse(t, draftRecorder.Body.Bytes())
	assertHasKeys(t, draftPayload, "message", "daily_assignment_draft")
	draftBlock, ok := draftPayload["daily_assignment_draft"].(map[string]any)
	if !ok {
		t.Fatalf("expected daily_assignment_draft object, got %+v", draftPayload["daily_assignment_draft"])
	}
	draftID, _ := draftBlock["draft_id"].(string)

	publishRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/publish", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"draft_id":      draftID,
	})
	if publishRecorder.Code != http.StatusCreated {
		t.Fatalf("expected publish to return 201, got %d: %s", publishRecorder.Code, publishRecorder.Body.String())
	}
	publishPayload := decodeObjectResponse(t, publishRecorder.Body.Bytes())
	assertHasKeys(t, publishPayload, "message", "daily_assignment", "task_board")
	if _, ok := publishPayload["hot_task_flags"]; ok {
		t.Fatalf("expected no hot_task_flags in publish payload when flags off")
	}

	dayRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/daily-assignments?family_id=306&child_id=1&date=2026-03-16", nil)
	if dayRecorder.Code != http.StatusOK {
		t.Fatalf("expected daily assignment fetch to return 200, got %d: %s", dayRecorder.Code, dayRecorder.Body.String())
	}
	dayPayload := decodeObjectResponse(t, dayRecorder.Body.Bytes())
	assertHasKeys(t, dayPayload, "date", "published", "daily_assignment", "task_board", "points_balance")
	if _, ok := dayPayload["hot_task_flags"]; ok {
		t.Fatalf("expected no hot_task_flags in day payload when flags off")
	}

	taskBoard, ok := dayPayload["task_board"].(map[string]any)
	if !ok {
		t.Fatalf("expected task_board object, got %+v", dayPayload["task_board"])
	}
	if _, ok := taskBoard["launch_recommendation"]; ok {
		t.Fatalf("expected no launch_recommendation in task_board when flags off")
	}
}

func TestDailyAssignment_LaunchRecommendationContract(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")
	t.Setenv("hot_task_launch_v1", "true")
	t.Setenv("hot_task_resume_v1", "")
	t.Setenv("hot_task_rewards_v1", "")

	router := SetupRouter()

	draftRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/drafts/parse", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"source_text":   routeSampleGroupMessage,
	})
	if draftRecorder.Code != http.StatusCreated {
		t.Fatalf("expected draft parse to return 201, got %d: %s", draftRecorder.Code, draftRecorder.Body.String())
	}
	draftPayload := decodeObjectResponse(t, draftRecorder.Body.Bytes())
	draftBlock := draftPayload["daily_assignment_draft"].(map[string]any)
	draftID := draftBlock["draft_id"].(string)

	publishRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/publish", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"draft_id":      draftID,
	})
	if publishRecorder.Code != http.StatusCreated {
		t.Fatalf("expected publish to return 201, got %d: %s", publishRecorder.Code, publishRecorder.Body.String())
	}

	dayRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/daily-assignments?family_id=306&child_id=1&date=2026-03-16", nil)
	if dayRecorder.Code != http.StatusOK {
		t.Fatalf("expected daily assignment fetch to return 200, got %d: %s", dayRecorder.Code, dayRecorder.Body.String())
	}
	dayPayload := decodeObjectResponse(t, dayRecorder.Body.Bytes())
	taskBoard := dayPayload["task_board"].(map[string]any)
	launchRecommendation, ok := taskBoard["launch_recommendation"].(map[string]any)
	if !ok {
		t.Fatalf("expected launch_recommendation object when launch flag on, got %+v", taskBoard["launch_recommendation"])
	}

	reasonCode, ok := launchRecommendation["reason_code"].(string)
	if !ok || reasonCode == "" {
		t.Fatalf("expected non-empty reason_code string, got %+v", launchRecommendation["reason_code"])
	}
	if reasonCode != "first_unfinished" {
		t.Fatalf("expected reason_code first_unfinished, got %q", reasonCode)
	}

	groupID, ok := launchRecommendation["group_id"].(string)
	if !ok || groupID == "" {
		t.Fatalf("expected non-empty group_id string, got %+v", launchRecommendation["group_id"])
	}
	if !strings.Contains(groupID, "\x00") {
		t.Fatalf("expected group_id in canonical <subject>\\x00<group_title> format, got %q", groupID)
	}

	itemValue, hasItemID := launchRecommendation["item_id"]
	if !hasItemID {
		t.Fatalf("expected item_id key to exist")
	}
	if itemValue != nil {
		if _, ok := itemValue.(float64); !ok {
			t.Fatalf("expected item_id to be number or null, got %T (%+v)", itemValue, itemValue)
		}
	}

	if value, ok := launchRecommendation["why_recommended"]; ok {
		if _, isString := value.(string); !isString {
			t.Fatalf("expected why_recommended to be omitted or string, got %T", value)
		}
	}
}

func TestFrozenSuccessFieldsForTaskboardRoutes(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()

	pingRecorder := performJSONRequest(t, router, http.MethodGet, "/ping", nil)
	if pingRecorder.Code != http.StatusOK {
		t.Fatalf("expected ping to return 200, got %d: %s", pingRecorder.Code, pingRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, pingRecorder.Body.Bytes()), "message")

	createRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks", map[string]any{
		"family_id":     306,
		"assignee_id":   1,
		"subject":       "数学",
		"group_title":   "校本P14-15",
		"content":       "校本P14-15",
		"assigned_date": "2026-03-12",
	})
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create to return 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, createRecorder.Body.Bytes()), "message", "date", "task")

	parseRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/parse", map[string]any{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-12",
		"auto_create":   false,
		"raw_text":      routeSampleGroupMessage,
	})
	if parseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected parse to return 201, got %d: %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, parseRecorder.Body.Bytes()), "message", "parsed_count", "parser_mode", "analysis", "auto_created", "date", "tasks")

	confirmRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/confirm", map[string]any{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-12",
		"tasks": []map[string]any{
			{
				"subject":     "英语",
				"group_title": "预习M1U2",
				"title":       "书本上标注好黄页单词的音标",
			},
		},
	})
	if confirmRecorder.Code != http.StatusCreated {
		t.Fatalf("expected confirm to return 201, got %d: %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, confirmRecorder.Body.Bytes()), "message", "created_count", "date", "tasks")

	listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-12", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, listRecorder.Body.Bytes()), "date", "tasks", "groups", "homework_groups", "summary")
}

func TestFrozenSuccessFieldsForStatusRoutes(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local)
	seedMarch6DemoTasks(t, familyID, userID, date)

	router := SetupRouter()

	itemRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]any{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if itemRecorder.Code != http.StatusOK {
		t.Fatalf("expected item update to return 200, got %d: %s", itemRecorder.Code, itemRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, itemRecorder.Body.Bytes()), "message", "updated_count", "date", "tasks", "groups", "homework_groups", "summary")

	groupRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]any{
		"family_id":     familyID,
		"assignee_id":   userID,
		"subject":       "英语",
		"group_title":   "预习M1U2",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if groupRecorder.Code != http.StatusOK {
		t.Fatalf("expected group update to return 200, got %d: %s", groupRecorder.Code, groupRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, groupRecorder.Body.Bytes()), "message", "updated_count", "subject", "group_title", "date", "tasks", "groups", "homework_groups", "summary")

	allRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/all", map[string]any{
		"family_id":     familyID,
		"assignee_id":   userID,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if allRecorder.Code != http.StatusOK {
		t.Fatalf("expected all update to return 200, got %d: %s", allRecorder.Code, allRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, allRecorder.Body.Bytes()), "message", "updated_count", "date", "tasks", "groups", "homework_groups", "summary")
}

func TestFrozenSuccessFieldsForPhaseOneRoutes(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()

	draftRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/drafts/parse", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"source_text":   routeSampleGroupMessage,
	})
	if draftRecorder.Code != http.StatusCreated {
		t.Fatalf("expected draft parse to return 201, got %d: %s", draftRecorder.Code, draftRecorder.Body.String())
	}
	draftPayload := decodeObjectResponse(t, draftRecorder.Body.Bytes())
	assertHasKeys(t, draftPayload, "message", "daily_assignment_draft")

	draftBlock, ok := draftPayload["daily_assignment_draft"].(map[string]any)
	if !ok {
		t.Fatalf("expected daily_assignment_draft object, got %+v", draftPayload["daily_assignment_draft"])
	}
	draftID, ok := draftBlock["draft_id"].(string)
	if !ok || draftID == "" {
		t.Fatalf("expected non-empty draft_id, got %+v", draftBlock)
	}

	publishRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/publish", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"draft_id":      draftID,
	})
	if publishRecorder.Code != http.StatusCreated {
		t.Fatalf("expected publish to return 201, got %d: %s", publishRecorder.Code, publishRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, publishRecorder.Body.Bytes()), "message", "daily_assignment", "task_board")

	dayRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/daily-assignments?family_id=306&child_id=1&date=2026-03-16", nil)
	if dayRecorder.Code != http.StatusOK {
		t.Fatalf("expected daily assignment fetch to return 200, got %d: %s", dayRecorder.Code, dayRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, dayRecorder.Body.Bytes()), "date", "published", "daily_assignment", "task_board", "points_balance")

	ledgerRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/points/ledger", map[string]any{
		"family_id":   306,
		"user_id":     1,
		"delta":       2,
		"source_type": "parent_reward",
		"occurred_on": "2026-03-16",
		"note":        "smoke",
	})
	if ledgerRecorder.Code != http.StatusCreated {
		t.Fatalf("expected points ledger create to return 201, got %d: %s", ledgerRecorder.Code, ledgerRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, ledgerRecorder.Body.Bytes()), "message", "points_entry", "points_balance")

	wordListRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/word-lists", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"title":         "英语 Day 1",
		"language":      "en",
		"items": []map[string]any{
			{"text": "apple"},
		},
	})
	if wordListRecorder.Code != http.StatusCreated {
		t.Fatalf("expected word list create to return 201, got %d: %s", wordListRecorder.Code, wordListRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, wordListRecorder.Body.Bytes()), "message", "word_list")

	sessionRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
	})
	if sessionRecorder.Code != http.StatusCreated {
		t.Fatalf("expected dictation session start to return 201, got %d: %s", sessionRecorder.Code, sessionRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, sessionRecorder.Body.Bytes()), "message", "dictation_session")

	dailyStatsRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/daily?family_id=306&user_id=1&date=2026-03-16", nil)
	if dailyStatsRecorder.Code != http.StatusOK {
		t.Fatalf("expected daily stats to return 200, got %d: %s", dailyStatsRecorder.Code, dailyStatsRecorder.Body.String())
	}
	assertHasKeys(t, decodeObjectResponse(t, dailyStatsRecorder.Body.Bytes()), "period", "start_date", "end_date", "totals", "subject_breakdown", "completion_series", "points_series", "word_series", "encouragement")
}

func TestFrozenPhaseOneQueryAndActionRoutes(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()
	familyID := uint(306)
	userID := uint(1)
	dateStr := "2026-03-16"

	// Seed some data first via POST
	performJSONRequest(t, router, http.MethodPost, "/api/v1/points/ledger", map[string]any{
		"family_id":   familyID,
		"user_id":     userID,
		"delta":       5,
		"source_type": "parent_reward",
		"occurred_on": dateStr,
	})
	performJSONRequest(t, router, http.MethodPost, "/api/v1/word-lists", map[string]any{
		"family_id":     familyID,
		"child_id":      userID,
		"assigned_date": dateStr,
		"title":         "Unit 1",
		"language":      "en",
		"items":         []any{map[string]any{"text": "apple"}},
	})
	sessionRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     familyID,
		"child_id":      userID,
		"assigned_date": dateStr,
	})
	sessionPayload := decodeObjectResponse(t, sessionRecorder.Body.Bytes())
	sessionObj := sessionPayload["dictation_session"].(map[string]any)
	sessionID := sessionObj["session_id"].(string)

	// Test GET /stats/weekly
	weeklyRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/weekly?family_id=306&user_id=1&end_date="+dateStr, nil)
	if weeklyRecorder.Code != http.StatusOK {
		t.Fatalf("expected weekly stats to return 200, got %d", weeklyRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, weeklyRecorder.Body.Bytes()), "period", "start_date", "end_date", "totals", "completion_series")

	// Test GET /stats/monthly
	monthlyRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03", nil)
	if monthlyRecorder.Code != http.StatusOK {
		t.Fatalf("expected monthly stats to return 200, got %d", monthlyRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, monthlyRecorder.Body.Bytes()), "period", "start_date", "end_date", "totals", "completion_series")

	// Test GET /points/balance
	balanceRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/points/balance?family_id=306&user_id=1&date="+dateStr, nil)
	if balanceRecorder.Code != http.StatusOK {
		t.Fatalf("expected points balance to return 200, got %d", balanceRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, balanceRecorder.Body.Bytes()), "points_balance")

	// Test GET /points/ledger
	ledgerListRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/points/ledger?family_id=306&user_id=1&start_date="+dateStr+"&end_date="+dateStr, nil)
	if ledgerListRecorder.Code != http.StatusOK {
		t.Fatalf("expected points ledger list to return 200, got %d", ledgerListRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, ledgerListRecorder.Body.Bytes()), "entries", "points_balance")

	// Test GET /word-lists
	wordListGetRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/word-lists?family_id=306&child_id=1&date="+dateStr, nil)
	if wordListGetRecorder.Code != http.StatusOK {
		t.Fatalf("expected word list fetch to return 200, got %d", wordListGetRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, wordListGetRecorder.Body.Bytes()), "word_list")

	// Test GET /dictation-sessions/:session_id
	sessionGetRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/dictation-sessions/"+sessionID, nil)
	if sessionGetRecorder.Code != http.StatusOK {
		t.Fatalf("expected session fetch to return 200, got %d", sessionGetRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, sessionGetRecorder.Body.Bytes()), "dictation_session")

	recitationRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/recitation/analyze", map[string]any{
		"transcript":     "江畔独步寻花杜甫黄师塔前江水东春光懒困倚微风",
		"scene":          "recitation",
		"reference_text": "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。",
	})
	if recitationRecorder.Code != http.StatusOK {
		t.Fatalf("expected recitation analyze to return 200, got %d", recitationRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, recitationRecorder.Body.Bytes()), "analysis")

	// Test POST /dictation-sessions/:session_id/replay
	replayRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+sessionID+"/replay", nil)
	if replayRecorder.Code != http.StatusOK {
		t.Fatalf("expected session replay to return 200, got %d", replayRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, replayRecorder.Body.Bytes()), "message", "dictation_session")

	// Test POST /dictation-sessions/:session_id/next
	nextRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+sessionID+"/next", nil)
	if nextRecorder.Code != http.StatusOK {
		t.Fatalf("expected session next to return 200, got %d", nextRecorder.Code)
	}
	assertHasKeys(t, decodeObjectResponse(t, nextRecorder.Body.Bytes()), "message", "dictation_session")
}
