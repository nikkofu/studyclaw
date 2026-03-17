package taskparse

import (
	"context"
	"strings"
	"testing"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

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

const sampleCorrectionMessage = `英语：
1. 订正今日默写错词并家长签字
2. 部分学生续做1号卷剩余题
3. 个别同学继续订正
`

const sampleNestedVariantMessage = `数学：
1. 校本P20
2. 口算练习
1）完成第3页
2）订正错题
3）全对可免做第4页

语文：
1. 预习第7课
① 圈画生字
② 朗读课文三遍
`

const sampleNormalHomeworkMessage = `数学：
1. 完成口算本第5页
2. 继续完成校本P19
3. 订正第2页错题

语文：
1. 阅读《假若给我三天光明》并摘抄好词
2. 预习第6课
1）圈画生字
2）朗读课文三遍
`

const sampleRecitationWithReferenceMessage = `语文：
1. 背诵《江畔独步寻花》

江畔独步寻花【唐】杜甫
黄师塔前江水东，春光懒困倚微风。
桃花一簇开无主，可爱深红爱浅红？
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

func TestParseFallbackDetectsCorrectionContinuationAndAmbiguousTargets(t *testing.T) {
	result := parseFallback(sampleCorrectionMessage)

	if result.Status != "success" {
		t.Fatalf("expected status success, got %s", result.Status)
	}
	if len(result.Data) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Data))
	}
	if !containsString(result.Analysis.FormatSignals, "correction_tasks") {
		t.Fatal("expected correction_tasks format signal")
	}
	if !containsString(result.Analysis.FormatSignals, "continuation_tasks") {
		t.Fatal("expected continuation_tasks format signal")
	}
	if !containsString(result.Analysis.FormatSignals, "audience_constraints") {
		t.Fatal("expected audience_constraints format signal")
	}
	if !containsString(result.Analysis.FormatSignals, "ambiguous_targets") {
		t.Fatal("expected ambiguous_targets format signal")
	}

	explicitCorrection := findTaskByTitle(t, result.Data, "订正今日默写错词并家长签字")
	if explicitCorrection.NeedsReview {
		t.Fatalf("expected explicit correction task to skip review: %+v", explicitCorrection)
	}

	audienceScoped := findTaskByTitle(t, result.Data, "部分学生续做1号卷剩余题")
	if !audienceScoped.NeedsReview {
		t.Fatalf("expected audience-scoped continuation to need review: %+v", audienceScoped)
	}
	if !containsString(audienceScoped.Notes, "作业适用对象不明确，建议家长确认是否针对孩子。") {
		t.Fatalf("expected audience review note, got %+v", audienceScoped.Notes)
	}

	ambiguousCorrection := findTaskByTitle(t, result.Data, "个别同学继续订正")
	if !ambiguousCorrection.NeedsReview {
		t.Fatalf("expected ambiguous correction to need review: %+v", ambiguousCorrection)
	}
	if !containsString(ambiguousCorrection.Notes, "订正/续做任务未写明具体对象，建议家长确认完成内容。") {
		t.Fatalf("expected ambiguous target note, got %+v", ambiguousCorrection.Notes)
	}
	if ambiguousCorrection.Confidence >= 0.7 {
		t.Fatalf("expected lower confidence for ambiguous correction, got %.2f", ambiguousCorrection.Confidence)
	}
}

func TestParseFallbackRecognizesNestedStepVariants(t *testing.T) {
	result := parseFallback(sampleNestedVariantMessage)

	if result.Status != "success" {
		t.Fatalf("expected status success, got %s", result.Status)
	}
	if len(result.Data) != 6 {
		t.Fatalf("expected 6 tasks, got %d", len(result.Data))
	}
	if !containsString(result.Analysis.FormatSignals, "nested_subtasks") {
		t.Fatal("expected nested_subtasks format signal")
	}

	stepTask := findTaskByTitle(t, result.Data, "完成第3页")
	if stepTask.GroupTitle != "口算练习" {
		t.Fatalf("expected numbered substep group title, got %+v", stepTask)
	}

	circledStep := findTaskByTitle(t, result.Data, "圈画生字")
	if circledStep.GroupTitle != "预习第7课" {
		t.Fatalf("expected circled substep group title, got %+v", circledStep)
	}

	conditionalStep := findTaskByTitle(t, result.Data, "全对可免做第4页")
	if !conditionalStep.NeedsReview {
		t.Fatalf("expected conditional substep to need review: %+v", conditionalStep)
	}
	if !containsString(conditionalStep.Notes, "包含条件性说明，建议家长确认触发条件。") {
		t.Fatalf("expected conditional note, got %+v", conditionalStep.Notes)
	}

	if stepTask.NeedsReview {
		t.Fatalf("expected normal numbered substep to avoid review: %+v", stepTask)
	}
	if circledStep.NeedsReview {
		t.Fatalf("expected circled substep to avoid review: %+v", circledStep)
	}
}

func TestParseFallbackRegressionReviewMatrix(t *testing.T) {
	testCases := []struct {
		name         string
		rawText      string
		title        string
		needsReview  bool
		requiredNote string
	}{
		{
			name:        "correction with explicit target stays actionable",
			rawText:     "英语：\n1. 订正默写本P3错词",
			title:       "订正默写本P3错词",
			needsReview: false,
		},
		{
			name:         "correction with ambiguous target needs review",
			rawText:      "英语：\n1. 继续订正",
			title:        "继续订正",
			needsReview:  true,
			requiredNote: "订正/续做任务未写明具体对象，建议家长确认完成内容。",
		},
		{
			name:        "continuation with explicit target stays actionable",
			rawText:     "数学：\n1. 继续完成校本P18",
			title:       "校本P18",
			needsReview: false,
		},
		{
			name:         "continuation with ambiguous target needs review",
			rawText:      "数学：\n1. 继续完成",
			title:        "继续完成",
			needsReview:  true,
			requiredNote: "订正/续做任务未写明具体对象，建议家长确认完成内容。",
		},
		{
			name:         "conditional task needs review",
			rawText:      "英语：\n1. 默写全对可免抄M2单词",
			title:        "默写全对可免抄M2单词",
			needsReview:  true,
			requiredNote: "包含条件性说明，建议家长确认触发条件。",
		},
		{
			name:         "partial students require audience confirmation",
			rawText:      "语文：\n1. 部分学生订正练习卷",
			title:        "部分学生订正练习卷",
			needsReview:  true,
			requiredNote: "作业适用对象不明确，建议家长确认是否针对孩子。",
		},
		{
			name:         "individual students require audience confirmation",
			rawText:      "语文：\n1. 个别同学继续完成口算本P9",
			title:        "个别同学继续完成口算本P9",
			needsReview:  true,
			requiredNote: "作业适用对象不明确，建议家长确认是否针对孩子。",
		},
		{
			name:         "related students require audience confirmation",
			rawText:      "语文：\n1. 相关同学背诵第3课",
			title:        "相关同学背诵第3课",
			needsReview:  true,
			requiredNote: "作业适用对象不明确，建议家长确认是否针对孩子。",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFallback(tc.rawText)
			task := findTaskByTitle(t, result.Data, tc.title)

			if task.NeedsReview != tc.needsReview {
				t.Fatalf("expected needs_review=%v, got %+v", tc.needsReview, task)
			}
			if tc.requiredNote == "" {
				if len(task.Notes) != 0 {
					t.Fatalf("expected no review notes, got %+v", task.Notes)
				}
				return
			}
			if !containsString(task.Notes, tc.requiredNote) {
				t.Fatalf("expected note %q, got %+v", tc.requiredNote, task.Notes)
			}
		})
	}
}

func TestParseFallbackNormalTasksAvoidFalsePositiveNeedsReview(t *testing.T) {
	result := parseFallback(sampleNormalHomeworkMessage)

	if result.Status != "success" {
		t.Fatalf("expected status success, got %s", result.Status)
	}
	if result.Analysis.NeedsReviewCount != 0 {
		t.Fatalf("expected no false positive review flags, got %+v", result.Data)
	}
	if !containsString(result.Analysis.FormatSignals, "nested_subtasks") {
		t.Fatal("expected nested_subtasks signal for normal preview task")
	}

	expectedSafeTitles := []string{
		"口算本第5页",
		"校本P19",
		"订正第2页错题",
		"阅读《假若给我三天光明》并摘抄好词",
		"圈画生字",
		"朗读课文三遍",
	}

	for _, title := range expectedSafeTitles {
		task := findTaskByTitle(t, result.Data, title)
		if task.NeedsReview {
			t.Fatalf("expected normal task %q to avoid review: %+v", title, task)
		}
		if len(task.Notes) != 0 {
			t.Fatalf("expected normal task %q to have no notes: %+v", title, task.Notes)
		}
	}
}

func TestParseFallbackInfersLearningTaskTypes(t *testing.T) {
	result := parseFallback(sampleNormalHomeworkMessage)

	readingTask := findTaskByTitle(t, result.Data, "朗读课文三遍")
	if readingTask.Type != "reading" {
		t.Fatalf("expected reading task type, got %+v", readingTask)
	}

	normalMath := findTaskByTitle(t, result.Data, "口算本第5页")
	if normalMath.Type != "homework" {
		t.Fatalf("expected normal math task to stay homework, got %+v", normalMath)
	}

	recitationResult := parseFallback("语文：\n1. 背诵《江畔独步寻花》\n2. 背作文")
	firstRecitation := findTaskByTitle(t, recitationResult.Data, "背诵《江畔独步寻花》")
	if firstRecitation.Type != "recitation" {
		t.Fatalf("expected poem recitation type, got %+v", firstRecitation)
	}
	secondRecitation := findTaskByTitle(t, recitationResult.Data, "背作文")
	if secondRecitation.Type != "recitation" {
		t.Fatalf("expected memorization task type, got %+v", secondRecitation)
	}
}

func TestParseFallbackExtractsReferenceMaterialFromTeacherRawText(t *testing.T) {
	service := NewService(nil)

	result, err := service.Parse(context.Background(), sampleRecitationWithReferenceMessage)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ParserMode != "rule_fallback" {
		t.Fatalf("expected rule_fallback parser mode, got %s", result.ParserMode)
	}

	task := findTaskByTitle(t, result.Data, "背诵《江畔独步寻花》")
	if task.Type != "recitation" {
		t.Fatalf("expected recitation task type, got %+v", task)
	}
	if task.ReferenceTitle != "江畔独步寻花" {
		t.Fatalf("expected extracted reference title, got %+v", task)
	}
	if task.ReferenceAuthor != "杜甫" {
		t.Fatalf("expected extracted author, got %+v", task)
	}
	if task.ReferenceText == "" || !strings.Contains(task.ReferenceText, "黄师塔前江水东") {
		t.Fatalf("expected extracted reference text, got %+v", task)
	}
	if task.ReferenceSource != "extracted" {
		t.Fatalf("expected extracted reference source, got %+v", task)
	}
	if !task.HideReferenceFromChild {
		t.Fatalf("expected recitation reference to be hidden from child, got %+v", task)
	}
	if task.AnalysisMode != "classical_poem" {
		t.Fatalf("expected classical_poem analysis mode, got %+v", task)
	}
	if !containsString(task.Notes, "已从老师原文自动带出参考内容。") {
		t.Fatalf("expected raw-text enrichment note, got %+v", task.Notes)
	}
}

func TestParseUsesLLMToFillMissingReferenceMaterialWhenTeacherDidNotProvideBody(t *testing.T) {
	service := NewService(&stubLLMClient{
		responses: []string{
			`{"status":"success","data":[{"subject":"语文","group_title":"背诵《江畔独步寻花》","title":"背诵《江畔独步寻花》","type":"recitation","confidence":0.95,"needs_review":false,"notes":[]}]}`,
			`{"status":"success","data":[{"index":0,"reference_title":"江畔独步寻花","reference_author":"杜甫","reference_text":"江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？","hide_reference_from_child":true,"analysis_mode":"classical_poem"}]}`,
		},
	})

	result, err := service.Parse(context.Background(), "语文：\n1. 背诵《江畔独步寻花》")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ParserMode != "llm_hybrid" {
		t.Fatalf("expected llm_hybrid parser mode, got %s", result.ParserMode)
	}

	task := findTaskByTitle(t, result.Data, "背诵《江畔独步寻花》")
	if task.ReferenceTitle != "江畔独步寻花" || task.ReferenceAuthor != "杜甫" {
		t.Fatalf("expected LLM completed identity, got %+v", task)
	}
	if task.ReferenceText == "" || !strings.Contains(task.ReferenceText, "可爱深红爱浅红") {
		t.Fatalf("expected LLM completed reference text, got %+v", task)
	}
	if task.ReferenceSource != "llm" {
		t.Fatalf("expected llm reference source, got %+v", task)
	}
	if !task.HideReferenceFromChild {
		t.Fatalf("expected hide_reference_from_child to remain true, got %+v", task)
	}
	if task.AnalysisMode != "classical_poem" {
		t.Fatalf("expected LLM completed analysis_mode, got %+v", task)
	}
	if !containsString(task.Notes, "参考素材已由 LLM 自动补全。") {
		t.Fatalf("expected LLM enrichment note, got %+v", task.Notes)
	}
	if !containsString(result.Analysis.Notes, "缺失的 1 条朗读/背诵任务已由 LLM 自动补全参考素材。") {
		t.Fatalf("expected analysis to mention LLM enrichment, got %+v", result.Analysis.Notes)
	}
}

func TestParseHybridReappliesDeterministicReviewRules(t *testing.T) {
	service := NewService(&stubLLMClient{
		response: `{"status":"success","data":[{"subject":"英语","group_title":"个别同学继续订正","title":"个别同学继续订正","type":"homework","confidence":0.95,"needs_review":false,"notes":[]}]}`,
	})

	result, err := service.Parse(context.Background(), "英语：1. 个别同学继续订正")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ParserMode != "llm_hybrid" {
		t.Fatalf("expected llm_hybrid parser mode, got %s", result.ParserMode)
	}

	task := findTaskByTitle(t, result.Data, "个别同学继续订正")
	if !task.NeedsReview {
		t.Fatalf("expected deterministic normalization to set needs_review: %+v", task)
	}
	if !containsString(task.Notes, "作业适用对象不明确，建议家长确认是否针对孩子。") {
		t.Fatalf("expected audience review note, got %+v", task.Notes)
	}
	if !containsString(task.Notes, "订正/续做任务未写明具体对象，建议家长确认完成内容。") {
		t.Fatalf("expected ambiguous target note, got %+v", task.Notes)
	}
}

func TestParseHybridCollapsesSimpleEquivalentTaskTitles(t *testing.T) {
	service := NewService(&stubLLMClient{
		response: `{"status":"success","data":[{"subject":"数学","group_title":"完成校本P14～15","title":"完成校本P14～15","type":"homework","confidence":0.95,"needs_review":false,"notes":[]},{"subject":"数学","group_title":"完成练习册P12～13","title":"完成练习册P12～13","type":"homework","confidence":0.95,"needs_review":false,"notes":[]}]}`,
	})

	result, err := service.Parse(context.Background(), "数学：\n1、校本P14～15\n2、练习册P12～13")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ParserMode != "llm_hybrid" {
		t.Fatalf("expected llm_hybrid parser mode, got %s", result.ParserMode)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 merged tasks, got %+v", result.Data)
	}

	first := findTaskByTitle(t, result.Data, "校本P14～15")
	if first.Subject != "数学" || first.GroupTitle != "校本P14～15" {
		t.Fatalf("expected canonical simple task to remain, got %+v", first)
	}

	second := findTaskByTitle(t, result.Data, "练习册P12～13")
	if second.Subject != "数学" || second.GroupTitle != "练习册P12～13" {
		t.Fatalf("expected second canonical simple task to remain, got %+v", second)
	}

	for _, unexpected := range []string{"完成校本P14～15", "完成练习册P12～13"} {
		for _, task := range result.Data {
			if task.Title == unexpected {
				t.Fatalf("expected semantic duplicate %q to be removed: %+v", unexpected, result.Data)
			}
		}
	}
}

func TestParseHybridPreservesRealMultiStepTasks(t *testing.T) {
	service := NewService(&stubLLMClient{
		response: `{"status":"success","data":[{"subject":"英语","group_title":"预习M1U2","title":"书本上标注好“黄页”出现单词的音标","type":"homework","confidence":0.95,"needs_review":false,"notes":[]}]}`,
	})

	result, err := service.Parse(context.Background(), "英语：\n1. 预习M1U2\n（1）书本上标注好“黄页”出现单词的音标\n（2）抄写单词（今天默写全对，可免抄）\n（3）沪学习听录音跟读")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ParserMode != "llm_hybrid" {
		t.Fatalf("expected llm_hybrid parser mode, got %s", result.ParserMode)
	}
	if len(result.Data) != 3 {
		t.Fatalf("expected 3 tasks after fallback fill, got %+v", result.Data)
	}
	if !containsString(result.Analysis.Notes, "LLM 结果缺失的 2 条任务已由结构兜底补全。") {
		t.Fatalf("expected merge note for missing substeps, got %+v", result.Analysis.Notes)
	}

	for _, expected := range []string{"书本上标注好“黄页”出现单词的音标", "抄写单词（今天默写全对，可免抄）", "沪学习听录音跟读"} {
		findTaskByTitle(t, result.Data, expected)
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

func findTaskByTitle(t *testing.T, tasks []ParsedTask, title string) ParsedTask {
	t.Helper()

	for _, task := range tasks {
		if task.Title == title {
			return task
		}
	}

	t.Fatalf("task %q not found in %+v", title, tasks)
	return ParsedTask{}
}

type stubLLMClient struct {
	response  string
	responses []string
	err       error
	callCount int
}

func (s *stubLLMClient) Generate(_ context.Context, _ llm.GenerateRequest) (string, error) {
	if len(s.responses) > 0 {
		index := s.callCount
		if index >= len(s.responses) {
			index = len(s.responses) - 1
		}
		s.callCount++
		return s.responses[index], s.err
	}
	return s.response, s.err
}
