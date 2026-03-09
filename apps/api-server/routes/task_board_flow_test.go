package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikkofu/studyclaw/api-server/services"
)

type taskBoardSummary struct {
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Pending   int    `json:"pending"`
	Status    string `json:"status"`
}

type taskBoardGroup struct {
	Subject   string `json:"subject"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Pending   int    `json:"pending"`
	Status    string `json:"status"`
}

type taskBoardResponse struct {
	Message        string                   `json:"message"`
	UpdatedCount   int                      `json:"updated_count"`
	Date           string                   `json:"date"`
	Tasks          []services.MarkdownTask  `json:"tasks"`
	Groups         []taskBoardGroup         `json:"groups"`
	HomeworkGroups []taskBoardHomeworkGroup `json:"homework_groups"`
	Summary        taskBoardSummary         `json:"summary"`
}

type taskBoardHomeworkGroup struct {
	Subject    string `json:"subject"`
	GroupTitle string `json:"group_title"`
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Pending    int    `json:"pending"`
	Status     string `json:"status"`
}

var march6DemoTasks = []struct {
	subject    string
	groupTitle string
	content    string
}{
	{subject: "数学", groupTitle: "校本P14～15", content: "校本P14～15"},
	{subject: "数学", groupTitle: "练习册P12～13", content: "练习册P12～13"},
	{subject: "英语", groupTitle: "背默M1U1知识梳理单小作文", content: "背默M1U1知识梳理单小作文"},
	{subject: "英语", groupTitle: "部分学生继续订正1号本", content: "部分学生继续订正1号本"},
	{subject: "英语", groupTitle: "预习M1U2", content: "书本上标注好“黄页”出现单词的音标"},
	{subject: "英语", groupTitle: "预习M1U2", content: "抄写单词（今天默写全对，可免抄）"},
	{subject: "英语", groupTitle: "预习M1U2", content: "沪学习听录音跟读"},
	{subject: "语文", groupTitle: "背作文", content: "背作文"},
	{subject: "语文", groupTitle: "练习卷", content: "练习卷"},
}

func seedMarch6DemoTasks(t *testing.T, familyID, userID uint, date time.Time) {
	t.Helper()

	for _, task := range march6DemoTasks {
		if err := services.SaveTaskWithGroupToMDAtDate(familyID, userID, task.subject, task.groupTitle, task.content, date); err != nil {
			t.Fatalf("SaveTaskWithGroupToMDAtDate returned error: %v", err)
		}
	}
}

func performJSONRequest(t *testing.T, router http.Handler, method, target string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeTaskBoardResponse(t *testing.T, recorder *httptest.ResponseRecorder) taskBoardResponse {
	t.Helper()

	var payload taskBoardResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	return payload
}

func findGroup(t *testing.T, groups []taskBoardGroup, subject string) taskBoardGroup {
	t.Helper()

	for _, group := range groups {
		if group.Subject == subject {
			return group
		}
	}

	t.Fatalf("group %s not found", subject)
	return taskBoardGroup{}
}

func findHomeworkGroup(t *testing.T, groups []taskBoardHomeworkGroup, subject, groupTitle string) taskBoardHomeworkGroup {
	t.Helper()

	for _, group := range groups {
		if group.Subject == subject && group.GroupTitle == groupTitle {
			return group
		}
	}

	t.Fatalf("homework group %s/%s not found", subject, groupTitle)
	return taskBoardHomeworkGroup{}
}

func TestTaskBoardStatusFlowForMarch6Demo(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("STUDYCLAW_DATA_DIR", dataDir)

	familyID := uint(306)
	userID := uint(1)
	date := time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local)

	seedMarch6DemoTasks(t, familyID, userID, date)
	router := SetupRouter()

	listRecorder := performJSONRequest(t, router, http.MethodGet, "/api/v1/tasks?family_id=306&user_id=1&date=2026-03-06", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected GET /tasks to return 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	initialBoard := decodeTaskBoardResponse(t, listRecorder)
	if initialBoard.Date != "2026-03-06" {
		t.Fatalf("expected date 2026-03-06, got %s", initialBoard.Date)
	}
	if len(initialBoard.Tasks) != 9 {
		t.Fatalf("expected 9 tasks, got %d", len(initialBoard.Tasks))
	}
	if initialBoard.Summary.Status != "pending" || initialBoard.Summary.Completed != 0 || initialBoard.Summary.Pending != 9 {
		t.Fatalf("unexpected initial summary: %+v", initialBoard.Summary)
	}
	if len(initialBoard.Groups) != 3 {
		t.Fatalf("expected 3 subject groups, got %d", len(initialBoard.Groups))
	}
	if len(initialBoard.HomeworkGroups) != 7 {
		t.Fatalf("expected 7 homework groups, got %d", len(initialBoard.HomeworkGroups))
	}
	if initialBoard.Groups[0].Subject != "数学" || initialBoard.Groups[1].Subject != "英语" || initialBoard.Groups[2].Subject != "语文" {
		t.Fatalf("unexpected group order: %+v", initialBoard.Groups)
	}

	itemRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/item", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"task_id":       1,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if itemRecorder.Code != http.StatusOK {
		t.Fatalf("expected PATCH /tasks/status/item to return 200, got %d: %s", itemRecorder.Code, itemRecorder.Body.String())
	}

	itemBoard := decodeTaskBoardResponse(t, itemRecorder)
	if !itemBoard.Tasks[0].Completed || itemBoard.Tasks[0].Status != "completed" {
		t.Fatalf("expected first task completed, got %+v", itemBoard.Tasks[0])
	}
	if itemBoard.Summary.Completed != 1 || itemBoard.Summary.Pending != 8 || itemBoard.Summary.Status != "partial" {
		t.Fatalf("unexpected item-update summary: %+v", itemBoard.Summary)
	}
	mathGroup := findGroup(t, itemBoard.Groups, "数学")
	if mathGroup.Completed != 1 || mathGroup.Pending != 1 || mathGroup.Status != "partial" {
		t.Fatalf("unexpected math group after single update: %+v", mathGroup)
	}

	groupRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"subject":       "英语",
		"group_title":   "预习M1U2",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if groupRecorder.Code != http.StatusOK {
		t.Fatalf("expected PATCH /tasks/status/group to return 200, got %d: %s", groupRecorder.Code, groupRecorder.Body.String())
	}

	groupBoard := decodeTaskBoardResponse(t, groupRecorder)
	previewGroup := findHomeworkGroup(t, groupBoard.HomeworkGroups, "英语", "预习M1U2")
	if previewGroup.Completed != 3 || previewGroup.Pending != 0 || previewGroup.Status != "completed" {
		t.Fatalf("unexpected homework group after group update: %+v", previewGroup)
	}
	englishGroup := findGroup(t, groupBoard.Groups, "英语")
	if englishGroup.Completed != 3 || englishGroup.Pending != 2 || englishGroup.Status != "partial" {
		t.Fatalf("unexpected english subject group after group update: %+v", englishGroup)
	}
	if groupBoard.Summary.Completed != 4 || groupBoard.Summary.Pending != 5 || groupBoard.Summary.Status != "partial" {
		t.Fatalf("unexpected group-update summary: %+v", groupBoard.Summary)
	}

	subjectRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/group", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"subject":       "语文",
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if subjectRecorder.Code != http.StatusOK {
		t.Fatalf("expected subject-level PATCH /tasks/status/group to return 200, got %d: %s", subjectRecorder.Code, subjectRecorder.Body.String())
	}

	subjectBoard := decodeTaskBoardResponse(t, subjectRecorder)
	chineseGroup := findGroup(t, subjectBoard.Groups, "语文")
	if chineseGroup.Completed != 2 || chineseGroup.Pending != 0 || chineseGroup.Status != "completed" {
		t.Fatalf("unexpected chinese group after subject update: %+v", chineseGroup)
	}
	if subjectBoard.Summary.Completed != 6 || subjectBoard.Summary.Pending != 3 || subjectBoard.Summary.Status != "partial" {
		t.Fatalf("unexpected subject-update summary: %+v", subjectBoard.Summary)
	}

	allRecorder := performJSONRequest(t, router, http.MethodPatch, "/api/v1/tasks/status/all", map[string]interface{}{
		"family_id":     familyID,
		"assignee_id":   userID,
		"completed":     true,
		"assigned_date": "2026-03-06",
	})
	if allRecorder.Code != http.StatusOK {
		t.Fatalf("expected PATCH /tasks/status/all to return 200, got %d: %s", allRecorder.Code, allRecorder.Body.String())
	}

	finalBoard := decodeTaskBoardResponse(t, allRecorder)
	if finalBoard.Summary.Completed != 9 || finalBoard.Summary.Pending != 0 || finalBoard.Summary.Status != "completed" {
		t.Fatalf("unexpected final summary: %+v", finalBoard.Summary)
	}
	for _, task := range finalBoard.Tasks {
		if !task.Completed || task.Status != "completed" {
			t.Fatalf("expected all tasks completed, got %+v", task)
		}
	}

	markdownPath := filepath.Join(dataDir, "workspaces", "family_306", "user_1", "2026-03-06.md")
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	expectedMarkdown := "# 2026年03月06日 - 今日成长轨迹\n\n## 🎯 任务清单\n" +
		"\n### 数学\n" +
		"\n#### 校本P14～15\n" +
		"- [x] 校本P14～15\n" +
		"\n#### 练习册P12～13\n" +
		"- [x] 练习册P12～13\n" +
		"\n### 英语\n" +
		"\n#### 背默M1U1知识梳理单小作文\n" +
		"- [x] 背默M1U1知识梳理单小作文\n" +
		"\n#### 部分学生继续订正1号本\n" +
		"- [x] 部分学生继续订正1号本\n" +
		"\n#### 预习M1U2\n" +
		"- [x] 书本上标注好“黄页”出现单词的音标\n" +
		"- [x] 抄写单词（今天默写全对，可免抄）\n" +
		"- [x] 沪学习听录音跟读\n" +
		"\n### 语文\n" +
		"\n#### 背作文\n" +
		"- [x] 背作文\n" +
		"\n#### 练习卷\n" +
		"- [x] 练习卷\n"
	if string(content) != expectedMarkdown {
		t.Fatalf("unexpected markdown content:\n%s", string(content))
	}
}
