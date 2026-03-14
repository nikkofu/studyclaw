package recitationanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

var recitationJSONFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

const (
	parserModeRuleFallback = "rule_fallback"
	parserModeLLMHybrid    = "llm_hybrid"

	lineStatusMatched = "matched"
	lineStatusPartial = "partial"
	lineStatusMissing = "missing"
)

var allowedLineStatuses = []string{
	lineStatusMatched,
	lineStatusPartial,
	lineStatusMissing,
}

type AnalyzeInput struct {
	Transcript    string            `json:"transcript"`
	Scene         string            `json:"scene,omitempty"`
	Locale        string            `json:"locale,omitempty"`
	ReferenceText string            `json:"reference_text,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type Analysis struct {
	Status               string         `json:"status"`
	ParserMode           string         `json:"parser_mode"`
	Scene                string         `json:"scene"`
	RecognizedTitle      string         `json:"recognized_title,omitempty"`
	RecognizedAuthor     string         `json:"recognized_author,omitempty"`
	ReferenceTitle       string         `json:"reference_title,omitempty"`
	ReferenceAuthor      string         `json:"reference_author,omitempty"`
	ReferenceText        string         `json:"reference_text,omitempty"`
	NormalizedTranscript string         `json:"normalized_transcript"`
	ReconstructedText    string         `json:"reconstructed_text,omitempty"`
	CompletionRatio      float64        `json:"completion_ratio"`
	NeedsRetry           bool           `json:"needs_retry"`
	Summary              string         `json:"summary"`
	Suggestion           string         `json:"suggestion"`
	Issues               []string       `json:"issues,omitempty"`
	MatchedLines         []LineAnalysis `json:"matched_lines"`
}

type LineAnalysis struct {
	Index      int     `json:"index"`
	Expected   string  `json:"expected"`
	Observed   string  `json:"observed,omitempty"`
	MatchRatio float64 `json:"match_ratio"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes,omitempty"`
}

type llmAnalysis struct {
	Status            string         `json:"status"`
	RecognizedTitle   string         `json:"recognized_title"`
	RecognizedAuthor  string         `json:"recognized_author"`
	ReferenceTitle    string         `json:"reference_title"`
	ReferenceAuthor   string         `json:"reference_author"`
	ReferenceText     string         `json:"reference_text"`
	ReconstructedText string         `json:"reconstructed_text"`
	CompletionRatio   float64        `json:"completion_ratio"`
	NeedsRetry        bool           `json:"needs_retry"`
	Summary           string         `json:"summary"`
	Suggestion        string         `json:"suggestion"`
	Issues            []string       `json:"issues"`
	MatchedLines      []LineAnalysis `json:"matched_lines"`
}

type Service struct {
	llmClient llm.Client
}

