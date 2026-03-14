package taskparse

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
	"github.com/nikkofu/studyclaw/api-server/internal/shared/agentic"
)

var (
	subjectAliases = map[string]string{
		"数":  "数学",
		"数学": "数学",
		"英":  "英语",
		"英语": "英语",
		"语":  "语文",
		"语文": "语文",
		"科学": "科学",
		"物理": "物理",
		"化学": "化学",
		"生物": "生物",
		"历史": "历史",
		"地理": "地理",
		"道法": "道法",
		"政治": "道法",
		"美术": "美术",
		"音乐": "音乐",
		"体育": "体育",
		"信息": "信息",
		"劳动": "劳动",
		"阅读": "阅读",
		"班会": "班会",
	}
	whitespacePattern          = regexp.MustCompile(`\s+`)
	headingPattern             = regexp.MustCompile(`^([\p{Han}A-Za-z]+[^：:]*)[：:]\s*(.*)$`)
	headingKeyPattern          = regexp.MustCompile(`^[\p{Han}A-Za-z0-9.\-]+$`)
	mainItemPattern            = regexp.MustCompile(`^\d+\s*[、.．)）]\s*(.+)$`)
	bracketedSubItemPattern    = regexp.MustCompile(`^[（(]\d+[）)]\s*(.+)$`)
	numberedSubItemPattern     = regexp.MustCompile(`^\d+\s*[)）]\s*(.+)$`)
	circledSubItemPattern      = regexp.MustCompile(`^[①②③④⑤⑥⑦⑧⑨⑩]\s*(.+)$`)
	bulletSubItemPattern       = regexp.MustCompile(`^[-•·]\s*(.+)$`)
	taskTargetReferencePattern = regexp.MustCompile(`[A-Za-z]+\d+|\d`)
	jsonFencePattern           = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
	quotedTitlePattern         = regexp.MustCompile(`《([^》]+)》`)
	learningActionTitlePattern = regexp.MustCompile(`(?:背诵|朗读|跟读|诵读|朗诵)\s*[《“"]?([^》”"，。；、\s]{2,30})[》”"]?`)
	referenceHeaderPattern     = regexp.MustCompile(`^(.*?)[【\[\(（〔][^】\]\)）〕]{1,12}[】\]\)）〕]\s*([^\s]{1,16})$`)
	referenceAuthorPattern     = regexp.MustCompile(`^作者[:：]\s*([^\s]{1,16})$`)
	conditionalSignalKeywords  = []string{"可免", "选做", "如需", "如果", "酌情", "全对", "完成后", "做完后"}
	conditionalSignalPatterns  = []*regexp.Regexp{
		regexp.MustCompile(`(^|[，,；;。:：\s])若(?:有|需|未|完成|全对|正确|时间|背会|默写)`),
	}
	audienceSignalKeywords = []string{"部分学生", "个别同学", "相关同学", "有需要的同学", "需要的同学", "未完成的同学", "未交的同学", "没交的同学"}
	correctionKeywords     = []string{"订正", "改错", "更正"}
	continuationKeywords   = []string{"续做", "继续", "接着", "补做"}
	taskTargetKeywords     = []string{
		"本", "卷", "册", "页", "课", "课文", "单词", "错词", "错题", "练习", "练习册", "试卷",
		"校本", "作业", "口算", "作文", "默写", "听写", "知识点", "音标", "录音", "题",
	}
	recitationTaskKeywords = []string{"背诵", "背默", "默背", "背课文", "背作文", "古诗", "诗词", "熟背", "背会"}
	readingTaskKeywords    = []string{"朗读", "跟读", "诵读", "朗诵", "read aloud", "follow reading"}
)

var parsePatternSelection = agentic.PhaseOneTaskParsePattern

type ParsedTask struct {
	Subject                string   `json:"subject"`
	GroupTitle             string   `json:"group_title,omitempty"`
	Title                  string   `json:"title"`
	Type                   string   `json:"type"`
	Confidence             float64  `json:"confidence,omitempty"`
	NeedsReview            bool     `json:"needs_review,omitempty"`
	Notes                  []string `json:"notes,omitempty"`
	ReferenceTitle         string   `json:"reference_title,omitempty"`
	ReferenceAuthor        string   `json:"reference_author,omitempty"`
	ReferenceText          string   `json:"reference_text,omitempty"`
	ReferenceSource        string   `json:"reference_source,omitempty"`
	HideReferenceFromChild bool     `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string   `json:"analysis_mode,omitempty"`
}

type ParseAnalysis struct {
	ParserMode       string                   `json:"parser_mode"`
	DetectedSubjects []string                 `json:"detected_subjects"`
	FormatSignals    []string                 `json:"format_signals"`
	RawLineCount     int                      `json:"raw_line_count"`
	TaskCount        int                      `json:"task_count"`
	GroupCount       int                      `json:"group_count"`
	NeedsReviewCount int                      `json:"needs_review_count"`
	LowConfidenceCnt int                      `json:"low_confidence_count"`
	Notes            []string                 `json:"notes"`
	AgenticPattern   agentic.PatternSelection `json:"agentic_pattern"`
}

type Result struct {
	Status     string        `json:"status"`
	Message    string        `json:"message,omitempty"`
	ParserMode string        `json:"parser_mode,omitempty"`
	Analysis   ParseAnalysis `json:"analysis"`
	Data       []ParsedTask  `json:"data"`
}

type llmResult struct {
	Status string       `json:"status"`
	Data   []ParsedTask `json:"data"`
}

type learningReferenceLLMResult struct {
	Status string                    `json:"status"`
	Data   []learningReferenceLLMRow `json:"data"`
}

type learningReferenceLLMRow struct {
	Index                  int    `json:"index"`
	ReferenceTitle         string `json:"reference_title,omitempty"`
	ReferenceAuthor        string `json:"reference_author,omitempty"`
	ReferenceText          string `json:"reference_text,omitempty"`
	HideReferenceFromChild *bool  `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string `json:"analysis_mode,omitempty"`
}

