package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

type draftParseResponse struct {
	Message string `json:"message"`
	Draft   struct {
		DraftID      string `json:"draft_id"`
		AssignedDate string `json:"assigned_date"`
		Status       string `json:"status"`
		Summary      struct {
			TotalTasks int `json:"total_tasks"`
		} `json:"summary"`
		TaskItems []struct {
			Title string `json:"title"`
		} `json:"task_items"`
	} `json:"daily_assignment_draft"`
}

type publishAssignmentResponse struct {
	Message         string `json:"message"`
	DailyAssignment struct {
		AssignmentID string `json:"assignment_id"`
		AssignedDate string `json:"assigned_date"`
		Status       string `json:"status"`
		Summary      struct {
			TotalTasks int `json:"total_tasks"`
		} `json:"summary"`
		TaskItems []struct {
			Title string `json:"title"`
		} `json:"task_items"`
	} `json:"daily_assignment"`
	TaskBoard taskBoardResponse `json:"task_board"`
}

type dayBundleResponse struct {
	Date            string `json:"date"`
	Published       bool   `json:"published"`
	DailyAssignment struct {
		AssignmentID string `json:"assignment_id"`
		Status       string `json:"status"`
	} `json:"daily_assignment"`
	TaskBoard     taskBoardResponse `json:"task_board"`
	PointsBalance struct {
		Balance int `json:"balance"`
	} `json:"points_balance"`
}

type pointsLedgerResponse struct {
	Entries []struct {
		EntryID      string `json:"entry_id"`
		Delta        int    `json:"delta"`
		SourceType   string `json:"source_type"`
		SourceOrigin string `json:"source_origin"`
	} `json:"entries"`
	PointsBalance struct {
		Balance      int `json:"balance"`
		AutoPoints   int `json:"auto_points"`
		ManualPoints int `json:"manual_points"`
	} `json:"points_balance"`
}

type wordListResponse struct {
	WordList struct {
		WordListID string `json:"word_list_id"`
		Language   string `json:"language"`
		TotalItems int    `json:"total_items"`
		Items      []struct {
			Text string `json:"text"`
		} `json:"items"`
	} `json:"word_list"`
}

type dictationSessionResponse struct {
	Message string `json:"message"`
	Session struct {
		SessionID     string `json:"session_id"`
		Status        string `json:"status"`
		GradingStatus string `json:"grading_status"`
		GradingError  string `json:"grading_error"`
		DebugContext  *struct {
			PhotoSHA1   string   `json:"photo_sha1"`
			PhotoBytes  int      `json:"photo_bytes"`
			Language    string   `json:"language"`
			Mode        string   `json:"mode"`
			WorkerStage string   `json:"worker_stage"`
			LogFile     string   `json:"log_file"`
			LogKeywords []string `json:"log_keywords"`
		} `json:"debug_context"`
		CurrentIndex   int `json:"current_index"`
		PlayedCount    int `json:"played_count"`
		CompletedItems int `json:"completed_items"`
		GradingResult  *struct {
			Score int `json:"score"`
		} `json:"grading_result"`
		CurrentItem *struct {
			Text string `json:"text"`
		} `json:"current_item"`
	} `json:"dictation_session"`
}

type dictationSessionListResponse struct {
	Sessions []struct {
		SessionID     string `json:"session_id"`
		AssignedDate  string `json:"assigned_date"`
		GradingStatus string `json:"grading_status"`
		GradingError  string `json:"grading_error"`
		DebugContext  *struct {
			PhotoSHA1   string   `json:"photo_sha1"`
			WorkerStage string   `json:"worker_stage"`
			LogFile     string   `json:"log_file"`
			LogKeywords []string `json:"log_keywords"`
		} `json:"debug_context"`
		GradingResult *struct {
			Score int `json:"score"`
		} `json:"grading_result"`
	} `json:"dictation_sessions"`
}

type statsResponse struct {
	Period string `json:"period"`
	Totals struct {
		TotalTasks         int `json:"total_tasks"`
		CompletedTasks     int `json:"completed_tasks"`
		WordItems          int `json:"word_items"`
		CompletedWordItems int `json:"completed_word_items"`
		PointsBalance      int `json:"points_balance"`
	} `json:"totals"`
	CompletionSeries []struct {
		Date string `json:"date"`
	} `json:"completion_series"`
	WordSeries []struct {
		Date string `json:"date"`
	} `json:"word_series"`
}

