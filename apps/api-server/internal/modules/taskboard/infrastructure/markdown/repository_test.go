package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

func TestAddTaskAndGetTasks(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	repository := NewRepository()
	familyID := uint(101)
	userID := uint(202)
	date := time.Now()

	if err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:   familyID,
		AssigneeID: userID,
		Subject:    "数学",
		Content:    "口算 30 题",
	}, date); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}

	tasks, err := repository.GetTasks(familyID, userID, date)
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Subject != "数学" {
		t.Fatalf("expected subject 数学, got %s", tasks[0].Subject)
	}
	if tasks[0].Content != "口算 30 题" {
		t.Fatalf("expected content 口算 30 题, got %s", tasks[0].Content)
	}
	if tasks[0].Completed {
		t.Fatal("expected task to be pending")
	}
	if tasks[0].TaskID != 1 {
		t.Fatalf("expected task id 1, got %d", tasks[0].TaskID)
	}
	if tasks[0].Status != "pending" {
		t.Fatalf("expected status pending, got %s", tasks[0].Status)
	}
	if tasks[0].GroupTitle != "口算 30 题" {
		t.Fatalf("expected group title 口算 30 题, got %s", tasks[0].GroupTitle)
	}
}

func TestGetTasksParsesCompletedCheckboxes(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("STUDYCLAW_DATA_DIR", tempDir)

	repository := NewRepository()
	familyID := uint(1)
	userID := uint(2)
	date := time.Date(2026, 3, 9, 8, 0, 0, 0, time.Local)

	path, err := repository.EnsureDailyFile(familyID, userID, date)
	if err != nil {
		t.Fatalf("EnsureDailyFile returned error: %v", err)
	}

	content := "# 2026年03月09日 - 今日成长轨迹\n\n## 🎯 任务清单\n\n### 语文\n\n#### 背诵课文\n- [ ] 背诵课文\n\n### 英语\n\n#### 听写单词\n- [x] 听写单词\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	tasks, err := repository.GetTasks(familyID, userID, date)
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Completed {
		t.Fatal("expected first task to be pending")
	}
	if !tasks[1].Completed {
		t.Fatal("expected second task to be completed")
	}
	if tasks[0].GroupTitle != "背诵课文" || tasks[1].GroupTitle != "听写单词" {
		t.Fatalf("unexpected parsed group titles: %+v", tasks)
	}

	expectedPath := filepath.Join(tempDir, "workspaces", "family_1", "user_2", "2026-03-09.md")
	if path != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, path)
	}
}