type looseReferenceBlock struct {
	StartIndex int
	Lines      []string
}

type outlineItem struct {
	Text     string
	Subitems []string
}

type outlineSection struct {
	Subject string
	Items   []outlineItem
}

type structureOutline struct {
	Sections         []outlineSection
	Tasks            []ParsedTask
	DetectedSubjects []string
	FormatSignals    []string
	RawLineCount     int
}

type taskSignalEvaluation struct {
	HasConditional     bool
	HasAudienceScope   bool
	HasCorrection      bool
	HasContinuation    bool
	HasAmbiguousTarget bool
}

type Service struct {
	llmClient llm.Client
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func normalizeSubject(rawSubject string) string {
	subject := whitespacePattern.ReplaceAllString(rawSubject, "")
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "未分类"
	}

	if value, exists := subjectAliases[subject]; exists {
		return value
	}

	for key, value := range subjectAliases {
		if strings.HasPrefix(subject, key) {
			return value
		}
	}

	return subject
}

func normalizedKeywordText(values ...string) string {
	combined := strings.TrimSpace(strings.Join(values, " "))
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(combined, " "))
}

func normalizedSignalParts(values ...string) []string {
	parts := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := trimStructuralMarker(value)
		trimmed = normalizedKeywordText(trimmed)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		parts = append(parts, trimmed)
	}
	return parts
}

func normalizedSignalText(values ...string) string {
	return normalizedKeywordText(normalizedSignalParts(values...)...)
}

func inferTaskType(values ...string) string {
	text := strings.ToLower(normalizedSignalText(values...))
	if text == "" {
		return "homework"
	}

	for _, keyword := range recitationTaskKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return "recitation"
		}
	}

	for _, keyword := range readingTaskKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return "reading"
		}
	}

	return "homework"
}

func normalizeLearningTaskType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "homework":
		return "homework"
	case "recitation", "memorization", "memorize", "poem_recitation", "classical_poem":
		return "recitation"
	case "reading", "read_aloud", "follow_reading", "english_reading":
		return "reading"
	default:
		return normalized
	}
}

func normalizeReferenceSource(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "manual", "extracted", "llm":
		return normalized
	default:
		return normalized
	}
}

func usesReferenceMaterial(taskType string) bool {
	switch normalizeLearningTaskType(taskType) {
	case "recitation", "reading":
		return true
	default:
		return false
	}
}

func normalizeComparisonText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func splitRawMessageLines(rawText string) []string {
	lines := strings.Split(strings.ReplaceAll(rawText, "\r\n", "\n"), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return normalized
}

func isSubjectHeadingLine(line string) bool {
	headingMatch := headingPattern.FindStringSubmatch(strings.TrimSpace(line))
	return len(headingMatch) == 3 && headingKeyPattern.MatchString(whitespacePattern.ReplaceAllString(headingMatch[1], ""))
}

func isStructuralLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if isSubjectHeadingLine(trimmed) {
		return true
	}
	if mainItemPattern.MatchString(trimmed) || bracketedSubItemPattern.MatchString(trimmed) || numberedSubItemPattern.MatchString(trimmed) {
		return true
	}
	if circledSubItemPattern.MatchString(trimmed) || bulletSubItemPattern.MatchString(trimmed) {
		return true
	}
	return false
}

func extractQuotedTitle(text string) string {
	if matches := quotedTitlePattern.FindStringSubmatch(strings.TrimSpace(text)); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if matches := learningActionTitlePattern.FindStringSubmatch(strings.TrimSpace(text)); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func parseReferenceHeader(firstLine, fallbackTitle string) (string, string) {
	line := strings.TrimSpace(firstLine)
	fallbackTitle = strings.TrimSpace(fallbackTitle)
	if line == "" {
		return fallbackTitle, ""
	}

	quotedTitle := extractQuotedTitle(line)
	if matches := referenceHeaderPattern.FindStringSubmatch(line); len(matches) == 3 {
		title := strings.TrimSpace(strings.ReplaceAll(matches[1], "《", ""))
		title = strings.TrimSpace(strings.ReplaceAll(title, "》", ""))
		if quotedTitle != "" {
			title = quotedTitle
		}
		if title == "" {
			title = fallbackTitle
		}
		return title, strings.TrimSpace(matches[2])
	}

	if matches := referenceAuthorPattern.FindStringSubmatch(line); len(matches) == 2 {
		return fallbackTitle, strings.TrimSpace(matches[1])
	}

	if quotedTitle != "" {
		return quotedTitle, ""
	}

	if !strings.ContainsAny(line, "，。！？；,.!?") && utf8.RuneCountInString(line) <= 24 {
		title := strings.TrimSpace(strings.ReplaceAll(line, "《", ""))
		title = strings.TrimSpace(strings.ReplaceAll(title, "》", ""))
		if title == "" {
			title = fallbackTitle
		}
		return title, ""
	}

	return fallbackTitle, ""
}

func looksLikeReferenceBlock(lines []string) bool {
	if len(lines) == 0 {
		return false
	}

	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if utf8.RuneCountInString(joined) < 12 {
		return false
	}

	if len(lines) >= 2 {
		return true
	}
	if strings.ContainsAny(joined, "，。！？；,.!?") {
		return true
	}
	if quotedTitlePattern.MatchString(joined) {
		return true
	}
	return referenceHeaderPattern.MatchString(joined)
}

func collectReferenceBlockAfterLine(lines []string, anchorIndex int) []string {
	if anchorIndex < 0 {
		return nil
	}

	block := make([]string, 0)
	started := false
	for index := anchorIndex + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" {
			if started {
				break
			}
			continue
		}
		if isStructuralLine(trimmed) {
			if started {
				break
			}
			continue
		}

		started = true
		block = append(block, trimmed)
	}

	if !looksLikeReferenceBlock(block) {
		return nil
	}
	return block
}