type referenceDoc struct {
	title     string
	author    string
	text      string
	bodyLines []string
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func (s *Service) Analyze(ctx context.Context, input AnalyzeInput) (Analysis, error) {
	input.Transcript = strings.TrimSpace(input.Transcript)
	input.Scene = normalizeScene(input.Scene)
	input.Locale = strings.TrimSpace(input.Locale)
	input.ReferenceText = strings.TrimSpace(input.ReferenceText)

	fallback := analyzeWithFallback(input)
	if s.llmClient == nil || input.Transcript == "" {
		return fallback, nil
	}

	contextJSON, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return fallback, nil
	}

	prompt := fmt.Sprintf(`
你是 StudyClaw 的背诵分析代理。你要处理的是“孩子背诵古诗词/课文时，被 STT 识别得很乱”的场景。

你需要根据 noisy transcript 推理：
1. 背诵标题和作者
2. 正确的参考原文
3. 每一句和原文的匹配情况
4. 是否需要重背

输入 JSON：
%s

输出要求：
1. 只返回 JSON，不要输出解释文字。
2. 如果提供了 reference_text，把它视为最高优先级的标准原文。
3. 即使 transcript 有大量同音字、错字，也要尽量恢复最可能的古诗词标题、作者和正文。
4. matched_lines 里的 status 只能是 matched / partial / missing。
5. completion_ratio 取 0 到 1。

输出格式：
{
  "status": "success",
  "recognized_title": "江畔独步寻花",
  "recognized_author": "杜甫",
  "reference_title": "江畔独步寻花",
  "reference_author": "杜甫",
  "reference_text": "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？",
  "reconstructed_text": "江畔独步寻花 杜甫 黄师塔前江水东 春光懒困倚微风 桃花一簇开无主 可爱深红爱浅红",
  "completion_ratio": 0.82,
  "needs_retry": false,
  "summary": "标题和正文主体已经对上，大部分内容背出来了。",
  "suggestion": "把少量不稳的句子再熟读一遍后重新背诵。",
  "issues": ["第一句个别字音对应不稳"],
  "matched_lines": [
    {
      "index": 1,
      "expected": "黄师塔前江水东，春光懒困倚微风。",
      "observed": "黄思帕钳将水东春光染会以微风",
      "match_ratio": 0.72,
      "status": "partial",
      "notes": "主体对上，但有同音字替换"
    }
  ]
}
`, string(contextJSON))

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey:  "LLM_RECITATION_MODEL_NAME",
		Temperature:  0.1,
		DefaultModel: "",
		Messages: []llm.Message{
			{Role: "system", Content: "You are a recitation analysis assistant for StudyClaw. Return valid JSON only."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return fallback, nil
	}

	var parsed llmAnalysis
	if err := json.Unmarshal([]byte(stripRecitationJSONFence(resultText)), &parsed); err != nil {
		return fallback, nil
	}

	return normalizeLLMAnalysis(parsed, input, fallback), nil
}

func analyzeWithFallback(input AnalyzeInput) Analysis {
	scene := normalizeScene(input.Scene)
	reference := parseReferenceText(input.ReferenceText)
	normalizedTranscript := strings.TrimSpace(input.Transcript)
	compactTranscript := normalizeComparableText(normalizedTranscript)

	if compactTranscript == "" {
		return Analysis{
			Status:               "success",
			ParserMode:           parserModeRuleFallback,
			Scene:                scene,
			ReferenceTitle:       reference.title,
			ReferenceAuthor:      reference.author,
			ReferenceText:        reference.text,
			NormalizedTranscript: normalizedTranscript,
			CompletionRatio:      0,
			NeedsRetry:           true,
			Summary:              "没有识别到可用的背诵内容。",
			Suggestion:           "请再说一遍，并尽量把标题和正文完整读出来。",
			Issues:               []string{"没有识别到有效语音内容"},
			MatchedLines:         []LineAnalysis{},
		}
	}

	recognizedTitle, remainingTranscript, titleDetected := detectReferencePart(
		compactTranscript,
		reference.title,
		0.34,
	)
	recognizedAuthor, remainingTranscript, _ := detectReferencePart(
		remainingTranscript,
		reference.author,
		0.5,
	)
	if !titleDetected {
		remainingTranscript = compactTranscript
	}

	lineAssessments := buildLineAssessments(remainingTranscript, reference.bodyLines)
	completionRatio := computeCompletionRatio(lineAssessments)
	issues := buildFallbackIssues(reference, recognizedTitle, recognizedAuthor, lineAssessments)
	summary := buildFallbackSummary(reference, recognizedTitle, completionRatio, issues)
	suggestion := buildFallbackSuggestion(reference, completionRatio, lineAssessments)
	needsRetry := completionRatio < 0.86

	if reference.title == "" && len(reference.bodyLines) == 0 {
		return Analysis{
			Status:               "success",
			ParserMode:           parserModeRuleFallback,
			Scene:                scene,
			RecognizedTitle:      recognizedTitle,
			RecognizedAuthor:     recognizedAuthor,
			NormalizedTranscript: normalizedTranscript,
			CompletionRatio:      0,
			NeedsRetry:           true,
			Summary:              "已记录本次背诵，但缺少参考原文，暂时无法可靠判断是否背对。",
			Suggestion:           "建议输入标题、作者和正文，或接入 LLM 后再做自动识别比对。",
			Issues:               []string{"缺少参考原文，当前只能保留 transcript，不能做可靠对照"},
			MatchedLines:         []LineAnalysis{},
		}
	}

	return Analysis{
		Status:               "success",
		ParserMode:           parserModeRuleFallback,
		Scene:                scene,
		RecognizedTitle:      recognizedTitle,
		RecognizedAuthor:     recognizedAuthor,
		ReferenceTitle:       reference.title,
		ReferenceAuthor:      reference.author,
		ReferenceText:        reference.text,
		NormalizedTranscript: normalizedTranscript,
		CompletionRatio:      completionRatio,
		NeedsRetry:           needsRetry,
		Summary:              summary,
		Suggestion:           suggestion,
		Issues:               issues,
		MatchedLines:         lineAssessments,
	}
}

func normalizeLLMAnalysis(parsed llmAnalysis, input AnalyzeInput, fallback Analysis) Analysis {
	result := Analysis{
		Status:               "success",
		ParserMode:           parserModeLLMHybrid,
		Scene:                normalizeScene(input.Scene),
		RecognizedTitle:      strings.TrimSpace(parsed.RecognizedTitle),
		RecognizedAuthor:     strings.TrimSpace(parsed.RecognizedAuthor),
		ReferenceTitle:       strings.TrimSpace(parsed.ReferenceTitle),
		ReferenceAuthor:      strings.TrimSpace(parsed.ReferenceAuthor),
		ReferenceText:        strings.TrimSpace(parsed.ReferenceText),
		NormalizedTranscript: strings.TrimSpace(input.Transcript),
		ReconstructedText:    strings.TrimSpace(parsed.ReconstructedText),
		CompletionRatio:      clampRatio(parsed.CompletionRatio),
		NeedsRetry:           parsed.NeedsRetry,
		Summary:              strings.TrimSpace(parsed.Summary),
		Suggestion:           strings.TrimSpace(parsed.Suggestion),
		Issues:               sanitizeIssues(parsed.Issues),
		MatchedLines:         sanitizeLineAssessments(parsed.MatchedLines),
	}

	if result.RecognizedTitle == "" {
		result.RecognizedTitle = fallback.RecognizedTitle
	}
	if result.RecognizedAuthor == "" {
		result.RecognizedAuthor = fallback.RecognizedAuthor
	}
	if result.ReferenceTitle == "" {
		result.ReferenceTitle = fallback.ReferenceTitle
	}
	if result.ReferenceAuthor == "" {
		result.ReferenceAuthor = fallback.ReferenceAuthor
	}
	if result.ReferenceText == "" {
		result.ReferenceText = fallback.ReferenceText
	}
	if result.CompletionRatio == 0 && fallback.CompletionRatio > 0 {
		result.CompletionRatio = fallback.CompletionRatio
	}
	if result.Summary == "" {
		result.Summary = fallback.Summary
	}
	if result.Suggestion == "" {
		result.Suggestion = fallback.Suggestion
	}
	if len(result.Issues) == 0 {
		result.Issues = fallback.Issues
	}
	if len(result.MatchedLines) == 0 {
		result.MatchedLines = fallback.MatchedLines
	}
	return result
}

func parseReferenceText(raw string) referenceDoc {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return referenceDoc{}
	}

	rawLines := strings.Split(trimmed, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return referenceDoc{}
	}

	title, author := extractReferenceHeader(lines[0])
	bodyLines := lines[1:]
	if len(bodyLines) == 0 {
		bodyLines = splitFallbackBodyLines(lines[0])
	}
	if len(bodyLines) == 0 && len(lines) > 1 {
		bodyLines = lines[1:]
	}

	return referenceDoc{
		title:     title,
		author:    author,
		text:      trimmed,
		bodyLines: bodyLines,
	}
}

