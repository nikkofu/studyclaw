package voicecommand

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

var jsonFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

const (
	ActionNone                = "none"
	ActionDictationNext       = "dictation_next"
	ActionDictationPrevious   = "dictation_previous"
	ActionDictationReplay     = "dictation_replay"
	ActionTaskCompleteItem    = "task_complete_item"
	ActionTaskCompleteGroup   = "task_complete_group"
	ActionTaskCompleteSubject = "task_complete_subject"
	ActionTaskCompleteAll     = "task_complete_all"

	SurfaceDictation = "dictation"
	SurfaceTaskBoard = "task_board"

	parserModeRuleFallback = "rule_fallback"
	parserModeLLMHybrid    = "llm_hybrid"
)

var allowedActions = []string{
	ActionNone,
	ActionDictationNext,
	ActionDictationPrevious,
	ActionDictationReplay,
	ActionTaskCompleteItem,
	ActionTaskCompleteGroup,
	ActionTaskCompleteSubject,
	ActionTaskCompleteAll,
}

type ResolveInput struct {
	Transcript string         `json:"transcript"`
	Context    CommandContext `json:"context"`
}

type CommandContext struct {
	Surface   string            `json:"surface"`
	Dictation *DictationContext `json:"dictation,omitempty"`
	TaskBoard *TaskBoardContext `json:"task_board,omitempty"`
	Locale    string            `json:"locale,omitempty"`
	Examples  []string          `json:"examples,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type DictationContext struct {
	SessionID    string `json:"session_id,omitempty"`
	CurrentWord  string `json:"current_word,omitempty"`
	CurrentIndex int    `json:"current_index,omitempty"`
	TotalItems   int    `json:"total_items,omitempty"`
	CanNext      bool   `json:"can_next,omitempty"`
	CanPrevious  bool   `json:"can_previous,omitempty"`
	IsCompleted  bool   `json:"is_completed,omitempty"`
	Language     string `json:"language,omitempty"`
	PlaybackMode string `json:"playback_mode,omitempty"`
}

type TaskBoardContext struct {
	FocusedSubject string               `json:"focused_subject,omitempty"`
	Summary        TaskBoardSummary     `json:"summary"`
	Subjects       []TaskSubjectContext `json:"subjects,omitempty"`
	Groups         []TaskGroupContext   `json:"groups,omitempty"`
	Tasks          []TaskItemContext    `json:"tasks,omitempty"`
}

type TaskBoardSummary struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
}

type TaskSubjectContext struct {
	Subject   string `json:"subject"`
	Status    string `json:"status,omitempty"`
	Completed int    `json:"completed,omitempty"`
	Pending   int    `json:"pending,omitempty"`
	Total     int    `json:"total,omitempty"`
}

type TaskGroupContext struct {
	Subject    string `json:"subject"`
	GroupTitle string `json:"group_title"`
	Status     string `json:"status,omitempty"`
	Completed  int    `json:"completed,omitempty"`
	Pending    int    `json:"pending,omitempty"`
	Total      int    `json:"total,omitempty"`
}

type TaskItemContext struct {
	TaskID     int    `json:"task_id"`
	Subject    string `json:"subject"`
	GroupTitle string `json:"group_title,omitempty"`
	Content    string `json:"content"`
	Completed  bool   `json:"completed,omitempty"`
	Status     string `json:"status,omitempty"`
}

type Resolution struct {
	Status               string           `json:"status"`
	Action               string           `json:"action"`
	Reason               string           `json:"reason"`
	ParserMode           string           `json:"parser_mode"`
	Confidence           float64          `json:"confidence"`
	NormalizedTranscript string           `json:"normalized_transcript"`
	Surface              string           `json:"surface"`
	Target               ResolutionTarget `json:"target"`
}

type ResolutionTarget struct {
	SessionID   string `json:"session_id,omitempty"`
	TaskID      int    `json:"task_id,omitempty"`
	Subject     string `json:"subject,omitempty"`
	GroupTitle  string `json:"group_title,omitempty"`
	TaskContent string `json:"task_content,omitempty"`
}

type llmResolution struct {
	Status     string           `json:"status"`
	Action     string           `json:"action"`
	Reason     string           `json:"reason"`
	Confidence float64          `json:"confidence"`
	Target     ResolutionTarget `json:"target"`
}

type Service struct {
	llmClient llm.Client
}

func NewService(llmClient llm.Client) *Service {
	return &Service{llmClient: llmClient}
}

func (s *Service) Resolve(ctx context.Context, input ResolveInput) (Resolution, error) {
	input.Transcript = strings.TrimSpace(input.Transcript)
	input.Context.Surface = strings.TrimSpace(input.Context.Surface)

	fallback := resolveRuleBased(input)
	if s.llmClient == nil || input.Transcript == "" {
		return fallback, nil
	}

	contextJSON, err := json.MarshalIndent(input.Context, "", "  ")
	if err != nil {
		return fallback, nil
	}

	prompt := fmt.Sprintf(`
你是 StudyClaw Pad 的语音交互推理代理。你的任务是把孩子说出的自然语言，映射成一个可以触发现有 UI 按钮行为的结构化 action。

允许动作仅限：
- %s

上下文 JSON：
%s

孩子说的话：
%s

推理规则：
1. 只允许从允许动作里选择一个 action。
2. 如果是听写场景，"好了"、"继续"、"next" 通常表示进入下一词；"重播" 表示重复当前词；"上一个" 表示回到上一词。
3. 如果是任务板场景，优先匹配最具体的目标：task > group > subject > all。
4. 如果语句不足以唯一定位或不应触发动作，返回 action="none"。
5. reason 保持简短，说明为什么这样判断。
6. confidence 取 0 到 1。
7. 只返回 JSON，不要输出解释文字。

输出格式：
{
  "status": "success",
  "action": "task_complete_group",
  "reason": "提到了分组名称并表达完成",
  "confidence": 0.93,
  "target": {
    "subject": "数学",
    "group_title": "一课一练"
  }
}
`, strings.Join(allowedActions, ", "), string(contextJSON), input.Transcript)

	resultText, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		ModelEnvKey:  "LLM_PARSER_MODEL_NAME",
		Temperature:  0.1,
		DefaultModel: "",
		Messages: []llm.Message{
			{Role: "system", Content: "You are a voice command resolver for StudyClaw. Return valid JSON only."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return fallback, nil
	}

	var parsed llmResolution
	if err := json.Unmarshal([]byte(stripJSONFence(resultText)), &parsed); err != nil {
		return fallback, nil
	}

	normalized := normalizeResolution(parsed, input, fallback.NormalizedTranscript)
	if !slices.Contains(allowedActions, normalized.Action) {
		return fallback, nil
	}
	if normalized.Action == ActionNone && strings.TrimSpace(normalized.Reason) == "" {
		normalized.Reason = fallback.Reason
	}
	normalized.ParserMode = parserModeLLMHybrid
	if normalized.Confidence <= 0 {
		normalized.Confidence = fallback.Confidence
	}
	return normalized, nil
}

func normalizeResolution(parsed llmResolution, input ResolveInput, normalizedTranscript string) Resolution {
	action := strings.TrimSpace(parsed.Action)
	if !slices.Contains(allowedActions, action) {
		action = ActionNone
	}

	return Resolution{
		Status:               "success",
		Action:               action,
		Reason:               strings.TrimSpace(parsed.Reason),
		ParserMode:           parserModeLLMHybrid,
		Confidence:           parsed.Confidence,
		NormalizedTranscript: normalizedTranscript,
		Surface:              normalizeSurface(input.Context.Surface),
		Target: ResolutionTarget{
			SessionID:   strings.TrimSpace(parsed.Target.SessionID),
			TaskID:      parsed.Target.TaskID,
			Subject:     strings.TrimSpace(parsed.Target.Subject),
			GroupTitle:  strings.TrimSpace(parsed.Target.GroupTitle),
			TaskContent: strings.TrimSpace(parsed.Target.TaskContent),
		},
	}
}

func resolveRuleBased(input ResolveInput) Resolution {
	surface := normalizeSurface(input.Context.Surface)
	normalizedTranscript := normalizeTranscript(input.Transcript)

	if normalizedTranscript == "" {
		return Resolution{
			Status:               "success",
			Action:               ActionNone,
			Reason:               "没有识别到有效语音内容",
			ParserMode:           parserModeRuleFallback,
			Confidence:           0.2,
			NormalizedTranscript: normalizedTranscript,
			Surface:              surface,
		}
	}

	if surface == SurfaceDictation {
		if action, reason := resolveDictationAction(normalizedTranscript, input.Context.Dictation); action != ActionNone {
			return Resolution{
				Status:               "success",
				Action:               action,
				Reason:               reason,
				ParserMode:           parserModeRuleFallback,
				Confidence:           0.86,
				NormalizedTranscript: normalizedTranscript,
				Surface:              surface,
				Target: ResolutionTarget{
					SessionID: strings.TrimSpace(input.Context.Dictation.SessionID),
				},
			}
		}
	}

	if surface == SurfaceTaskBoard {
		if resolution, ok := resolveTaskBoardAction(normalizedTranscript, input.Context.TaskBoard); ok {
			resolution.ParserMode = parserModeRuleFallback
			resolution.NormalizedTranscript = normalizedTranscript
			resolution.Surface = surface
			return resolution
		}
	}

	return Resolution{
		Status:               "success",
		Action:               ActionNone,
		Reason:               "当前语句不足以确定要触发的交互",
		ParserMode:           parserModeRuleFallback,
		Confidence:           0.35,
		NormalizedTranscript: normalizedTranscript,
		Surface:              surface,
	}
}

func resolveDictationAction(normalizedTranscript string, dictation *DictationContext) (string, string) {
	if containsAny(normalizedTranscript, "重播", "再来一遍", "再读一遍", "重复", "repeat", "again") {
		return ActionDictationReplay, "识别到重播类指令"
	}
	if containsAny(normalizedTranscript, "上一个", "上一题", "前一个", "back", "previous") && dictation != nil && dictation.CanPrevious {
		return ActionDictationPrevious, "识别到上一词类指令"
	}
	if containsAny(normalizedTranscript, "下一个", "下个", "继续", "next", "好了", "好啦", "ok", "okay", "完成了") && (dictation == nil || dictation.CanNext || !dictation.IsCompleted) {
		return ActionDictationNext, "识别到继续下一词类指令"
	}
	return ActionNone, ""
}

func resolveTaskBoardAction(normalizedTranscript string, taskBoard *TaskBoardContext) (Resolution, bool) {
	if taskBoard == nil {
		return Resolution{}, false
	}

	if isCompletionUtterance(normalizedTranscript) && containsAny(normalizedTranscript, "全部", "都", "all", "所有") {
		return Resolution{
			Status:     "success",
			Action:     ActionTaskCompleteAll,
			Reason:     "表达了全部任务完成",
			Confidence: 0.82,
		}, true
	}

	if !isCompletionUtterance(normalizedTranscript) {
		return Resolution{}, false
	}

	if target, ok := matchTaskItem(normalizedTranscript, taskBoard.Tasks); ok {
		return Resolution{
			Status:     "success",
			Action:     ActionTaskCompleteItem,
			Reason:     "提到了具体任务内容并表达完成",
			Confidence: 0.84,
			Target: ResolutionTarget{
				TaskID:      target.TaskID,
				Subject:     target.Subject,
				GroupTitle:  target.GroupTitle,
				TaskContent: target.Content,
			},
		}, true
	}

	if target, ok := matchTaskGroup(normalizedTranscript, taskBoard.Groups); ok {
		return Resolution{
			Status:     "success",
			Action:     ActionTaskCompleteGroup,
			Reason:     "提到了任务分组并表达完成",
			Confidence: 0.83,
			Target: ResolutionTarget{
				Subject:    target.Subject,
				GroupTitle: target.GroupTitle,
			},
		}, true
	}

	if target, ok := matchTaskSubject(normalizedTranscript, taskBoard.Subjects, taskBoard.FocusedSubject); ok {
		return Resolution{
			Status:     "success",
			Action:     ActionTaskCompleteSubject,
			Reason:     "提到了学科并表达完成",
			Confidence: 0.79,
			Target: ResolutionTarget{
				Subject: target.Subject,
			},
		}, true
	}

	return Resolution{}, false
}

func matchTaskItem(normalizedTranscript string, tasks []TaskItemContext) (TaskItemContext, bool) {
	best := TaskItemContext{}
	bestScore := 0
	for _, task := range tasks {
		if task.Completed || strings.EqualFold(strings.TrimSpace(task.Status), "completed") {
			continue
		}
		candidate := bestMatchScore(normalizedTranscript, task.Content)
		if candidate > bestScore {
			best = task
			bestScore = candidate
		}
	}
	return best, bestScore > 0
}

func matchTaskGroup(normalizedTranscript string, groups []TaskGroupContext) (TaskGroupContext, bool) {
	best := TaskGroupContext{}
	bestScore := 0
	for _, group := range groups {
		if strings.EqualFold(strings.TrimSpace(group.Status), "completed") || group.Pending <= 0 {
			continue
		}
		candidate := bestMatchScore(normalizedTranscript, group.GroupTitle, group.Subject)
		if candidate > bestScore {
			best = group
			bestScore = candidate
		}
	}
	return best, bestScore > 0
}

func matchTaskSubject(normalizedTranscript string, subjects []TaskSubjectContext, focusedSubject string) (TaskSubjectContext, bool) {
	best := TaskSubjectContext{}
	bestScore := 0
	for _, subject := range subjects {
		if strings.EqualFold(strings.TrimSpace(subject.Status), "completed") || subject.Pending <= 0 {
			continue
		}
		candidate := bestMatchScore(normalizedTranscript, subject.Subject)
		if candidate > bestScore {
			best = subject
			bestScore = candidate
		}
	}
	if bestScore > 0 {
		return best, true
	}

	if focused := strings.TrimSpace(focusedSubject); focused != "" && containsAny(normalizedTranscript, "好了", "完成", "done", "finished") {
		return TaskSubjectContext{Subject: focused}, true
	}
	return TaskSubjectContext{}, false
}

func bestMatchScore(normalizedTranscript string, candidates ...string) int {
	best := 0
	for _, candidate := range candidates {
		normalizedCandidate := normalizeTranscript(candidate)
		if normalizedCandidate == "" {
			continue
		}
		if strings.Contains(normalizedTranscript, normalizedCandidate) {
			if len(normalizedCandidate) > best {
				best = len(normalizedCandidate)
			}
		}
	}
	return best
}

func isCompletionUtterance(normalizedTranscript string) bool {
	return containsAny(
		normalizedTranscript,
		"做好了",
		"做完了",
		"完成了",
		"订正好了",
		"弄好了",
		"写好了",
		"都好了",
		"搞定了",
		"done",
		"finished",
		"complete",
	)
}

func containsAny(normalizedTranscript string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(normalizedTranscript, normalizeTranscript(phrase)) {
			return true
		}
	}
	return false
}

func normalizeTranscript(raw string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(raw)) {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			builder.WriteRune(r)
		case unicode.In(r, unicode.Han):
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func normalizeSurface(surface string) string {
	switch strings.TrimSpace(surface) {
	case SurfaceDictation:
		return SurfaceDictation
	case SurfaceTaskBoard:
		return SurfaceTaskBoard
	default:
		return SurfaceTaskBoard
	}
}

func stripJSONFence(value string) string {
	trimmed := strings.TrimSpace(value)
	matches := jsonFencePattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}