func extractLooseReferenceBlocks(lines []string) []looseReferenceBlock {
	blocks := make([]looseReferenceBlock, 0)
	current := make([]string, 0)
	startIndex := -1

	flush := func() {
		if looksLikeReferenceBlock(current) {
			blocks = append(blocks, looseReferenceBlock{
				StartIndex: startIndex,
				Lines:      append([]string(nil), current...),
			})
		}
		current = current[:0]
		startIndex = -1
	}

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isStructuralLine(trimmed) {
			flush()
			continue
		}

		if len(current) == 0 {
			startIndex = index
		}
		current = append(current, trimmed)
	}
	flush()

	return blocks
}

func findBestAnchorIndex(lines []string, task ParsedTask) int {
	searchTerms := []string{
		task.Title,
		task.GroupTitle,
		extractQuotedTitle(task.Title),
		extractQuotedTitle(task.GroupTitle),
	}

	bestIndex := -1
	bestScore := -1
	for index, line := range lines {
		normalizedLine := normalizeComparisonText(line)
		if normalizedLine == "" {
			continue
		}

		for _, term := range searchTerms {
			normalizedTerm := normalizeComparisonText(term)
			if normalizedTerm == "" || !strings.Contains(normalizedLine, normalizedTerm) {
				continue
			}

			score := utf8.RuneCountInString(normalizedTerm) * 10
			if isStructuralLine(line) {
				score += 5
			}
			if score > bestScore {
				bestScore = score
				bestIndex = index
			}
		}
	}

	return bestIndex
}

func findReferenceBlockForTask(lines []string, task ParsedTask, extractedTitle string) []string {
	anchorIndex := findBestAnchorIndex(lines, task)
	if block := collectReferenceBlockAfterLine(lines, anchorIndex); len(block) > 0 {
		return block
	}

	titleHint := normalizeComparisonText(extractedTitle)
	if titleHint == "" {
		return nil
	}

	for _, block := range extractLooseReferenceBlocks(lines) {
		if len(block.Lines) == 0 {
			continue
		}
		if strings.Contains(normalizeComparisonText(block.Lines[0]), titleHint) {
			return append([]string(nil), block.Lines...)
		}
	}

	return nil
}

func inferAnalysisMode(taskType, referenceText, title, author string) string {
	normalizedType := normalizeLearningTaskType(taskType)
	lines := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(referenceText), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	joined := strings.Join(lines, "")
	if normalizedType == "reading" {
		if containsASCIIAlpha(joined) {
			return "english_reading"
		}
		return "read_aloud"
	}

	if normalizedType != "recitation" {
		return ""
	}

	shortLineCount := 0
	for _, line := range lines {
		if utf8.RuneCountInString(line) <= 18 {
			shortLineCount++
		}
	}
	hasDynastyMarker := false
	if len(lines) > 0 {
		hasDynastyMarker = referenceHeaderPattern.MatchString(lines[0])
	}
	if (hasDynastyMarker || strings.TrimSpace(author) != "") && len(lines) >= 2 && shortLineCount >= minInt(len(lines), 3) {
		return "classical_poem"
	}
	if strings.TrimSpace(title) != "" && len(lines) >= 2 && shortLineCount == len(lines) && strings.ContainsAny(joined, "，。！？；") {
		return "classical_poem"
	}
	return "text_recitation"
}