func extractReferenceHeader(header string) (string, string) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return "", ""
	}

	if strings.Contains(trimmed, "《") && strings.Contains(trimmed, "》") {
		start := strings.Index(trimmed, "《")
		end := strings.Index(trimmed, "》")
		if start >= 0 && end > start {
			return strings.TrimSpace(trimmed[start+len("《") : end]), strings.TrimSpace(trimmed[end+len("》"):])
		}
	}

	titleEnd := len(trimmed)
	for _, marker := range []string{"【", "[", "（", "(", "-", "—"} {
		if index := strings.Index(trimmed, marker); index >= 0 && index < titleEnd {
			titleEnd = index
		}
	}
	title := strings.TrimSpace(trimmed[:titleEnd])
	author := strings.TrimSpace(trimmed[titleEnd:])
	for _, marker := range [][2]string{
		{"【", "】"},
		{"[", "]"},
		{"（", "）"},
		{"(", ")"},
	} {
		if strings.HasPrefix(author, marker[0]) {
			if end := strings.Index(author, marker[1]); end >= 0 {
				author = strings.TrimSpace(author[end+len(marker[1]):])
			}
		}
	}
	author = strings.Trim(author, "【】[]（）() -—")
	for _, dynasty := range []string{"唐", "宋", "元", "明", "清"} {
		author = strings.TrimPrefix(author, dynasty)
	}
	return title, author
}