func TestPhaseOneDailyAssignmentPublishAndPadFetch(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()

	parseRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/drafts/parse", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-13",
		"source_text":   routeSampleGroupMessage,
	})
	if parseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected draft parse to return 201, got %d: %s", parseRecorder.Code, parseRecorder.Body.String())
	}

	var parsePayload draftParseResponse
	if err := json.Unmarshal(parseRecorder.Body.Bytes(), &parsePayload); err != nil {
		t.Fatalf("unmarshal draft parse response: %v", err)
	}
	if parsePayload.Draft.DraftID == "" || parsePayload.Draft.Status != "draft" {
		t.Fatalf("unexpected draft payload: %+v", parsePayload.Draft)
	}
	if parsePayload.Draft.Summary.TotalTasks != 9 || len(parsePayload.Draft.TaskItems) != 9 {
		t.Fatalf("unexpected parsed draft task summary: %+v", parsePayload.Draft)
	}

	publishRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/publish", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-13",
		"draft_id":      parsePayload.Draft.DraftID,
	})
	if publishRecorder.Code != http.StatusCreated {
		t.Fatalf("expected publish to return 201, got %d: %s", publishRecorder.Code, publishRecorder.Body.String())
	}

	var publishPayload publishAssignmentResponse
	if err := json.Unmarshal(publishRecorder.Body.Bytes(), &publishPayload); err != nil {
		t.Fatalf("unmarshal publish response: %v", err)
	}
	if publishPayload.DailyAssignment.AssignmentID == "" || publishPayload.DailyAssignment.Status != "published" {
		t.Fatalf("unexpected published assignment payload: %+v", publishPayload.DailyAssignment)
	}
	if publishPayload.TaskBoard.Summary.Total != 9 || publishPayload.DailyAssignment.Summary.TotalTasks != 9 {
		t.Fatalf("unexpected published board summary: %+v / %+v", publishPayload.TaskBoard.Summary, publishPayload.DailyAssignment.Summary)
	}

	dayRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/daily-assignments?family_id=306&child_id=1&date=2026-03-13", nil)
	if dayRecorder.Code != http.StatusOK {
		t.Fatalf("expected day bundle to return 200, got %d: %s", dayRecorder.Code, dayRecorder.Body.String())
	}

	var dayPayload dayBundleResponse
	if err := json.Unmarshal(dayRecorder.Body.Bytes(), &dayPayload); err != nil {
		t.Fatalf("unmarshal day bundle response: %v", err)
	}
	if !dayPayload.Published || dayPayload.Date != "2026-03-13" {
		t.Fatalf("unexpected day bundle header: %+v", dayPayload)
	}
	if dayPayload.TaskBoard.Summary.Total != 9 {
		t.Fatalf("expected task board total 9, got %+v", dayPayload.TaskBoard.Summary)
	}
}

func TestPhaseOnePointsLedgerAndBalanceFlow(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 14, 8, 0, 0, 0, time.Local)
	seedMarch6DemoTasks(t, familyID, userID, date)

	router := SetupRouter()

	itemRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]any{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-14",
	})
	if itemRecorder.Code != http.StatusOK {
		t.Fatalf("expected item status update to return 200, got %d: %s", itemRecorder.Code, itemRecorder.Body.String())
	}

	ledgerCreateRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/points/ledger", map[string]any{
		"family_id":   familyID,
		"user_id":     userID,
		"delta":       2,
		"source_type": "parent_reward",
		"occurred_on": "2026-03-14",
		"note":        "主动完成额外练习",
	})
	if ledgerCreateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected ledger create to return 201, got %d: %s", ledgerCreateRecorder.Code, ledgerCreateRecorder.Body.String())
	}

	ledgerRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/points/ledger?family_id=306&user_id=1&start_date=2026-03-14&end_date=2026-03-14", nil)
	if ledgerRecorder.Code != http.StatusOK {
		t.Fatalf("expected ledger list to return 200, got %d: %s", ledgerRecorder.Code, ledgerRecorder.Body.String())
	}

	var ledgerPayload pointsLedgerResponse
	if err := json.Unmarshal(ledgerRecorder.Body.Bytes(), &ledgerPayload); err != nil {
		t.Fatalf("unmarshal ledger response: %v", err)
	}
	if len(ledgerPayload.Entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %+v", ledgerPayload.Entries)
	}
	if ledgerPayload.PointsBalance.Balance != 3 || ledgerPayload.PointsBalance.AutoPoints != 1 || ledgerPayload.PointsBalance.ManualPoints != 2 {
		t.Fatalf("unexpected points balance: %+v", ledgerPayload.PointsBalance)
	}
}