func containsASCIIAlpha(value string) bool {
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			return true
		}
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstMeaningfulLine(value string) string {
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func finalizeLearningReferenceFields(task ParsedTask) ParsedTask {
	task.Type = normalizeLearningTaskType(task.Type)
	task.ReferenceTitle = strings.TrimSpace(task.ReferenceTitle)
	task.ReferenceAuthor = strings.TrimSpace(task.ReferenceAuthor)
	task.ReferenceText = strings.TrimSpace(strings.ReplaceAll(task.ReferenceText, "\r\n", "\n"))
	task.ReferenceSource = normalizeReferenceSource(task.ReferenceSource)
	task.AnalysisMode = strings.TrimSpace(task.AnalysisMode)

	if !usesReferenceMaterial(task.Type) {
		task.ReferenceSource = ""
		return task
	}

	fallbackTitle := extractQuotedTitle(task.Title)
	if fallbackTitle == "" {
		fallbackTitle = extractQuotedTitle(task.GroupTitle)
	}

	headerTitle, headerAuthor := parseReferenceHeader(firstMeaningfulLine(task.ReferenceText), fallbackTitle)
	if task.ReferenceTitle == "" {
		task.ReferenceTitle = headerTitle
	}
	if task.ReferenceAuthor == "" {
		task.ReferenceAuthor = headerAuthor
	}
	if task.ReferenceTitle == "" {
		task.ReferenceTitle = fallbackTitle
	}

	if task.ReferenceText != "" {
		if task.AnalysisMode == "" {
			task.AnalysisMode = inferAnalysisMode(task.Type, task.ReferenceText, task.ReferenceTitle, task.ReferenceAuthor)
		}
		if task.Type == "recitation" {
			task.HideReferenceFromChild = true
		}
	}

	if task.ReferenceTitle == "" &&
		task.ReferenceAuthor == "" &&
		task.ReferenceText == "" &&
		task.AnalysisMode == "" &&
		!task.HideReferenceFromChild {
		task.ReferenceSource = ""
	}

	return task
}

func enrichTasksWithLearningReferences(rawText string, tasks []ParsedTask) []ParsedTask {
	lines := splitRawMessageLines(rawText)
	enriched := make([]ParsedTask, 0, len(tasks))

	for _, original := range tasks {
		task := original
		taskType := task.Type
		inferredType := inferTaskType(task.Subject, task.GroupTitle, task.Title)
		if strings.TrimSpace(taskType) == "" || (normalizeLearningTaskType(taskType) == "homework" && usesReferenceMaterial(inferredType)) {
			taskType = inferredType
		}
		task.Type = normalizeLearningTaskType(taskType)

		fallbackTitle := extractQuotedTitle(task.Title)
		if fallbackTitle == "" {
			fallbackTitle = extractQuotedTitle(task.GroupTitle)
		}

		if usesReferenceMaterial(task.Type) && strings.TrimSpace(task.ReferenceText) == "" {
			if block := findReferenceBlockForTask(lines, task, fallbackTitle); len(block) > 0 {
				task.ReferenceText = strings.Join(block, "\n")
				if normalizeReferenceSource(task.ReferenceSource) == "" {
					task.ReferenceSource = "extracted"
				}
				task.Notes = mergeNotes(task.Notes, "已从老师原文自动带出参考内容。")
			}
		}

		enriched = append(enriched, finalizeLearningReferenceFields(task))
	}

	return enriched
}

func looksLikeReferenceLineForTask(taskTitle, line string) bool {
	taskType := inferTaskType(taskTitle)
	if !usesReferenceMaterial(taskType) {
		return false
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" || isStructuralLine(trimmed) {
		return false
	}

	fallbackTitle := extractQuotedTitle(taskTitle)
	headerTitle, headerAuthor := parseReferenceHeader(trimmed, fallbackTitle)
	if headerAuthor != "" {
		return true
	}
	if fallbackTitle != "" && headerTitle == fallbackTitle {
		return true
	}
	if fallbackTitle != "" && strings.ContainsAny(trimmed, "，。！？；,.!?") && utf8.RuneCountInString(trimmed) >= 8 {
		return true
	}
	return false
}

func containsAnyKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func containsConditionalSignal(values ...string) bool {
	text := normalizedSignalText(values...)
	if containsAnyKeyword(text, conditionalSignalKeywords) {
		return true
	}
	for _, pattern := range conditionalSignalPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func containsAudienceSignal(values ...string) bool {
	return containsAnyKeyword(normalizedSignalText(values...), audienceSignalKeywords)
}

func containsCorrectionSignal(values ...string) bool {
	return containsAnyKeyword(normalizedSignalText(values...), correctionKeywords)
}

func containsContinuationSignal(values ...string) bool {
	return containsAnyKeyword(normalizedSignalText(values...), continuationKeywords)
}

func trimStructuralMarker(text string) string {
	trimmed := strings.TrimSpace(text)
	if matches := mainItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if matches := bracketedSubItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if matches := numberedSubItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if matches := circledSubItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	if matches := bulletSubItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}

func hasExplicitTaskTarget(text string) bool {
	text = trimStructuralMarker(text)
	if containsAnyKeyword(text, taskTargetKeywords) {
		return true
	}
	if taskTargetReferencePattern.MatchString(text) {
		return true
	}
	return false
}

func hasAmbiguousTaskTarget(values ...string) bool {
	combined := normalizedSignalText(values...)
	if combined == "" {
		return false
	}
	if !containsCorrectionSignal(combined) && !containsContinuationSignal(combined) {
		return false
	}
	if hasExplicitTaskTarget(combined) {
		return false
	}

	remainder := combined
	for _, keyword := range audienceSignalKeywords {
		remainder = strings.ReplaceAll(remainder, keyword, "")
	}
	for _, keyword := range correctionKeywords {
		remainder = strings.ReplaceAll(remainder, keyword, "")
	}
	for _, keyword := range continuationKeywords {
		remainder = strings.ReplaceAll(remainder, keyword, "")
	}
	remainder = strings.TrimSpace(strings.Trim(remainder, "，,。；;：:（）()"))
	return remainder == "" || len([]rune(remainder)) <= 4
}

func containsReviewSignal(values ...string) bool {
	return containsConditionalSignal(values...) || containsAudienceSignal(values...) || hasAmbiguousTaskTarget(values...)
}

func evaluateTaskSignals(subject, groupTitle, title string) taskSignalEvaluation {
	return taskSignalEvaluation{
		HasConditional:     containsConditionalSignal(groupTitle, title),
		HasAudienceScope:   containsAudienceSignal(groupTitle, title),
		HasCorrection:      containsCorrectionSignal(groupTitle, title),
		HasContinuation:    containsContinuationSignal(groupTitle, title),
		HasAmbiguousTarget: hasAmbiguousTaskTarget(groupTitle, title),
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func mergeNotes(existing []string, extra ...string) []string {
	merged := make([]string, 0, len(existing)+len(extra))
	for _, note := range existing {
		trimmed := strings.TrimSpace(note)
		if trimmed != "" {
			merged = appendUnique(merged, trimmed)
		}
	}
	for _, note := range extra {
		trimmed := strings.TrimSpace(note)
		if trimmed != "" {
			merged = appendUnique(merged, trimmed)
		}
	}
	return merged
}

func buildReviewNotes(subject string, signals taskSignalEvaluation) []string {
	notes := make([]string, 0, 4)
	if subject == "未分类" {
		notes = append(notes, "学科不明确，建议家长确认归类。")
	}
	if signals.HasConditional {
		notes = append(notes, "包含条件性说明，建议家长确认触发条件。")
	}
	if signals.HasAudienceScope {
		notes = append(notes, "作业适用对象不明确，建议家长确认是否针对孩子。")
	}
	if signals.HasAmbiguousTarget {
		notes = append(notes, "订正/续做任务未写明具体对象，建议家长确认完成内容。")
	}
	return notes
}

func scoreTaskConfidence(base float64, subject string, signals taskSignalEvaluation) float64 {
	confidence := base
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	if confidence == 0 {
		confidence = 0.84
	}
	if subject == "未分类" && confidence > 0.68 {
		confidence = 0.68
	}
	if signals.HasConditional && confidence > 0.72 {
		confidence = 0.72
	}
	if signals.HasAudienceScope && confidence > 0.64 {
		confidence = 0.64
	}
	if signals.HasAmbiguousTarget && confidence > 0.58 {
		confidence = 0.58
	}
	if confidence < 0 {
		confidence = 0
	}
	return confidence
}

func normalizeTaskMetadata(subject, groupTitle, title string, confidence float64, needsReview bool, notes []string) (float64, bool, []string) {
	signals := evaluateTaskSignals(subject, groupTitle, title)
	normalizedNotes := mergeNotes(notes, buildReviewNotes(subject, signals)...)
	normalizedConfidence := scoreTaskConfidence(confidence, subject, signals)
	normalizedNeedsReview := needsReview || subject == "未分类" || signals.HasConditional || signals.HasAudienceScope || signals.HasAmbiguousTarget || normalizedConfidence < 0.7
	return normalizedConfidence, normalizedNeedsReview, normalizedNotes
}

func parseSubItemLine(line string, allowLooseNumbering bool) (string, bool) {
	if matches := bracketedSubItemPattern.FindStringSubmatch(line); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), true
	}
	if allowLooseNumbering {
		if matches := numberedSubItemPattern.FindStringSubmatch(line); len(matches) == 2 {
			return strings.TrimSpace(matches[1]), true
		}
		if matches := circledSubItemPattern.FindStringSubmatch(line); len(matches) == 2 {
			return strings.TrimSpace(matches[1]), true
		}
		if matches := bulletSubItemPattern.FindStringSubmatch(line); len(matches) == 2 {
			return strings.TrimSpace(matches[1]), true
		}
	}
	return "", false
}

func normalizeInlineHeadingRemainder(remainder string) (string, bool, bool) {
	if matches := mainItemPattern.FindStringSubmatch(remainder); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), true, false
	}
	if subitem, ok := parseSubItemLine(remainder, true); ok {
		return subitem, false, true
	}
	return strings.TrimSpace(remainder), false, false
}