func TestUpdateTaskCompletionFlows(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("STUDYCLAW_DATA_DIR", tempDir)

	repository := NewRepository()
	familyID := uint(3)
	userID := uint(6)
	date := time.Date(2026, 3, 6, 8, 0, 0, 0, time.Local)

	if err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:   familyID,
		AssigneeID: userID,
		Subject:    "数学",
		GroupTitle: "校本作业",
		Content:    "校本P14～15",
	}, date); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:   familyID,
		AssigneeID: userID,
		Subject:    "数学",
		GroupTitle: "练习册",
		Content:    "练习册P12～13",
	}, date); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:   familyID,
		AssigneeID: userID,
		Subject:    "英语",
		GroupTitle: "预习M1U2",
		Content:    "书本上标注好“黄页”出现单词的音标",
	}, date); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:   familyID,
		AssigneeID: userID,
		Subject:    "英语",
		GroupTitle: "预习M1U2",
		Content:    "沪学习听录音跟读",
	}, date); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}

	tasks, matchedCount, updatedCount, err := repository.UpdateTaskCompletionByID(familyID, userID, date, 1, true)
	if err != nil {
		t.Fatalf("UpdateTaskCompletionByID returned error: %v", err)
	}
	if matchedCount != 1 {
		t.Fatalf("expected single-task matched count 1, got %d", matchedCount)
	}
	if updatedCount != 1 {
		t.Fatalf("expected single-task update count 1, got %d", updatedCount)
	}
	if !tasks[0].Completed || tasks[0].Status != "completed" {
		t.Fatalf("expected first task completed, got %+v", tasks[0])
	}

	tasks, matchedCount, updatedCount, err = repository.UpdateTaskCompletionBySubject(familyID, userID, date, "数学", true)
	if err != nil {
		t.Fatalf("UpdateTaskCompletionBySubject returned error: %v", err)
	}
	if matchedCount != 2 {
		t.Fatalf("expected subject matched count 2, got %d", matchedCount)
	}
	if updatedCount != 1 {
		t.Fatalf("expected subject update count 1, got %d", updatedCount)
	}
	if !tasks[1].Completed {
		t.Fatalf("expected second math task completed, got %+v", tasks[1])
	}

	tasks, matchedCount, updatedCount, err = repository.UpdateTaskCompletionByHomeworkGroup(familyID, userID, date, "英语", "预习M1U2", true)
	if err != nil {
		t.Fatalf("UpdateTaskCompletionByHomeworkGroup returned error: %v", err)
	}
	if matchedCount != 2 {
		t.Fatalf("expected homework-group matched count 2, got %d", matchedCount)
	}
	if updatedCount != 2 {
		t.Fatalf("expected homework-group update count 2, got %d", updatedCount)
	}
	if !tasks[2].Completed || !tasks[3].Completed {
		t.Fatalf("expected english group tasks completed, got %+v", tasks)
	}

	tasks, matchedCount, updatedCount, err = repository.UpdateAllTasksCompletion(familyID, userID, date, true)
	if err != nil {
		t.Fatalf("UpdateAllTasksCompletion returned error: %v", err)
	}
	if matchedCount != 4 {
		t.Fatalf("expected all-task matched count 4, got %d", matchedCount)
	}
	if updatedCount != 0 {
		t.Fatalf("expected duplicate all-task update count 0, got %d", updatedCount)
	}

	tasks, matchedCount, updatedCount, err = repository.UpdateAllTasksCompletion(familyID, userID, date, false)
	if err != nil {
		t.Fatalf("UpdateAllTasksCompletion returned error: %v", err)
	}
	if matchedCount != 4 {
		t.Fatalf("expected all-task matched count 4, got %d", matchedCount)
	}
	if updatedCount != 4 {
		t.Fatalf("expected all-task update count 4, got %d", updatedCount)
	}
	for _, task := range tasks {
		if task.Completed {
			t.Fatalf("expected all tasks pending, got %+v", task)
		}
	}

	path, err := repository.EnsureDailyFile(familyID, userID, date)
	if err != nil {
		t.Fatalf("EnsureDailyFile returned error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	expectedMarkdown := "# 2026年03月06日 - 今日成长轨迹\n\n## 🎯 任务清单\n\n### 数学\n\n#### 校本作业\n- [ ] 校本P14～15\n\n#### 练习册\n- [ ] 练习册P12～13\n\n### 英语\n\n#### 预习M1U2\n- [ ] 书本上标注好“黄页”出现单词的音标\n- [ ] 沪学习听录音跟读\n"
	if string(content) != expectedMarkdown {
		t.Fatalf("unexpected markdown content:\n%s", string(content))
	}
}

func TestTaskMetadataRoundTrip(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())

	repository := NewRepository()
	familyID := uint(12)
	userID := uint(34)
	date := time.Date(2026, 3, 13, 8, 0, 0, 0, time.Local)

	err := repository.AddTask(domain.CreateTaskInput{
		FamilyID:               familyID,
		AssigneeID:             userID,
		Subject:                "语文",
		GroupTitle:             "古诗背诵",
		Content:                "背诵《江畔独步寻花》",
		TaskType:               "recitation",
		ReferenceTitle:         "江畔独步寻花",
		ReferenceAuthor:        "杜甫",
		ReferenceText:          "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？",
		ReferenceSource:        "manual",
		HideReferenceFromChild: true,
		AnalysisMode:           "classical_poem",
	}, date)
	if err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}

	tasks, err := repository.GetTasks(familyID, userID, date)
	if err != nil {
		t.Fatalf("GetTasks returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.TaskType != "recitation" {
		t.Fatalf("expected task_type recitation, got %q", task.TaskType)
	}
	if task.ReferenceTitle != "江畔独步寻花" || task.ReferenceAuthor != "杜甫" {
		t.Fatalf("unexpected reference identity: %+v", task)
	}
	if task.ReferenceText == "" {
		t.Fatalf("expected reference text to round-trip, got %+v", task)
	}
	if task.ReferenceSource != "manual" {
		t.Fatalf("expected reference_source manual, got %+v", task)
	}
	if !task.HideReferenceFromChild {
		t.Fatalf("expected hide_reference_from_child true, got %+v", task)
	}
	if task.AnalysisMode != "classical_poem" {
		t.Fatalf("expected analysis_mode classical_poem, got %q", task.AnalysisMode)
	}

	path, err := repository.EnsureDailyFile(familyID, userID, date)
	if err != nil {
		t.Fatalf("EnsureDailyFile returned error: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !contains(string(content), "studyclaw:task:") {
		t.Fatalf("expected markdown to contain metadata comment, got:\n%s", string(content))
	}
}

func contains(text, fragment string) bool {
	return strings.Contains(text, fragment)
}
