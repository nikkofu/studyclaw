package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const routeSampleGroupMessage = `数学3.6：
1、校本P14～15
2、练习册P12～13

英：
1. 背默M1U1知识梳理单小作文
2. 部分学生继续订正1号本
3. 预习M1U2
（1）书本上标注好“黄页”出现单词的音标
（2）抄写单词（今天默写全对，可免抄）
（3）沪学习听录音跟读

语文：
1. 背作文
2. 练习卷
`

type parseTaskResponse struct {
	Message     string `json:"message"`
	ParsedCount int    `json:"parsed_count"`
	ParserMode  string `json:"parser_mode"`
	AutoCreated bool   `json:"auto_created"`
	Date        string `json:"date"`
	Analysis    struct {
		TaskCount        int `json:"task_count"`
		NeedsReviewCount int `json:"needs_review_count"`
	} `json:"analysis"`
	Tasks []struct {
		Subject     string   `json:"subject"`
		GroupTitle  string   `json:"group_title"`
		Title       string   `json:"title"`
		Confidence  float64  `json:"confidence"`
		NeedsReview bool     `json:"needs_review"`
		Notes       []string `json:"notes"`
	} `json:"tasks"`
}

func decodeParseTaskResponse(t *testing.T, recorderBody []byte) parseTaskResponse {
	t.Helper()

	var payload parseTaskResponse
	if err := json.Unmarshal(recorderBody, &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	return payload
}

func TestParseThenConfirmTasksForSpecificDate(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()

	parseRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/parse", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-10",
		"auto_create":   false,
		"raw_text":      routeSampleGroupMessage,
	})
	if parseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected parse to return 201, got %d: %s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parsePayload := decodeParseTaskResponse(t, parseRecorder.Body.Bytes())
	if parsePayload.ParserMode != "rule_fallback" {
		t.Fatalf("expected rule_fallback parser mode, got %s", parsePayload.ParserMode)
	}
	if parsePayload.Date != "2026-03-10" {
		t.Fatalf("expected assigned date 2026-03-10, got %s", parsePayload.Date)
	}
	if parsePayload.AutoCreated {
		t.Fatal("expected auto_create false response")
	}
	if parsePayload.ParsedCount != 9 || len(parsePayload.Tasks) != 9 {
		t.Fatalf("expected 9 parsed tasks, got parsed_count=%d len=%d", parsePayload.ParsedCount, len(parsePayload.Tasks))
	}
	if parsePayload.Analysis.TaskCount != 9 {
		t.Fatalf("expected analysis task_count 9, got %d", parsePayload.Analysis.TaskCount)
	}
	if parsePayload.Analysis.NeedsReviewCount == 0 {
		t.Fatal("expected at least one needs_review task")
	}
	if !parsePayload.Tasks[3].NeedsReview {
		t.Fatalf("expected conditional task to need review: %+v", parsePayload.Tasks[3])
	}
	if parsePayload.Tasks[4].GroupTitle != "预习M1U2" || parsePayload.Tasks[4].Title != "书本上标注好“黄页”出现单词的音标" {
		t.Fatalf("unexpected nested task mapping: %+v", parsePayload.Tasks[4])
	}

	confirmRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/confirm", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-10",
		"tasks":         parsePayload.Tasks,
	})
	if confirmRecorder.Code != http.StatusCreated {
		t.Fatalf("expected confirm to return 201, got %d: %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}

	listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-10", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	board := decodeTaskBoardResponse(t, listRecorder)
	if board.Date != "2026-03-10" {
		t.Fatalf("expected board date 2026-03-10, got %s", board.Date)
	}
	if len(board.Tasks) != 9 {
		t.Fatalf("expected 9 stored tasks, got %d", len(board.Tasks))
	}
	if board.Summary.Total != 9 || board.Summary.Pending != 9 || board.Summary.Completed != 0 {
		t.Fatalf("unexpected summary after confirm: %+v", board.Summary)
	}
	if len(board.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(board.Groups))
	}
	if len(board.HomeworkGroups) != 7 {
		t.Fatalf("expected 7 homework groups, got %d", len(board.HomeworkGroups))
	}

	markdownPath := filepath.Join(os.Getenv("STUDYCLAW_DATA_DIR"), "workspaces", "family_306", "user_1", "2026-03-10.md")
	if _, err := os.Stat(markdownPath); err != nil {
		t.Fatalf("expected markdown file to exist: %v", err)
	}
}