func flattenSectionsToTasks(sections []outlineSection) []ParsedTask {
	tasks := make([]ParsedTask, 0)
	for _, section := range sections {
		for _, item := range section.Items {
			groupTitle := strings.TrimSpace(item.Text)
			if groupTitle == "" {
				continue
			}

			atomicTitles := make([]string, 0, len(item.Subitems))
			for _, subitem := range item.Subitems {
				trimmed := strings.TrimSpace(subitem)
				if trimmed != "" {
					atomicTitles = append(atomicTitles, trimmed)
				}
			}

			if len(atomicTitles) == 0 {
				atomicTitles = append(atomicTitles, groupTitle)
			}

			for _, atomicTitle := range atomicTitles {
				confidence, needsReview, notes := normalizeTaskMetadata(section.Subject, groupTitle, atomicTitle, 0.84, false, nil)

				tasks = append(tasks, ParsedTask{
					Subject:     section.Subject,
					GroupTitle:  groupTitle,
					Title:       atomicTitle,
					Type:        inferTaskType(section.Subject, groupTitle, atomicTitle),
					Confidence:  confidence,
					NeedsReview: needsReview,
					Notes:       notes,
				})
			}
		}
	}

	return tasks
}

func extractStructureOutline(rawText string) structureOutline {
	lines := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(rawText, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	sections := make([]outlineSection, 0)
	var currentSection *outlineSection
	var currentTask *outlineItem
	skippingReferenceBlock := false
	signals := make([]string, 0)

	markSignal := func(signal string) {
		for _, existing := range signals {
			if existing == signal {
				return
			}
		}
		signals = append(signals, signal)
	}

	ensureSection := func(subject string) *outlineSection {
		normalizedSubject := normalizeSubject(subject)
		for index := range sections {
			if sections[index].Subject == normalizedSubject {
				return &sections[index]
			}
		}

		sections = append(sections, outlineSection{Subject: normalizedSubject, Items: []outlineItem{}})
		return &sections[len(sections)-1]
	}

	flushTask := func() {
		if currentSection == nil || currentTask == nil || strings.TrimSpace(currentTask.Text) == "" {
			currentTask = nil
			return
		}

		item := outlineItem{
			Text:     strings.TrimSpace(currentTask.Text),
			Subitems: make([]string, 0, len(currentTask.Subitems)),
		}
		for _, subitem := range currentTask.Subitems {
			trimmed := strings.TrimSpace(subitem)
			if trimmed != "" {
				item.Subitems = append(item.Subitems, trimmed)
			}
		}
		currentSection.Items = append(currentSection.Items, item)
		currentTask = nil
	}

	for _, line := range lines {
		if skippingReferenceBlock {
			if isStructuralLine(line) {
				skippingReferenceBlock = false
			} else {
				continue
			}
		}

		if containsConditionalSignal(line) {
			markSignal("conditional_notes")
		}
		if containsAudienceSignal(line) {
			markSignal("audience_constraints")
		}
		if containsCorrectionSignal(line) {
			markSignal("correction_tasks")
		}
		if containsContinuationSignal(line) {
			markSignal("continuation_tasks")
		}
		if hasAmbiguousTaskTarget(line) {
			markSignal("ambiguous_targets")
		}

		headingMatch := headingPattern.FindStringSubmatch(line)
		if len(headingMatch) == 3 {
			headingKey := whitespacePattern.ReplaceAllString(headingMatch[1], "")
			if headingKeyPattern.MatchString(headingKey) {
				markSignal("subject_headings")
				flushTask()
				currentSection = ensureSection(headingMatch[1])
				remainder := strings.TrimSpace(headingMatch[2])
				if remainder != "" {
					normalizedRemainder, isMainItem, isSubitem := normalizeInlineHeadingRemainder(remainder)
					if isMainItem {
						markSignal("numbered_tasks")
					}
					if isSubitem {
						markSignal("nested_subtasks")
					}
					currentTask = &outlineItem{Text: normalizedRemainder}
				} else {
					currentTask = nil
				}
				continue
			}
		}

		if subitem, ok := parseSubItemLine(line, currentTask != nil); ok {
			markSignal("nested_subtasks")
			if currentSection == nil {
				currentSection = ensureSection("未分类")
			}
			if currentTask == nil {
				currentTask = &outlineItem{Text: "补充说明"}
			}
			currentTask.Subitems = append(currentTask.Subitems, subitem)
			continue
		}

		if matches := mainItemPattern.FindStringSubmatch(line); len(matches) == 2 {
			markSignal("numbered_tasks")
			flushTask()
			skippingReferenceBlock = false
			if currentSection == nil {
				currentSection = ensureSection("未分类")
			}
			currentTask = &outlineItem{Text: strings.TrimSpace(matches[1])}
			continue
		}

		if currentSection == nil {
			currentSection = ensureSection("未分类")
		}

		if currentTask != nil && len(currentTask.Subitems) == 0 && looksLikeReferenceLineForTask(currentTask.Text, line) {
			flushTask()
			currentTask = nil
			skippingReferenceBlock = true
			continue
		}

		if currentTask == nil {
			currentTask = &outlineItem{Text: line}
			continue
		}

		if len(currentTask.Subitems) > 0 {
			lastIndex := len(currentTask.Subitems) - 1
			currentTask.Subitems[lastIndex] = strings.TrimSpace(currentTask.Subitems[lastIndex] + " " + line)
		} else {
			currentTask.Text = strings.TrimSpace(currentTask.Text + " " + line)
		}
	}

	flushTask()

	detectedSubjects := make([]string, 0, len(sections))
	for _, section := range sections {
		detectedSubjects = append(detectedSubjects, section.Subject)
	}

	previewTasks := flattenSectionsToTasks(sections)
	return structureOutline{
		Sections:         sections,
		Tasks:            previewTasks,
		DetectedSubjects: detectedSubjects,
		FormatSignals:    signals,
		RawLineCount:     len(lines),
	}
}

func normalizeTaskItem(item ParsedTask) (ParsedTask, bool) {
	subject := normalizeSubject(item.Subject)
	title := whitespacePattern.ReplaceAllString(item.Title, " ")
	title = strings.TrimSpace(title)
	if title == "" {
		return ParsedTask{}, false
	}

	groupTitle := item.GroupTitle
	if strings.TrimSpace(groupTitle) == "" {
		groupTitle = title
	}
	groupTitle = strings.TrimSpace(whitespacePattern.ReplaceAllString(groupTitle, " "))

	confidence := item.Confidence

	notes := make([]string, 0, len(item.Notes))
	for _, note := range item.Notes {
		trimmed := strings.TrimSpace(note)
		if trimmed != "" {
			notes = append(notes, trimmed)
		}
	}

	confidence, needsReview, notes := normalizeTaskMetadata(subject, groupTitle, title, confidence, item.NeedsReview, notes)

	taskType := normalizeLearningTaskType(item.Type)
	inferredType := inferTaskType(subject, groupTitle, title)
	if strings.TrimSpace(item.Type) == "" || (taskType == "homework" && usesReferenceMaterial(inferredType)) {
		taskType = inferredType
	}

	normalized := finalizeLearningReferenceFields(ParsedTask{
		Subject:                subject,
		GroupTitle:             groupTitle,
		Title:                  title,
		Type:                   taskType,
		Confidence:             confidence,
		NeedsReview:            needsReview,
		Notes:                  notes,
		ReferenceTitle:         item.ReferenceTitle,
		ReferenceAuthor:        item.ReferenceAuthor,
		ReferenceText:          item.ReferenceText,
		ReferenceSource:        item.ReferenceSource,
		HideReferenceFromChild: item.HideReferenceFromChild,
		AnalysisMode:           item.AnalysisMode,
	})

	return normalized, true
}

func normalizeTaskList(items []ParsedTask) []ParsedTask {
	normalized := make([]ParsedTask, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		task, ok := normalizeTaskItem(item)
		if !ok {
			continue
		}

		key := task.Subject + "\x00" + task.GroupTitle + "\x00" + task.Title
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		normalized = append(normalized, task)
	}

	return normalized
}

func mergeTaskLists(primary []ParsedTask, fallback []ParsedTask) ([]ParsedTask, []string) {
	merged := normalizeTaskList(primary)
	fallbackNormalized := normalizeTaskList(fallback)
	existing := make(map[string]struct{}, len(merged))
	for _, task := range merged {
		key := task.Subject + "\x00" + task.GroupTitle + "\x00" + task.Title
		existing[key] = struct{}{}
	}

	mergedCount := 0
	for _, task := range fallbackNormalized {
		key := task.Subject + "\x00" + task.GroupTitle + "\x00" + task.Title
		if _, exists := existing[key]; exists {
			continue
		}

		existing[key] = struct{}{}
		merged = append(merged, task)
		mergedCount++
	}

	notes := make([]string, 0)
	if mergedCount > 0 {
		notes = append(notes, fmt.Sprintf("LLM 结果缺失的 %d 条任务已由结构兜底补全。", mergedCount))
	}

	return merged, notes
}

func buildAnalysis(parserMode string, structure structureOutline, tasks []ParsedTask, notes []string) ParseAnalysis {
	needsReviewCount := 0
	lowConfidenceCount := 0
	groupKeys := make(map[string]struct{})
	for _, task := range tasks {
		if task.NeedsReview {
			needsReviewCount++
		}
		if task.Confidence < 0.7 {
			lowConfidenceCount++
		}
		groupKeys[task.Subject+"\x00"+task.GroupTitle] = struct{}{}
	}

	return ParseAnalysis{
		ParserMode:       parserMode,
		DetectedSubjects: structure.DetectedSubjects,
		FormatSignals:    structure.FormatSignals,
		RawLineCount:     structure.RawLineCount,
		TaskCount:        len(tasks),
		GroupCount:       len(groupKeys),
		NeedsReviewCount: needsReviewCount,
		LowConfidenceCnt: lowConfidenceCount,
		Notes:            notes,
		AgenticPattern:   parsePatternSelection,
	}
}

func parseFallback(rawText string) Result {
	structure := extractStructureOutline(rawText)
	tasks := normalizeTaskList(structure.Tasks)
	status := "success"
	notes := []string{"当前未使用 LLM，采用结构规则完成任务拆解。"}
	if len(tasks) == 0 {
		status = "failed"
		notes = []string{"未从原文中识别出可创建任务。"}
	}

	return Result{
		Status:     status,
		ParserMode: "rule_fallback",
		Analysis:   buildAnalysis("rule_fallback", structure, tasks, notes),
		Data:       tasks,
	}
}

func buildPrompt(rawText string, structure structureOutline) (string, error) {
	detectedSubjects, err := json.Marshal(structure.DetectedSubjects)
	if err != nil {
		return "", err
	}
	formatSignals, err := json.Marshal(structure.FormatSignals)
	if err != nil {
		return "", err
	}
	candidateTasks, err := json.Marshal(structure.Tasks)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`
你是 StudyClaw 的任务分析 Agent。你的职责不是机械照抄，而是结合老师原始通知和结构提示，做稳健、可落地的任务拆解。

【老师原始内容】
%s

【结构提示】
检测到的学科: %s
检测到的格式信号: %s
规则预解析候选任务: %s

【目标】
1. 输出适合直接创建到孩子今日任务清单里的任务列表。
2. 老师格式可能每天变化，你需要理解语义，不要依赖固定模板。
3. 对同一主任务下的多个子步骤，请拆成多条原子任务，并让它们共享同一个 group_title。
4. 保留条件信息，例如“默写全对可免抄”“部分学生继续订正”。
5. 忽略纯通知、寒暄、表情和无执行动作的内容。
6. 对每条任务给出 confidence（0 到 1）以及 needs_review（是否建议家长确认）。
7. 如果任务带有条件限制、对象不明确、学科不明确，请把 needs_review 设为 true，并在 notes 里写明原因。

【输出要求】
1. 只返回 JSON，不要输出 markdown。
2. JSON 结构必须为:
{
  "status": "success",
  "data": [
    {
      "subject": "数学/语文/英语等学科名称",
      "group_title": "这条原子任务所属的作业分组，例如预习M1U2",
      "title": "适合孩子执行的一条原子任务，必须是可勾选完成的最小动作",
      "type": "homework",
      "confidence": 0.91,
      "needs_review": false,
      "notes": []
    }
  ]
}
3. 如果原文无法形成任务，返回:
{"status": "failed", "data": []}
`, rawText, string(detectedSubjects), string(formatSignals), string(candidateTasks)), nil
}