func splitFallbackBodyLines(line string) []string {
	candidates := strings.FieldsFunc(line, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '；'
	})
	result := make([]string, 0, len(candidates))
	for _, item := range candidates {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if utf8.RuneCountInString(item) < 8 {
			continue
		}
		result = append(result, item)
	}
	return result
}

func detectReferencePart(transcript, target string, threshold float64) (string, string, bool) {
	target = strings.TrimSpace(target)
	if target == "" || transcript == "" {
		return "", transcript, false
	}

	targetComparable := normalizeComparableText(target)
	if targetComparable == "" {
		return "", transcript, false
	}

	transcriptRunes := []rune(transcript)
	targetLen := len([]rune(targetComparable))
	minLen := maxInt(1, targetLen-2)
	maxLen := targetLen + 6
	maxStart := minInt(len(transcriptRunes)-1, targetLen/2+4)
	bestStart := 0
	bestEnd := 0
	bestScore := 0.0

	for start := 0; start <= maxStart; start++ {
		localMaxLen := minInt(len(transcriptRunes)-start, maxLen)
		if localMaxLen < minLen {
			continue
		}
		for candidateLen := minLen; candidateLen <= localMaxLen; candidateLen++ {
			candidate := string(transcriptRunes[start : start+candidateLen])
			score := similarityScore(targetComparable, candidate)
			score -= float64(start) * 0.02
			score -= float64(absInt(candidateLen-targetLen)) * 0.01
			if start == 0 {
				score += 0.04
			}
			if score > bestScore {
				bestScore = score
				bestStart = start
				bestEnd = start + candidateLen
			}
		}
	}

	if bestScore < threshold || bestEnd <= bestStart {
		return "", transcript, false
	}
	return target, string(transcriptRunes[bestEnd:]), true
}

func buildLineAssessments(transcript string, referenceLines []string) []LineAnalysis {
	if len(referenceLines) == 0 {
		return []LineAnalysis{}
	}

	normalizedReferenceLines := make([]string, 0, len(referenceLines))
	for _, line := range referenceLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		normalizedReferenceLines = append(normalizedReferenceLines, trimmed)
	}
	if len(normalizedReferenceLines) == 0 {
		return []LineAnalysis{}
	}

	segments := splitObservedByReference(transcript, normalizedReferenceLines)
	assessments := make([]LineAnalysis, 0, len(normalizedReferenceLines))
	for index, expected := range normalizedReferenceLines {
		observed := ""
		if index < len(segments) {
			observed = segments[index]
		}
		ratio := similarityScore(expected, observed)
		status := lineStatusMissing
		notes := "这一句和原文差距较大，建议先看原文再重背。"
		switch {
		case ratio >= 0.82:
			status = lineStatusMatched
			notes = "这一句整体比较稳，主体内容已经对上。"
		case ratio >= 0.48:
			status = lineStatusPartial
			notes = "这一句主体对上了，但有同音字、漏字或少量顺序偏差。"
		}
		assessments = append(assessments, LineAnalysis{
			Index:      index + 1,
			Expected:   expected,
			Observed:   observed,
			MatchRatio: clampRatio(ratio),
			Status:     status,
			Notes:      notes,
		})
	}
	return assessments
}