func TestPhaseOneWordListDictationAndStatsFlow(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 15, 8, 0, 0, 0, time.Local)
	seedMarch6DemoTasks(t, familyID, userID, date)

	router := SetupRouter()

	statusRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]any{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-15",
	})
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("expected seed status update to return 200, got %d: %s", statusRecorder.Code, statusRecorder.Body.String())
	}

	wordListRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/word-lists", map[string]any{
		"family_id":     familyID,
		"child_id":      userID,
		"assigned_date": "2026-03-15",
		"title":         "英语默写 Day 1",
		"language":      "en",
		"items": []map[string]any{
			{"text": "apple", "meaning": "苹果"},
			{"text": "orange", "meaning": "橙子"},
			{"text": "banana", "meaning": "香蕉"},
		},
	})
	if wordListRecorder.Code != http.StatusCreated {
		t.Fatalf("expected word list create to return 201, got %d: %s", wordListRecorder.Code, wordListRecorder.Body.String())
	}

	var wordListPayload wordListResponse
	if err := json.Unmarshal(wordListRecorder.Body.Bytes(), &wordListPayload); err != nil {
		t.Fatalf("unmarshal word list response: %v", err)
	}
	if wordListPayload.WordList.WordListID == "" || wordListPayload.WordList.TotalItems != 3 {
		t.Fatalf("unexpected word list payload: %+v", wordListPayload.WordList)
	}

	startRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     familyID,
		"child_id":      userID,
		"assigned_date": "2026-03-15",
	})
	if startRecorder.Code != http.StatusCreated {
		t.Fatalf("expected session start to return 201, got %d: %s", startRecorder.Code, startRecorder.Body.String())
	}

	var startPayload dictationSessionResponse
	if err := json.Unmarshal(startRecorder.Body.Bytes(), &startPayload); err != nil {
		t.Fatalf("unmarshal session response: %v", err)
	}
	if startPayload.Session.SessionID == "" || startPayload.Session.CurrentItem == nil || startPayload.Session.CurrentItem.Text != "apple" {
		t.Fatalf("unexpected started session: %+v", startPayload.Session)
	}

	replayRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+startPayload.Session.SessionID+"/replay", nil)
	if replayRecorder.Code != http.StatusOK {
		t.Fatalf("expected replay to return 200, got %d: %s", replayRecorder.Code, replayRecorder.Body.String())
	}

	nextRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+startPayload.Session.SessionID+"/next", nil)
	if nextRecorder.Code != http.StatusOK {
		t.Fatalf("expected next to return 200, got %d: %s", nextRecorder.Code, nextRecorder.Body.String())
	}

	var nextPayload dictationSessionResponse
	if err := json.Unmarshal(nextRecorder.Body.Bytes(), &nextPayload); err != nil {
		t.Fatalf("unmarshal next session response: %v", err)
	}
	if nextPayload.Session.CurrentIndex != 1 || nextPayload.Session.CurrentItem == nil || nextPayload.Session.CurrentItem.Text != "orange" {
		t.Fatalf("unexpected advanced session: %+v", nextPayload.Session)
	}

	dailyStatsRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/daily?family_id=306&user_id=1&date=2026-03-15", nil)
	if dailyStatsRecorder.Code != http.StatusOK {
		t.Fatalf("expected daily stats to return 200, got %d: %s", dailyStatsRecorder.Code, dailyStatsRecorder.Body.String())
	}

	var dailyStatsPayload statsResponse
	if err := json.Unmarshal(dailyStatsRecorder.Body.Bytes(), &dailyStatsPayload); err != nil {
		t.Fatalf("unmarshal daily stats response: %v", err)
	}
	if dailyStatsPayload.Period != "daily" || dailyStatsPayload.Totals.TotalTasks == 0 || dailyStatsPayload.Totals.WordItems != 3 {
		t.Fatalf("unexpected daily stats payload: %+v", dailyStatsPayload)
	}

	monthlyStatsRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/monthly?family_id=306&user_id=1&month=2026-03", nil)
	if monthlyStatsRecorder.Code != http.StatusOK {
		t.Fatalf("expected monthly stats to return 200, got %d: %s", monthlyStatsRecorder.Code, monthlyStatsRecorder.Body.String())
	}

	var monthlyStatsPayload statsResponse
	if err := json.Unmarshal(monthlyStatsRecorder.Body.Bytes(), &monthlyStatsPayload); err != nil {
		t.Fatalf("unmarshal monthly stats response: %v", err)
	}
	if monthlyStatsPayload.Period != "monthly" || len(monthlyStatsPayload.CompletionSeries) == 0 || len(monthlyStatsPayload.WordSeries) == 0 {
		t.Fatalf("unexpected monthly stats payload: %+v", monthlyStatsPayload)
	}
}