func stripJSONFence(value string) string {
	trimmed := strings.TrimSpace(value)
	matches := jsonFencePattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}

func (s *Service) invokeLLM(ctx context.Context, rawText string, structure structureOutline) (llmResult, error) {
	prompt, err := buildPrompt(rawText, structure)
	if err != nil {
		return llmResult{}, fmt.Errorf("build llm prompt: %w", err)
	}

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_PARSER_MODEL_NAME",
		Temperature: 0.1,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a careful homework parsing agent. Return valid JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return llmResult{}, err
	}

	var result llmResult
	if err := json.Unmarshal([]byte(stripJSONFence(resultText)), &result); err != nil {
		return llmResult{}, fmt.Errorf("decode llm parser result: %w", err)
	}

	return result, nil
}

func buildLearningReferencePrompt(rawText string, tasks []ParsedTask) (string, error) {
	type promptTask struct {
		Index          int    `json:"index"`
		Subject        string `json:"subject"`
		GroupTitle     string `json:"group_title"`
		Title          string `json:"title"`
		TaskType       string `json:"task_type"`
		ReferenceTitle string `json:"reference_title,omitempty"`
	}

	candidates := make([]promptTask, 0)
	for index, task := range tasks {
		taskType := normalizeLearningTaskType(task.Type)
		if !usesReferenceMaterial(taskType) || strings.TrimSpace(task.ReferenceText) != "" {
			continue
		}

		candidates = append(candidates, promptTask{
			Index:          index,
			Subject:        task.Subject,
			GroupTitle:     task.GroupTitle,
			Title:          task.Title,
			TaskType:       taskType,
			ReferenceTitle: strings.TrimSpace(task.ReferenceTitle),
		})
	}

	if len(candidates) == 0 {
		return "", nil
	}

	candidateJSON, err := json.Marshal(candidates)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`
你是 StudyClaw 的学习素材补全 Agent。你要为“背诵/朗读”任务补全标准参考内容，优先服务孩子的学习分析和陪伴场景。

【老师原始内容】
%s

【待补全任务】
%s

【规则】
1. 只处理 task_type 为 recitation 或 reading 的任务。
2. 优先使用老师原始内容里的标准正文；如果原文没有给出正文，但标题明确且你对标准内容高度确定，也可以补全。
3. 不确定时宁可留空，不要编造。
4. recitation 有参考正文时，hide_reference_from_child 设为 true；reading 默认 false。
5. analysis_mode 只能是 classical_poem / text_recitation / english_reading / read_aloud / ""。
6. 只返回 JSON，不要输出解释。

【输出格式】
{
  "status": "success",
  "data": [
    {
      "index": 0,
      "reference_title": "江畔独步寻花",
      "reference_author": "杜甫",
      "reference_text": "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？",
      "hide_reference_from_child": true,
      "analysis_mode": "classical_poem"
    }
  ]
}
`, rawText, string(candidateJSON)), nil
}