func splitObservedByReference(transcript string, referenceLines []string) []string {
	segments := splitObservedByReferenceWithDP(transcript, referenceLines)
	if len(segments) == len(referenceLines) {
		return segments
	}
	return splitObservedByReferenceHeuristic(transcript, referenceLines)
}

func splitObservedByReferenceWithDP(transcript string, referenceLines []string) []string {
	if len(referenceLines) == 0 {
		return []string{}
	}

	normalizedTranscript := normalizeComparableText(transcript)
	transcriptRunes := []rune(normalizedTranscript)
	transcriptLen := len(transcriptRunes)

	normalizedReferenceLines := make([]string, 0, len(referenceLines))
	for _, line := range referenceLines {
		normalizedReferenceLines = append(normalizedReferenceLines, normalizeComparableText(line))
	}

	const negativeInf = -1e9
	dp := make([][]float64, len(referenceLines)+1)
	prev := make([][]int, len(referenceLines)+1)
	for index := range dp {
		dp[index] = make([]float64, transcriptLen+1)
		prev[index] = make([]int, transcriptLen+1)
		for offset := range dp[index] {
			dp[index][offset] = negativeInf
			prev[index][offset] = -1
		}
	}
	dp[0][0] = 0

	for lineIndex := 1; lineIndex <= len(normalizedReferenceLines); lineIndex++ {
		expected := normalizedReferenceLines[lineIndex-1]
		expectedLen := len([]rune(expected))

		for consumed := 0; consumed <= transcriptLen; consumed++ {
			if dp[lineIndex-1][consumed] <= negativeInf/2 {
				continue
			}

			remainingChars := transcriptLen - consumed
			maxSegLen := minInt(remainingChars, expectedLen*3/2+6)
			if lineIndex == len(normalizedReferenceLines) {
				maxSegLen = remainingChars
			}
			if maxSegLen < 0 {
				continue
			}

			for segLen := 0; segLen <= maxSegLen; segLen++ {
				end := consumed + segLen
				observed := string(transcriptRunes[consumed:end])
				score := segmentAlignmentScore(expected, observed)
				total := dp[lineIndex-1][consumed] + score
				if total > dp[lineIndex][end] {
					dp[lineIndex][end] = total
					prev[lineIndex][end] = consumed
				}
			}
		}
	}

	bestEnd := transcriptLen
	if prev[len(normalizedReferenceLines)][bestEnd] < 0 {
		return nil
	}

	boundaries := make([]int, len(normalizedReferenceLines)+1)
	boundaries[len(normalizedReferenceLines)] = bestEnd
	for lineIndex := len(normalizedReferenceLines); lineIndex > 0; lineIndex-- {
		start := prev[lineIndex][boundaries[lineIndex]]
		if start < 0 {
			return nil
		}
		boundaries[lineIndex-1] = start
	}

	segments := make([]string, 0, len(normalizedReferenceLines))
	for index := 0; index < len(normalizedReferenceLines); index++ {
		segment := string(transcriptRunes[boundaries[index]:boundaries[index+1]])
		segments = append(segments, strings.TrimSpace(segment))
	}
	return segments
}

