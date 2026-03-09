package taskparse

import "testing"

const sampleGroupMessage = `数学3.6：
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

func TestExtractStructureOutlineDetectsSectionsAndSignals(t *testing.T) {
	outline := extractStructureOutline(sampleGroupMessage)

	expectedSubjects := []string{"数学", "英语", "语文"}
	if len(outline.DetectedSubjects) != len(expectedSubjects) {
		t.Fatalf("expected %d subjects, got %d", len(expectedSubjects), len(outline.DetectedSubjects))
	}
	for index, expected := range expectedSubjects {
		if outline.DetectedSubjects[index] != expected {
			t.Fatalf("expected subject %d to be %s, got %s", index, expected, outline.DetectedSubjects[index])
		}
	}

	if !containsString(outline.FormatSignals, "subject_headings") {
		t.Fatal("expected subject_headings format signal")
	}
	if !containsString(outline.FormatSignals, "numbered_tasks") {
		t.Fatal("expected numbered_tasks format signal")
	}
	if !containsString(outline.FormatSignals, "nested_subtasks") {
		t.Fatal("expected nested_subtasks format signal")
	}
	if len(outline.Tasks) != 9 {
		t.Fatalf("expected 9 preview tasks, got %d", len(outline.Tasks))
	}
}

func TestParseFallbackMergesNestedSubtasks(t *testing.T) {
	result := parseFallback(sampleGroupMessage)

	expectedTasks := []struct {
		subject    string
		groupTitle string
		title      string
	}{
		{subject: "数学", groupTitle: "校本P14～15", title: "校本P14～15"},
		{subject: "数学", groupTitle: "练习册P12～13", title: "练习册P12～13"},
		{subject: "英语", groupTitle: "背默M1U1知识梳理单小作文", title: "背默M1U1知识梳理单小作文"},
		{subject: "英语", groupTitle: "部分学生继续订正1号本", title: "部分学生继续订正1号本"},
		{subject: "英语", groupTitle: "预习M1U2", title: "书本上标注好“黄页”出现单词的音标"},
		{subject: "英语", groupTitle: "预习M1U2", title: "抄写单词（今天默写全对，可免抄）"},
		{subject: "英语", groupTitle: "预习M1U2", title: "沪学习听录音跟读"},
		{subject: "语文", groupTitle: "背作文", title: "背作文"},
		{subject: "语文", groupTitle: "练习卷", title: "练习卷"},
	}

	if result.Status != "success" {
		t.Fatalf("expected status success, got %s", result.Status)
	}
	if result.ParserMode != "rule_fallback" {
		t.Fatalf("expected parser_mode rule_fallback, got %s", result.ParserMode)
	}
	if len(result.Data) != len(expectedTasks) {
		t.Fatalf("expected %d tasks, got %d", len(expectedTasks), len(result.Data))
	}
	if result.Analysis.NeedsReviewCount == 0 {
		t.Fatal("expected needs_review_count to be greater than 0")
	}

	for index, expected := range expectedTasks {
		actual := result.Data[index]
		if actual.Subject != expected.subject || actual.GroupTitle != expected.groupTitle || actual.Title != expected.title {
			t.Fatalf("unexpected task at index %d: %+v", index, actual)
		}
	}

	if !result.Data[3].NeedsReview {
		t.Fatal("expected conditional student-specific task to need review")
	}
	if !result.Data[5].NeedsReview {
		t.Fatal("expected conditional optional-copy task to need review")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