func (s *Service) enrichMissingLearningReferencesWithLLM(ctx context.Context, rawText string, tasks []ParsedTask) ([]ParsedTask, int, error) {
	prompt, err := buildLearningReferencePrompt(rawText, tasks)
	if err != nil {
		return tasks, 0, fmt.Errorf("build learning reference prompt: %w", err)
	}
	if strings.TrimSpace(prompt) == "" {
		return tasks, 0, nil
	}

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_PARSER_MODEL_NAME",
		Temperature: 0.1,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a careful learning reference completion agent. Return valid JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return tasks, 0, err
	}

	var parsed learningReferenceLLMResult
	if err := json.Unmarshal([]byte(stripJSONFence(resultText)), &parsed); err != nil {
		return tasks, 0, fmt.Errorf("decode learning reference result: %w", err)
	}

	if strings.EqualFold(strings.TrimSpace(parsed.Status), "failed") {
		return tasks, 0, nil
	}

	updated := append([]ParsedTask(nil), tasks...)
	appliedCount := 0
	for _, row := range parsed.Data {
		if row.Index < 0 || row.Index >= len(updated) {
			continue
		}

		task := updated[row.Index]
		changed := false

		if task.ReferenceTitle == "" && strings.TrimSpace(row.ReferenceTitle) != "" {
			task.ReferenceTitle = strings.TrimSpace(row.ReferenceTitle)
			changed = true
		}
		if task.ReferenceAuthor == "" && strings.TrimSpace(row.ReferenceAuthor) != "" {
			task.ReferenceAuthor = strings.TrimSpace(row.ReferenceAuthor)
			changed = true
		}
		if task.ReferenceText == "" && strings.TrimSpace(row.ReferenceText) != "" {
			task.ReferenceText = strings.TrimSpace(row.ReferenceText)
			changed = true
		}
		if task.AnalysisMode == "" && strings.TrimSpace(row.AnalysisMode) != "" {
			task.AnalysisMode = strings.TrimSpace(row.AnalysisMode)
			changed = true
		}
		if row.HideReferenceFromChild != nil && *row.HideReferenceFromChild && !task.HideReferenceFromChild {
			task.HideReferenceFromChild = true
			changed = true
		}
		if changed && normalizeReferenceSource(task.ReferenceSource) == "" {
			task.ReferenceSource = "llm"
		}

		task = finalizeLearningReferenceFields(task)
		if changed {
			task.Notes = mergeNotes(task.Notes, "参考素材已由 LLM 自动补全。")
			appliedCount++
		}
		updated[row.Index] = task
	}

	return updated, appliedCount, nil
}