func TestPhaseOneDictationGradeAcceptedAndFailsWithoutLLM(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_GRADER_MODEL_NAME", "")

	router := SetupRouter()

	performJSONRequest(t, router, http.MethodPost, "/api/v1/word-lists", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
		"title":         "英语默写 Day 2",
		"language":      "en",
		"items": []map[string]any{
			{"text": "touch", "meaning": "触碰"},
			{"text": "feel", "meaning": "摸起来"},
		},
	})

	startRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-16",
	})
	if startRecorder.Code != http.StatusCreated {
		t.Fatalf("expected session start to return 201, got %d: %s", startRecorder.Code, startRecorder.Body.String())
	}

	var startPayload dictationSessionResponse
	if err := json.Unmarshal(startRecorder.Body.Bytes(), &startPayload); err != nil {
		t.Fatalf("unmarshal start session response: %v", err)
	}

	gradeRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+startPayload.Session.SessionID+"/grade", map[string]any{
		"photo":    "ZmFrZS1pbWFnZS1ieXRlcw==",
		"language": "english",
		"mode":     "word",
	})
	if gradeRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected grade request to return 202, got %d: %s", gradeRecorder.Code, gradeRecorder.Body.String())
	}

	var acceptedPayload dictationSessionResponse
	if err := json.Unmarshal(gradeRecorder.Body.Bytes(), &acceptedPayload); err != nil {
		t.Fatalf("unmarshal accepted grading response: %v", err)
	}
	if acceptedPayload.Session.GradingStatus != "pending" {
		t.Fatalf("expected accepted grading status to be pending, got %+v", acceptedPayload.Session)
	}
	if acceptedPayload.Session.DebugContext == nil {
		t.Fatalf("expected accepted grading response to include debug context, got %+v", acceptedPayload.Session)
	}
	if acceptedPayload.Session.DebugContext.WorkerStage != "queued" {
		t.Fatalf("expected accepted grading response to record queued stage, got %+v", acceptedPayload.Session.DebugContext)
	}
	if acceptedPayload.Session.DebugContext.PhotoSHA1 == "" || acceptedPayload.Session.DebugContext.LogFile == "" {
		t.Fatalf("expected accepted grading response to include traceable debug metadata, got %+v", acceptedPayload.Session.DebugContext)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		sessionRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/dictation-sessions/"+startPayload.Session.SessionID, nil)
		if sessionRecorder.Code != http.StatusOK {
			t.Fatalf("expected session get to return 200, got %d: %s", sessionRecorder.Code, sessionRecorder.Body.String())
		}

		var sessionPayload dictationSessionResponse
		if err := json.Unmarshal(sessionRecorder.Body.Bytes(), &sessionPayload); err != nil {
			t.Fatalf("unmarshal session payload: %v", err)
		}

		if sessionPayload.Session.GradingStatus == "failed" {
			if sessionPayload.Session.GradingError == "" {
				t.Fatalf("expected grading failure to include an error message: %+v", sessionPayload.Session)
			}
			if sessionPayload.Session.DebugContext == nil {
				t.Fatalf("expected failed grading session to include debug context: %+v", sessionPayload.Session)
			}
			if sessionPayload.Session.DebugContext.PhotoSHA1 == "" || len(sessionPayload.Session.DebugContext.LogKeywords) == 0 {
				t.Fatalf("expected failed grading session to include log keywords and photo hash: %+v", sessionPayload.Session.DebugContext)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected async grading to fail within timeout, got %+v", sessionPayload.Session)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestPhaseOneListDictationSessionsByDate(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_GRADER_MODEL_NAME", "")

	router := SetupRouter()

	performJSONRequest(t, router, http.MethodPost, "/api/v1/word-lists", map[string]any{
		"family_id":     406,
		"child_id":      506,
		"assigned_date": "2026-03-17",
		"title":         "英语默写 Day 3",
		"language":      "en",
		"items": []map[string]any{
			{"text": "noise", "meaning": "响声"},
		},
	})

	startRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     406,
		"child_id":      506,
		"assigned_date": "2026-03-17",
	})
	if startRecorder.Code != http.StatusCreated {
		t.Fatalf("expected session start to return 201, got %d: %s", startRecorder.Code, startRecorder.Body.String())
	}

	var startPayload dictationSessionResponse
	if err := json.Unmarshal(startRecorder.Body.Bytes(), &startPayload); err != nil {
		t.Fatalf("unmarshal start session response: %v", err)
	}

	gradeRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/"+startPayload.Session.SessionID+"/grade", map[string]any{
		"photo":    "ZmFrZS1pbWFnZS1ieXRlcw==",
		"language": "english",
		"mode":     "word",
	})
	if gradeRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected grade request to return 202, got %d: %s", gradeRecorder.Code, gradeRecorder.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/dictation-sessions?family_id=406&child_id=506&date=2026-03-17", nil)
		if listRecorder.Code != http.StatusOK {
			t.Fatalf("expected list dictation sessions to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
		}

		var payload dictationSessionListResponse
		if err := json.Unmarshal(listRecorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal dictation session list response: %v", err)
		}
		if len(payload.Sessions) != 1 {
			t.Fatalf("expected exactly one dictation session, got %+v", payload.Sessions)
		}

		session := payload.Sessions[0]
		if session.SessionID != startPayload.Session.SessionID {
			t.Fatalf("expected listed session id %s, got %+v", startPayload.Session.SessionID, session)
		}
		if session.GradingStatus == "failed" {
			if session.GradingError == "" {
				t.Fatalf("expected listed failed session to include grading error, got %+v", session)
			}
			if session.DebugContext == nil || session.DebugContext.PhotoSHA1 == "" || session.DebugContext.LogFile == "" {
				t.Fatalf("expected listed failed session to include debug context, got %+v", session)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected listed dictation session to reach terminal state, got %+v", session)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestPhaseOneEndpointErrors(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()

	publishRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/daily-assignments/publish", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-20",
		"draft_id":      "draft_missing",
	})
	if publishRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing draft publish to return 404, got %d: %s", publishRecorder.Code, publishRecorder.Body.String())
	}

	publishError := decodeRouteErrorResponse(t, publishRecorder)
	if publishError.ErrorCode != "daily_assignment_draft_not_found" {
		t.Fatalf("expected daily_assignment_draft_not_found, got %s", publishError.ErrorCode)
	}

	pointsRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/points/ledger", map[string]any{
		"family_id":   306,
		"user_id":     1,
		"delta":       2,
		"source_type": "task_completion",
		"occurred_on": "2026-03-20",
	})
	if pointsRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid points source to return 400, got %d: %s", pointsRecorder.Code, pointsRecorder.Body.String())
	}

	pointsError := decodeRouteErrorResponse(t, pointsRecorder)
	if pointsError.ErrorCode != "invalid_points_source" {
		t.Fatalf("expected invalid_points_source, got %s", pointsError.ErrorCode)
	}

	startRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/dictation-sessions/start", map[string]any{
		"family_id":     306,
		"child_id":      1,
		"assigned_date": "2026-03-20",
	})
	if startRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing word list start to return 404, got %d: %s", startRecorder.Code, startRecorder.Body.String())
	}

	startError := decodeRouteErrorResponse(t, startRecorder)
	if startError.ErrorCode != "word_list_not_found" {
		t.Fatalf("expected word_list_not_found, got %s", startError.ErrorCode)
	}

	monthlyRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/stats/monthly?family_id=306&user_id=1&month=2026-13", nil)
	if monthlyRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid month to return 400, got %d: %s", monthlyRecorder.Code, monthlyRecorder.Body.String())
	}

	monthlyError := decodeRouteErrorResponse(t, monthlyRecorder)
	if monthlyError.ErrorCode != "invalid_month" {
		t.Fatalf("expected invalid_month, got %s", monthlyError.ErrorCode)
	}
}