func splitObservedByReferenceHeuristic(transcript string, referenceLines []string) []string {
	remaining := []rune(strings.TrimSpace(transcript))
	segments := make([]string, 0, len(referenceLines))

	for index, expected := range referenceLines {
		if len(remaining) == 0 {
			segments = append(segments, "")
			continue
		}
		if index == len(referenceLines)-1 {
			segments = append(segments, string(remaining))
			remaining = nil
			continue
		}

		expectedComparable := normalizeComparableText(expected)
		expectedLen := len([]rune(expectedComparable))
		minLen := maxInt(1, expectedLen*3/5)
		maxLen := minInt(len(remaining), expectedLen*3/2+4)
		if maxLen < minLen {
			maxLen = minInt(len(remaining), expectedLen)
		}

		nextComparable := normalizeComparableText(referenceLines[index+1])
		bestLen := minLen
		bestScore := -1.0
		for candidateLen := minLen; candidateLen <= maxLen; candidateLen++ {
			prefix := string(remaining[:candidateLen])
			score := similarityScore(expectedComparable, prefix)
			if nextComparable != "" && candidateLen < len(remaining) {
				nextWindowLen := minInt(len(remaining)-candidateLen, len([]rune(nextComparable))+4)
				if nextWindowLen > 0 {
					nextObserved := string(remaining[candidateLen : candidateLen+nextWindowLen])
					score += similarityScore(nextComparable, nextObserved) * 0.25
				}
			}
			score -= float64(absInt(candidateLen-expectedLen)) * 0.01
			if score > bestScore {
				bestScore = score
				bestLen = candidateLen
			}
		}

		segments = append(segments, string(remaining[:bestLen]))
		remaining = remaining[bestLen:]
	}

	return segments
}

func segmentAlignmentScore(expected, observed string) float64 {
	expectedComparable := normalizeComparableText(expected)
	observedComparable := normalizeComparableText(observed)
	expectedLen := len([]rune(expectedComparable))
	observedLen := len([]rune(observedComparable))

	score := similarityScore(expectedComparable, observedComparable) * 1.15
	score -= float64(absInt(observedLen-expectedLen)) * 0.015
	if observedLen == 0 && expectedLen > 0 {
		score -= 0.18
	}
	if expectedLen > 0 && observedLen > expectedLen*2 {
		score -= 0.1
	}
	return score
}

func computeCompletionRatio(lines []LineAnalysis) float64 {
	if len(lines) == 0 {
		return 0
	}

	totalWeight := 0
	accumulated := 0.0
	for _, line := range lines {
		weight := maxInt(1, utf8.RuneCountInString(normalizeComparableText(line.Expected)))
		totalWeight += weight
		accumulated += line.MatchRatio * float64(weight)
	}
	if totalWeight == 0 {
		return 0
	}
	return clampRatio(accumulated / float64(totalWeight))
}

func buildFallbackIssues(reference referenceDoc, recognizedTitle, recognizedAuthor string, lines []LineAnalysis) []string {
	issues := []string{}
	if reference.title != "" && recognizedTitle == "" {
		issues = append(issues, "标题没有被稳定识别出来，建议把标题和作者也一起读清楚。")
	}
	if reference.author != "" && recognizedAuthor == "" {
		issues = append(issues, "作者信息没有被稳定识别出来。")
	}

	weakLineIndexes := []string{}
	for _, line := range lines {
		if line.Status != lineStatusMatched {
			weakLineIndexes = append(weakLineIndexes, fmt.Sprintf("第 %d 句", line.Index))
		}
	}
	if len(weakLineIndexes) > 0 {
		issues = append(issues, strings.Join(weakLineIndexes, "、")+" 还不够稳。")
	}
	return issues
}

func buildFallbackSummary(reference referenceDoc, recognizedTitle string, completionRatio float64, issues []string) string {
	title := recognizedTitle
	if title == "" {
		title = reference.title
	}
	titlePrefix := ""
	if title != "" {
		titlePrefix = "《" + title + "》"
	}

	switch {
	case reference.title == "" && len(reference.bodyLines) == 0:
		return "已记录本次背诵内容，但暂时没有参考原文可供核对。"
	case completionRatio >= 0.9 && len(issues) == 0:
		return titlePrefix + " 基本完整背出来了，标题和正文都比较稳。"
	case completionRatio >= 0.7:
		return titlePrefix + " 主体内容已经对上，大部分句子能听出是在背这首作品。"
	case completionRatio >= 0.45:
		return titlePrefix + " 有一部分内容对上了，但还存在明显错漏和同音替换。"
	default:
		return titlePrefix + " 当前和参考原文差距还比较大，暂时不能判定为已经背熟。"
	}
}