func (s *Service) Parse(ctx context.Context, rawText string) (Result, error) {
	structure := extractStructureOutline(rawText)
	fallbackResult := parseFallback(rawText)
	fallbackTasks := enrichTasksWithLearningReferences(rawText, fallbackResult.Data)

	result := Result{
		Status:     fallbackResult.Status,
		Message:    fallbackResult.Message,
		ParserMode: fallbackResult.ParserMode,
		Analysis:   buildAnalysis(fallbackResult.ParserMode, structure, fallbackTasks, append([]string(nil), fallbackResult.Analysis.Notes...)),
		Data:       fallbackTasks,
	}

	if s.llmClient != nil {
		llmResult, err := s.invokeLLM(ctx, rawText, structure)
		if err != nil {
			result.Analysis.Notes = append(result.Analysis.Notes, "LLM 调用失败，已自动降级到规则解析: "+err.Error())
			result.Message = err.Error()
		} else {
			llmTasks := normalizeTaskList(llmResult.Data)
			if llmResult.Status == "success" && len(llmTasks) > 0 {
				mergedTasks, mergeNotes := mergeTaskLists(llmTasks, fallbackTasks)
				mergedTasks = enrichTasksWithLearningReferences(rawText, mergedTasks)
				analysisNotes := []string{"已使用 LLM 结合结构提示完成语义拆解，并自动创建任务。"}
				analysisNotes = append(analysisNotes, mergeNotes...)
				result = Result{
					Status:     "success",
					ParserMode: "llm_hybrid",
					Analysis:   buildAnalysis("llm_hybrid", structure, mergedTasks, analysisNotes),
					Data:       mergedTasks,
				}
			}
		}

		enrichedTasks, enrichedCount, err := s.enrichMissingLearningReferencesWithLLM(ctx, rawText, result.Data)
		if err == nil && enrichedCount > 0 {
			result.Data = normalizeTaskList(enrichedTasks)
			result.Analysis = buildAnalysis(result.ParserMode, structure, result.Data, append(result.Analysis.Notes, fmt.Sprintf("缺失的 %d 条朗读/背诵任务已由 LLM 自动补全参考素材。", enrichedCount)))
		}
	}

	return result, nil
}
