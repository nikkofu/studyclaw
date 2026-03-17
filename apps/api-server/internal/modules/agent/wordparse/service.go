package wordparse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

var (
	jsonFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
)

type ParsedWord struct {
	Text    string `json:"text"`
	Meaning string `json:"meaning"`
}

type llmResult struct {
	Status string       `json:"status"`
	Data   []ParsedWord `json:"data"`
}

type Service struct {
	llmClient llm.Client
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func (s *Service) Parse(ctx context.Context, rawText string) ([]ParsedWord, error) {
	if s.llmClient == nil {
		return parseRuleBased(rawText), nil
	}

	prompt := fmt.Sprintf(`
你是 StudyClaw 的单词清单解析 Agent。你的任务是将家长录入的原始文本拆解为“单词/短语”与“中文释义”的结构化对。

【输入内容】
%s

【目标】
1. 识别每一行中的主词（通常是英文或中文待默写词）和其对应的解释。
2. 处理各种分隔符：空格、破折号、等号、括号等。
3. 如果一行只有单词没有释义，将 meaning 设为空字符串。
4. 修正明显的拼写或格式杂质。

【输出要求】
1. 只返回 JSON。
2. JSON 结构必须为:
{
  "status": "success",
  "data": [
    {"text": "apple", "meaning": "苹果"},
    {"text": "soft", "meaning": "柔软的"}
  ]
}
`, rawText)

	log.Printf("[WordParse.Service] Parse: Sending prompt to LLM:\n%s", prompt)
	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_PARSER_MODEL_NAME",
		Temperature: 0.1,
		Messages: []llm.Message{
			{Role: "system", Content: "You are a professional vocabulary parser. Return valid JSON only."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		log.Printf("[WordParse.Service] Parse: LLM request failed: %v", err)
		return parseRuleBased(rawText), nil // Fallback
	}
	log.Printf("[WordParse.Service] Parse: Received raw response from LLM:\n%s", resultText)

	var result llmResult
	if err := json.Unmarshal([]byte(stripJSONFence(resultText)), &result); err != nil {
		return parseRuleBased(rawText), nil // Fallback
	}

	return result.Data, nil
}

func parseRuleBased(rawText string) []ParsedWord {
	lines := strings.Split(rawText, "\n")
	var result []ParsedWord

	// 分隔符正则：包含冒号、等号、破折号、或者两个以上的空格
	sepRegex := regexp.MustCompile(`[:=—\-\t]| {2,}`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := sepRegex.Split(line, 2)
		if len(parts) == 2 {
			result = append(result, ParsedWord{
				Text:    strings.TrimSpace(parts[0]),
				Meaning: strings.TrimSpace(parts[1]),
			})
		} else {
			// 如果没有显式分隔符，则整行作为单词，释义留空
			result = append(result, ParsedWord{
				Text:    line,
				Meaning: "",
			})
		}
	}
	return result
}

func stripJSONFence(value string) string {
	trimmed := strings.TrimSpace(value)
	matches := jsonFencePattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}

func decodeGradeResult(raw string) (*DictationGradeResult, error) {
	cleaned := stripJSONFence(raw)

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode grading result: %w", err)
	}

	if wrapped, ok := envelope["data"]; ok {
		var result DictationGradeResult
		if err := json.Unmarshal(wrapped, &result); err != nil {
			return nil, fmt.Errorf("failed to decode wrapped grading result: %w", err)
		}
		if err := validateGradeResult(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}

	if !hasGradeResultKeys(envelope) {
		return nil, fmt.Errorf("failed to decode grading result: missing expected grading fields")
	}

	var result DictationGradeResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to decode direct grading result: %w", err)
	}
	if err := validateGradeResult(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func hasGradeResultKeys(envelope map[string]json.RawMessage) bool {
	for _, key := range []string{"status", "score", "graded_items", "feedback", "mark_regions", "annotated_photo_url"} {
		if _, ok := envelope[key]; ok {
			return true
		}
	}
	return false
}

func validateGradeResult(result *DictationGradeResult) error {
	if result == nil {
		return fmt.Errorf("failed to decode grading result: empty grading result")
	}
	if strings.TrimSpace(result.Status) == "" {
		return fmt.Errorf("failed to decode grading result: missing status")
	}
	if len(result.GradedItems) == 0 {
		return fmt.Errorf("failed to decode grading result: missing graded_items")
	}
	if result.Score < 0 || result.Score > 100 {
		return fmt.Errorf("failed to decode grading result: score out of range")
	}
	result.Feedback = normalizeGradeFeedback(result.Feedback, result.GradedItems, result.Score)
	return nil
}

type DictationGradeResult struct {
	Status               string       `json:"status"`
	Score                int          `json:"score"`
	AnnotatedPhotoURL    string       `json:"annotated_photo_url"`
	AnnotatedPhotoWidth  int          `json:"annotated_photo_width"`
	AnnotatedPhotoHeight int          `json:"annotated_photo_height"`
	MarkRegions          []MarkRegion `json:"mark_regions"`
	GradedItems          []GradedWord `json:"graded_items"`
	Feedback             string       `json:"feedback"`
}

type MarkRegion struct {
	Index       int     `json:"index"`
	Expected    string  `json:"expected"`
	Actual      string  `json:"actual"`
	IsCorrect   bool    `json:"is_correct"`
	Left        float64 `json:"left"`
	Top         float64 `json:"top"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	MarkerLabel string  `json:"marker_label"`
}

type GradedWord struct {
	Index      int    `json:"index"`
	Expected   string `json:"expected"`
	Meaning    string `json:"meaning"`
	Actual     string `json:"actual"`
	IsCorrect  bool   `json:"is_correct"`
	Comment    string `json:"comment"`
	NeedsRetry bool   `json:"needs_retry"`
}

func normalizeGradeFeedback(feedback string, gradedItems []GradedWord, score int) string {
	trimmed := strings.TrimSpace(feedback)
	if trimmed != "" && !looksLikeEnglishFeedback(trimmed) {
		return trimmed
	}

	incorrectCount := countIncorrectWords(gradedItems)
	if incorrectCount == 0 {
		return fmt.Sprintf("这次默写全部正确，得分 %d 分，继续保持。", score)
	}

	if incorrectCount == 1 {
		return fmt.Sprintf("这次默写得分 %d 分，有 1 个词需要订正，继续认真改一改就更好了。", score)
	}

	return fmt.Sprintf("这次默写得分 %d 分，有 %d 个词需要订正，按错词清单逐个改一改吧。", score, incorrectCount)
}

func looksLikeEnglishFeedback(feedback string) bool {
	if feedback == "" {
		return true
	}
	asciiLetters := 0
	letterCount := 0
	for _, r := range feedback {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			asciiLetters++
			letterCount++
			continue
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			return false
		}
	}
	if letterCount == 0 {
		return false
	}
	return asciiLetters == letterCount
}

func countIncorrectWords(items []GradedWord) int {
	incorrect := 0
	for _, item := range items {
		if !item.IsCorrect || item.NeedsRetry {
			incorrect++
		}
	}
	return incorrect
}

func (s *Service) GradeDictation(ctx context.Context, wordList []ParsedWord, photoBase64 string, language string, mode string) (*DictationGradeResult, error) {
	if s.llmClient == nil {
		return nil, fmt.Errorf("llm client is required for grading")
	}

	var wordListStr strings.Builder
	for index, w := range wordList {
		wordListStr.WriteString(fmt.Sprintf("%d. %s | %s\n", index+1, w.Text, w.Meaning))
	}

	prompt := fmt.Sprintf(`
请对照图片中的手写内容和下面的有序答案列表完成听写批改。

语言：%s
模式：%s
有序答案：
%s

只返回 JSON：
{
  "status": "success",
  "score": 0,
  "annotated_photo_url": "",
  "annotated_photo_width": 0,
  "annotated_photo_height": 0,
  "mark_regions": [
    {
      "index": 1,
      "expected": "touch",
      "actual": "touch",
      "is_correct": true,
      "left": 0,
      "top": 0,
      "width": 0,
      "height": 0,
      "marker_label": "✅"
    }
  ],
  "graded_items": [
    {
      "index": 1,
      "expected": "touch",
      "meaning": "触碰",
      "actual": "touch",
      "is_correct": true,
      "comment": "",
      "needs_retry": false
    }
  ],
  "feedback": "继续保持。"
}

规则：
- 按顺序逐词比对。
- 尽量识别手写内容；无法确认时在 actual 中保留你能识别的部分，并将 needs_retry 设为 true。
- feedback 默认用简体中文，简短鼓励，不要输出英文反馈。
- comment 默认用简体中文，保持简短，让孩子容易看懂。
- 如果当前无法可靠定位图片中的单词位置，可返回空字符串 annotated_photo_url、0 宽高，以及空数组 mark_regions。
- 如果能提供位置，请使用 0 到 1 之间的归一化坐标填写 left/top/width/height。
- marker_label 仅使用“✅”或“❌”。
`, language, mode, wordListStr.String())

	// Note: We'll assume the LLM Client implementation can handle image data in the future.
	// For current Phase One, if client only supports text, we might need an OCR pre-processor.
	// But we'll define the contract here.
	log.Printf("[WordParse.Service] GradeDictation: preparing multimodal request language=%s mode=%s words=%d image_base64_bytes=%d", strings.TrimSpace(language), strings.TrimSpace(mode), len(wordList), len(photoBase64))
	log.Printf("[WordParse.Service] GradeDictation: prompt:\n%s", prompt)

	// Construct multimodal content
	contentParts := []llm.ContentPart{
		{
			Type: "image_url",
			ImageURL: &llm.ImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", photoBase64),
			},
		},
		{
			Type: "text",
			Text: prompt,
		},
	}

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey: "LLM_GRADER_MODEL_NAME",
		Temperature: 0.1,
		Messages: []llm.Message{
			{Role: "system", Content: "你是 StudyClaw 的听写批改老师助手。必须只返回合法 JSON，默认使用简体中文反馈和评语。"},
			{Role: "user", Content: contentParts},
		},
	})
	if err != nil {
		log.Printf("[WordParse.Service] GradeDictation: LLM request failed language=%s mode=%s err=%v", strings.TrimSpace(language), strings.TrimSpace(mode), err)
		return nil, err
	}
	log.Printf("[WordParse.Service] GradeDictation: Received raw response from LLM:\n%s", resultText)

	return decodeGradeResult(resultText)
}