func buildFallbackSuggestion(reference referenceDoc, completionRatio float64, lines []LineAnalysis) string {
	if reference.title == "" && len(reference.bodyLines) == 0 {
		return "先输入标题、作者和正文，再重新背诵，系统才能可靠比对。"
	}

	weakLines := []string{}
	for _, line := range lines {
		if line.Status != lineStatusMatched {
			weakLines = append(weakLines, fmt.Sprintf("第 %d 句", line.Index))
		}
	}

	switch {
	case completionRatio >= 0.9 && len(weakLines) == 0:
		return "已经可以尝试更流畅、更有节奏地完整背一遍。"
	case completionRatio >= 0.7:
		return "重点回看 " + strings.Join(weakLines, "、") + "，熟读后再完整背一遍。"
	case completionRatio >= 0.45:
		return "先逐句对照原文熟读，再从标题开始重新背诵，会更容易稳住。"
	default:
		return "建议先看着原文朗读几遍，把标题、作者和每一句顺下来后再重背。"
	}
}

func sanitizeLineAssessments(lines []LineAnalysis) []LineAnalysis {
	if len(lines) == 0 {
		return nil
	}
	result := make([]LineAnalysis, 0, len(lines))
	for index, line := range lines {
		status := strings.TrimSpace(line.Status)
		if !slices.Contains(allowedLineStatuses, status) {
			status = lineStatusPartial
		}
		result = append(result, LineAnalysis{
			Index:      maxInt(1, line.Index),
			Expected:   strings.TrimSpace(line.Expected),
			Observed:   strings.TrimSpace(line.Observed),
			MatchRatio: clampRatio(line.MatchRatio),
			Status:     status,
			Notes:      strings.TrimSpace(line.Notes),
		})
		if result[index].Expected == "" {
			result[index].Expected = fmt.Sprintf("第 %d 句", result[index].Index)
		}
	}
	return result
}

func sanitizeIssues(issues []string) []string {
	result := make([]string, 0, len(issues))
	for _, item := range issues {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func normalizeScene(scene string) string {
	scene = strings.TrimSpace(strings.ToLower(scene))
	switch scene {
	case "", "recitation", "poem", "poem_recitation", "classical_poem":
		return "recitation"
	case "reading":
		return "reading"
	default:
		return scene
	}
}

func normalizeComparableText(raw string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case isCJK(r):
			builder.WriteRune(r)
		default:
			// Drop punctuation and spaces for fuzzy comparison.
		}
	}
	return builder.String()
}

func isCJK(r rune) bool {
	return (r >= 0x3400 && r <= 0x9FFF) || (r >= 0xF900 && r <= 0xFAFF)
}

func similarityScore(expected, observed string) float64 {
	left := []rune(normalizeComparableText(expected))
	right := []rune(normalizeComparableText(observed))
	switch {
	case len(left) == 0 && len(right) == 0:
		return 1
	case len(left) == 0 || len(right) == 0:
		return 0
	}

	distance := levenshteinDistance(left, right)
	maxLen := maxInt(len(left), len(right))
	return clampRatio(1 - float64(distance)/float64(maxLen))
}

func levenshteinDistance(left, right []rune) int {
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}

	prev := make([]int, len(right)+1)
	for index := range prev {
		prev[index] = index
	}

	for leftIndex := 1; leftIndex <= len(left); leftIndex++ {
		current := make([]int, len(right)+1)
		current[0] = leftIndex
		for rightIndex := 1; rightIndex <= len(right); rightIndex++ {
			cost := 0
			if left[leftIndex-1] != right[rightIndex-1] {
				cost = 1
			}
			current[rightIndex] = minInt(
				minInt(current[rightIndex-1]+1, prev[rightIndex]+1),
				prev[rightIndex-1]+cost,
			)
		}
		prev = current
	}

	return prev[len(right)]
}

func stripRecitationJSONFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	matches := recitationJSONFencePattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}

func clampRatio(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
