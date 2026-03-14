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
		Subject                string   `json:"subject"`
		GroupTitle             string   `json:"group_title"`
		Title                  string   `json:"title"`
		Type                   string   `json:"type"`
		Confidence             float64  `json:"confidence"`
		NeedsReview            bool     `json:"needs_review"`
		Notes                  []string `json:"notes"`
		ReferenceTitle         string   `json:"reference_title"`
		ReferenceAuthor        string   `json:"reference_author"`
		ReferenceText          string   `json:"reference_text"`
		ReferenceSource        string   `json:"reference_source"`
		HideReferenceFromChild bool     `json:"hide_reference_from_child"`
		AnalysisMode           string   `json:"analysis_mode"`
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

func TestCreateSingleTaskForSpecificDate(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	router := SetupRouter()

	createRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"subject":       "数学",
		"group_title":   "校本P14-15",
		"content":       "校本P14-15",
		"assigned_date": "2026-03-10",
	})
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create to return 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-10", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	board := decodeTaskBoardResponse(t, listRecorder)
	if board.Date != "2026-03-10" {
		t.Fatalf("expected board date 2026-03-10, got %s", board.Date)
	}
	if len(board.Tasks) != 1 {
		t.Fatalf("expected 1 stored task, got %d", len(board.Tasks))
	}
	if board.Tasks[0].Subject != "数学" || board.Tasks[0].GroupTitle != "校本P14-15" || board.Tasks[0].Content != "校本P14-15" {
		t.Fatalf("unexpected stored task: %+v", board.Tasks[0])
	}
	if board.Summary.Total != 1 || board.Summary.Pending != 1 || board.Summary.Completed != 0 {
		t.Fatalf("unexpected summary after create: %+v", board.Summary)
	}
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

	preConfirmRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-10", nil)
	if preConfirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected pre-confirm list to return 200, got %d: %s", preConfirmRecorder.Code, preConfirmRecorder.Body.String())
	}

	preConfirmBoard := decodeTaskBoardResponse(t, preConfirmRecorder)
	if len(preConfirmBoard.Tasks) != 0 {
		t.Fatalf("expected parse without auto_create to keep board empty, got %d tasks", len(preConfirmBoard.Tasks))
	}
	if preConfirmBoard.Summary.Total != 0 || preConfirmBoard.Summary.Status != "empty" {
		t.Fatalf("unexpected pre-confirm summary: %+v", preConfirmBoard.Summary)
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

func TestParseAndConfirmCarriesLearningReferenceMetadata(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL_NAME", "")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	router := SetupRouter()

	rawText := "语文：\n1. 背诵《江畔独步寻花》\n\n江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？"
	parseRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/parse", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-11",
		"auto_create":   false,
		"raw_text":      rawText,
	})
	if parseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected parse to return 201, got %d: %s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parsePayload := decodeParseTaskResponse(t, parseRecorder.Body.Bytes())
	if parsePayload.ParsedCount != 1 || len(parsePayload.Tasks) != 1 {
		t.Fatalf("expected 1 parsed task, got %+v", parsePayload)
	}

	task := parsePayload.Tasks[0]
	if task.Type != "recitation" {
		t.Fatalf("expected recitation type, got %+v", task)
	}
	if task.ReferenceTitle != "江畔独步寻花" || task.ReferenceAuthor != "杜甫" {
		t.Fatalf("expected parse response to include reference identity, got %+v", task)
	}
	if task.ReferenceText == "" || task.AnalysisMode != "classical_poem" {
		t.Fatalf("expected parse response to include reference metadata, got %+v", task)
	}
	if task.ReferenceSource != "extracted" {
		t.Fatalf("expected parse response to include extracted reference source, got %+v", task)
	}
	if !task.HideReferenceFromChild {
		t.Fatalf("expected parse response to hide recitation text from child, got %+v", task)
	}

	confirmRecorder := performJSONRequest(t, router, http.MethodPost, "/api/v1/tasks/confirm", map[string]interface{}{
		"family_id":     306,
		"assignee_id":   1,
		"assigned_date": "2026-03-11",
		"tasks":         parsePayload.Tasks,
	})
	if confirmRecorder.Code != http.StatusCreated {
		t.Fatalf("expected confirm to return 201, got %d: %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}

	listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-11", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	board := decodeTaskBoardResponse(t, listRecorder)
	if len(board.Tasks) != 1 {
		t.Fatalf("expected 1 stored task, got %d", len(board.Tasks))
	}
	if board.Tasks[0].ReferenceTitle != "江畔独步寻花" || board.Tasks[0].ReferenceAuthor != "杜甫" {
		t.Fatalf("expected stored task to preserve reference identity, got %+v", board.Tasks[0])
	}
	if board.Tasks[0].ReferenceText == "" || board.Tasks[0].AnalysisMode != "classical_poem" {
		t.Fatalf("expected stored task to preserve reference metadata, got %+v", board.Tasks[0])
	}
	if board.Tasks[0].ReferenceSource != "extracted" {
		t.Fatalf("expected stored task to preserve extracted reference source, got %+v", board.Tasks[0])
	}
	if !board.Tasks[0].HideReferenceFromChild {
		t.Fatalf("expected stored task to keep hide_reference_from_child, got %+v", board.Tasks[0])
	}
}
