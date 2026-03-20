package domain

import (
	"errors"
	"fmt"
)

const (
	AssignmentStatusDraft     = "draft"
	AssignmentStatusPublished = "published"

	PointsSourceTaskCompletion = "task_completion"
	PointsSourceParentReward   = "parent_reward"
	PointsSourceParentPenalty  = "parent_penalty"
	PointsSourceSchoolPraise   = "school_praise"
	PointsSourceSchoolCritic   = "school_criticism"

	PointsOriginSystem = "system"
	PointsOriginParent = "parent"

	DictationSessionActive    = "active"
	DictationSessionCompleted = "completed"

	DictationGradingIdle       = "idle"
	DictationGradingPending    = "pending"
	DictationGradingProcessing = "processing"
	DictationGradingCompleted  = "completed"
	DictationGradingFailed     = "failed"

	VoiceLearningSessionCompleted = "completed"

	PersistenceSessionStatusPreparing = "preparing"
	PersistenceSessionStatusActive    = "active"
	PersistenceSessionStatusPaused    = "paused"
	PersistenceSessionStatusResumed   = "resumed"
	PersistenceSessionStatusClosing   = "closing"
	PersistenceSessionStatusCompleted = "completed"
	PersistenceSessionStatusAborted   = "aborted"

	PersistenceEventStarted     = "started"
	PersistenceEventPaused      = "paused"
	PersistenceEventResumed     = "resumed"
	PersistenceEventInterrupted = "interrupted"
	PersistenceEventRecovered   = "recovered"
	PersistenceEventCompleted   = "completed"
	PersistenceEventAborted     = "aborted"

	PersistenceDurationSpeech  = "speech"
	PersistenceDurationSilence = "silence"
)

var ErrInvalidPersistenceTransition = errors.New("invalid persistence transition")

var allowedPersistenceTransitions = map[string]map[string]struct{}{
	PersistenceSessionStatusPreparing: {
		PersistenceSessionStatusActive:  {},
		PersistenceSessionStatusAborted: {},
	},
	PersistenceSessionStatusActive: {
		PersistenceSessionStatusPaused:  {},
		PersistenceSessionStatusClosing: {},
		PersistenceSessionStatusAborted: {},
	},
	PersistenceSessionStatusPaused: {
		PersistenceSessionStatusResumed: {},
		PersistenceSessionStatusAborted: {},
	},
	PersistenceSessionStatusResumed: {
		PersistenceSessionStatusActive:  {},
		PersistenceSessionStatusClosing: {},
		PersistenceSessionStatusAborted: {},
	},
	PersistenceSessionStatusClosing: {
		PersistenceSessionStatusCompleted: {},
		PersistenceSessionStatusAborted:   {},
	},
	PersistenceSessionStatusCompleted: {},
	PersistenceSessionStatusAborted:   {},
}

type PersistenceDurationSegment struct {
	Kind            string `json:"kind"`
	DurationSeconds int    `json:"duration_seconds"`
}

type PersistenceDayRecord struct {
	Completed bool `json:"completed"`
	Makeup    bool `json:"makeup,omitempty"`
}

type PersistenceStreak struct {
	DisplayStreak int `json:"display_streak"`
	CoreKPIStreak int `json:"core_kpi_streak"`
}

type PersistenceCompletionRate struct {
	Completed int     `json:"completed"`
	Total     int     `json:"total"`
	Rate      float64 `json:"rate"`
}

type PersistenceEffectiveDuration struct {
	TotalSeconds     int `json:"total_seconds"`
	EffectiveSeconds int `json:"effective_seconds"`
}

type PersistenceGuardrails struct {
	InvalidTriggerRate float64 `json:"invalid_trigger_rate"`
}

type PersistenceSummary struct {
	Streak            PersistenceStreak            `json:"streak"`
	CompletionRate    PersistenceCompletionRate    `json:"completion_rate"`
	EffectiveDuration PersistenceEffectiveDuration `json:"effective_duration"`
	Guardrails        PersistenceGuardrails        `json:"guardrails"`
}

func ValidatePersistenceTransition(fromStatus, toStatus string) error {
	next, ok := allowedPersistenceTransitions[fromStatus]
	if !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidPersistenceTransition, fromStatus, toStatus)
	}
	if _, ok := next[toStatus]; !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidPersistenceTransition, fromStatus, toStatus)
	}
	return nil
}

func CalculateEffectiveDurationSeconds(segments []PersistenceDurationSegment) int {
	total := 0
	for _, segment := range segments {
		if segment.DurationSeconds <= 0 {
			continue
		}
		if segment.Kind == PersistenceDurationSilence && segment.DurationSeconds >= 20 {
			continue
		}
		total += segment.DurationSeconds
	}
	return total
}

func ComputePersistenceStreak(days []PersistenceDayRecord) PersistenceStreak {
	streak := PersistenceStreak{}
	for _, day := range days {
		if !day.Completed {
			streak.DisplayStreak = 0
			streak.CoreKPIStreak = 0
			continue
		}
		streak.DisplayStreak++
		if !day.Makeup {
			streak.CoreKPIStreak++
		}
	}
	return streak
}

type TaskItem struct {
	TaskID                 int      `json:"task_id"`
	Subject                string   `json:"subject"`
	GroupTitle             string   `json:"group_title"`
	Title                  string   `json:"title"`
	Content                string   `json:"content"`
	Type                   string   `json:"type,omitempty"`
	Confidence             float64  `json:"confidence,omitempty"`
	NeedsReview            bool     `json:"needs_review"`
	Notes                  []string `json:"notes,omitempty"`
	Completed              bool     `json:"completed"`
	Status                 string   `json:"status"`
	PointsValue            int      `json:"points_value"`
	ReferenceTitle         string   `json:"reference_title,omitempty"`
	ReferenceAuthor        string   `json:"reference_author,omitempty"`
	ReferenceText          string   `json:"reference_text,omitempty"`
	ReferenceSource        string   `json:"reference_source,omitempty"`
	HideReferenceFromChild bool     `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string   `json:"analysis_mode,omitempty"`
}

type DailyAssignmentSummary struct {
	TotalTasks       int     `json:"total_tasks"`
	CompletedTasks   int     `json:"completed_tasks"`
	PendingTasks     int     `json:"pending_tasks"`
	NeedsReviewTasks int     `json:"needs_review_tasks"`
	SubjectCount     int     `json:"subject_count"`
	GroupCount       int     `json:"group_count"`
	CompletionRate   float64 `json:"completion_rate"`
	Status           string  `json:"status"`
}

type DailyAssignmentDraft struct {
	DraftID      string                 `json:"draft_id"`
	FamilyID     uint                   `json:"family_id"`
	ChildID      uint                   `json:"child_id"`
	AssignedDate string                 `json:"assigned_date"`
	SourceText   string                 `json:"source_text"`
	Status       string                 `json:"status"`
	ParserMode   string                 `json:"parser_mode"`
	Analysis     map[string]any         `json:"analysis"`
	TaskItems    []TaskItem             `json:"task_items"`
	Summary      DailyAssignmentSummary `json:"summary"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

type PublishedDailyAssignment struct {
	AssignmentID string                 `json:"assignment_id"`
	DraftID      string                 `json:"draft_id,omitempty"`
	FamilyID     uint                   `json:"family_id"`
	ChildID      uint                   `json:"child_id"`
	AssignedDate string                 `json:"assigned_date"`
	SourceText   string                 `json:"source_text,omitempty"`
	Status       string                 `json:"status"`
	TaskItems    []TaskItem             `json:"task_items"`
	Summary      DailyAssignmentSummary `json:"summary"`
	PublishedAt  string                 `json:"published_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

type PointsLedgerEntry struct {
	EntryID       string `json:"entry_id"`
	FamilyID      uint   `json:"family_id"`
	UserID        uint   `json:"user_id"`
	OccurredOn    string `json:"occurred_on"`
	Delta         int    `json:"delta"`
	SourceType    string `json:"source_type"`
	SourceOrigin  string `json:"source_origin"`
	SourceRefType string `json:"source_ref_type,omitempty"`
	SourceRefID   string `json:"source_ref_id,omitempty"`
	Note          string `json:"note,omitempty"`
	BalanceAfter  int    `json:"balance_after,omitempty"`
}

type PointsBalance struct {
	FamilyID     uint   `json:"family_id"`
	UserID       uint   `json:"user_id"`
	AsOfDate     string `json:"as_of_date"`
	Balance      int    `json:"balance"`
	TodayDelta   int    `json:"today_delta"`
	AutoPoints   int    `json:"auto_points"`
	ManualPoints int    `json:"manual_points"`
}

type WordItem struct {
	Index   int    `json:"index"`
	Text    string `json:"text"`
	Meaning string `json:"meaning,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type WordList struct {
	WordListID   string     `json:"word_list_id"`
	FamilyID     uint       `json:"family_id"`
	ChildID      uint       `json:"child_id"`
	AssignedDate string     `json:"assigned_date"`
	Title        string     `json:"title"`
	Language     string     `json:"language"`
	Items        []WordItem `json:"items"`
	TotalItems   int        `json:"total_items"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

type DictationSession struct {
	SessionID          string                   `json:"session_id"`
	WordListID         string                   `json:"word_list_id"`
	FamilyID           uint                     `json:"family_id"`
	ChildID            uint                     `json:"child_id"`
	AssignedDate       string                   `json:"assigned_date"`
	Mode               string                   `json:"mode"`
	Scene              string                   `json:"scene"`
	Status             string                   `json:"status"`
	CurrentIndex       int                      `json:"current_index"`
	TotalItems         int                      `json:"total_items"`
	PlayedCount        int                      `json:"played_count"`
	CompletedItems     int                      `json:"completed_items"`
	CurrentItem        *WordItem                `json:"current_item,omitempty"`
	TranscriptSegments []TranscriptSegment      `json:"transcript_segments"`
	MergedTranscript   string                   `json:"merged_transcript"`
	AnalysisSummary    DictationAnalysisSummary `json:"analysis_summary"`
	GradingStatus      string                   `json:"grading_status"`
	GradingError       string                   `json:"grading_error,omitempty"`
	GradingRequestedAt string                   `json:"grading_requested_at,omitempty"`
	GradingCompletedAt string                   `json:"grading_completed_at,omitempty"`
	GradingResult      *DictationGradingResult  `json:"grading_result,omitempty"`
	DebugContext       *DictationDebugContext   `json:"debug_context,omitempty"`
	StartedAt          string                   `json:"started_at"`
	EndedAt            string                   `json:"ended_at,omitempty"`
	UpdatedAt          string                   `json:"updated_at"`
}

type VoiceLearningSession struct {
	SessionID                string                    `json:"session_id"`
	FamilyID                 uint                      `json:"family_id"`
	ChildID                  uint                      `json:"child_id"`
	AssignedDate             string                    `json:"assigned_date"`
	Mode                     string                    `json:"mode"`
	Scene                    string                    `json:"scene"`
	Status                   string                    `json:"status"`
	TaskID                   int                       `json:"task_id,omitempty"`
	TaskTitle                string                    `json:"task_title,omitempty"`
	TaskType                 string                    `json:"task_type,omitempty"`
	ReferenceTitle           string                    `json:"reference_title,omitempty"`
	ReferenceAuthor          string                    `json:"reference_author,omitempty"`
	ReferenceSource          string                    `json:"reference_source,omitempty"`
	HideReferenceFromChild   bool                      `json:"hide_reference_from_child,omitempty"`
	TranscriptSegments       []TranscriptSegment       `json:"transcript_segments"`
	MergedTranscript         string                    `json:"merged_transcript"`
	Summary                  string                    `json:"summary,omitempty"`
	Encouragement            string                    `json:"encouragement,omitempty"`
	Analysis                 *VoiceLearningAnalysis    `json:"analysis,omitempty"`
	StartedAt                string                    `json:"started_at,omitempty"`
	EndedAt                  string                    `json:"ended_at,omitempty"`
	CreatedAt                string                    `json:"created_at,omitempty"`
	UpdatedAt                string                    `json:"updated_at,omitempty"`
}

type TranscriptSegment struct {
	SegmentID   string  `json:"segment_id"`
	Sequence    int     `json:"sequence"`
	StartedAt   string  `json:"started_at,omitempty"`
	EndedAt     string  `json:"ended_at,omitempty"`
	Transcript  string  `json:"transcript"`
	Source      string  `json:"source,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

type DictationAnalysisSummary struct {
	Status               string   `json:"status"`
	CompletionRatio      float64  `json:"completion_ratio"`
	NeedsRetry           bool     `json:"needs_retry"`
	Recommendation       string   `json:"recommendation"`
	RecommendationReason string   `json:"recommendation_reason"`
	Explainability       []string `json:"explainability"`
}

type VoiceLearningAnalysis struct {
	RecognizedTitle      string                     `json:"recognized_title,omitempty"`
	RecognizedAuthor     string                     `json:"recognized_author,omitempty"`
	ReferenceTitle       string                     `json:"reference_title,omitempty"`
	ReferenceAuthor      string                     `json:"reference_author,omitempty"`
	CompletionRatio      float64                    `json:"completion_ratio"`
	NeedsRetry           bool                       `json:"needs_retry"`
	Summary              string                     `json:"summary,omitempty"`
	Suggestion           string                     `json:"suggestion,omitempty"`
	Issues               []string                   `json:"issues,omitempty"`
	MatchedLines         []VoiceLearningMatchedLine `json:"matched_lines,omitempty"`
	ParserMode           string                     `json:"parser_mode,omitempty"`
	NormalizedTranscript string                     `json:"normalized_transcript,omitempty"`
}

type VoiceLearningMatchedLine struct {
	Index      int     `json:"index"`
	Expected   string  `json:"expected"`
	Observed   string  `json:"observed,omitempty"`
	MatchRatio float64 `json:"match_ratio"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes,omitempty"`
}

type DictationDebugContext struct {
	PhotoSHA1   string   `json:"photo_sha1,omitempty"`
	PhotoBytes  int      `json:"photo_bytes,omitempty"`
	Language    string   `json:"language,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	WorkerStage string   `json:"worker_stage,omitempty"`
	LogFile     string   `json:"log_file,omitempty"`
	LogKeywords []string `json:"log_keywords,omitempty"`
}

type DictationGradingResult struct {
	GradingID            string               `json:"grading_id"`
	PhotoURL             string               `json:"photo_url,omitempty"`
	AnnotatedPhotoURL    string               `json:"annotated_photo_url,omitempty"`
	AnnotatedPhotoWidth  int                  `json:"annotated_photo_width,omitempty"`
	AnnotatedPhotoHeight int                  `json:"annotated_photo_height,omitempty"`
	MarkRegions          []GradedWordRegion   `json:"mark_regions,omitempty"`
	Status               string               `json:"status"` // "passed", "needs_correction"
	Score                int                  `json:"score"`
	GradedItems          []GradedWordItem     `json:"graded_items"`
	AIFeedback           string               `json:"ai_feedback"`
	CreatedAt            string               `json:"created_at"`
}

type GradedWordRegion struct {
	Index       int     `json:"index"`
	Expected    string  `json:"expected,omitempty"`
	Actual      string  `json:"actual,omitempty"`
	IsCorrect   bool    `json:"is_correct"`
	Left        float64 `json:"left,omitempty"`
	Top         float64 `json:"top,omitempty"`
	Width       float64 `json:"width,omitempty"`
	Height      float64 `json:"height,omitempty"`
	MarkerLabel string  `json:"marker_label,omitempty"`
}

type GradedWordItem struct {
	Index      int    `json:"index"`
	Expected   string `json:"expected"`
	Meaning    string `json:"meaning,omitempty"`
	Actual     string `json:"actual"`
	IsCorrect  bool   `json:"is_correct"`
	Comment    string `json:"comment,omitempty"` // e.g., "Missing letter 'e'"
	NeedsRetry bool   `json:"needs_correction"`
}

type SubjectStats struct {
	Subject        string  `json:"subject"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	PendingTasks   int     `json:"pending_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

type StatsTotals struct {
	TotalTasks         int     `json:"total_tasks"`
	CompletedTasks     int     `json:"completed_tasks"`
	PendingTasks       int     `json:"pending_tasks"`
	CompletionRate     float64 `json:"completion_rate"`
	AutoPoints         int     `json:"auto_points"`
	ManualPoints       int     `json:"manual_points"`
	TotalPointsDelta   int     `json:"total_points_delta"`
	PointsBalance      int     `json:"points_balance"`
	WordItems          int     `json:"word_items"`
	CompletedWordItems int     `json:"completed_word_items"`
	DictationSessions  int     `json:"dictation_sessions"`
}

type CompletionSeriesPoint struct {
	Label          string  `json:"label"`
	Date           string  `json:"date"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

type PointsSeriesPoint struct {
	Label   string `json:"label"`
	Date    string `json:"date"`
	Delta   int    `json:"delta"`
	Balance int    `json:"balance"`
}

type WordSeriesPoint struct {
	Label          string `json:"label"`
	Date           string `json:"date"`
	TotalItems     int    `json:"total_items"`
	CompletedItems int    `json:"completed_items"`
	Sessions       int    `json:"sessions"`
}

type StatsResponse struct {
	Period           string                  `json:"period"`
	StartDate        string                  `json:"start_date"`
	EndDate          string                  `json:"end_date"`
	Totals           StatsTotals             `json:"totals"`
	SubjectBreakdown []SubjectStats          `json:"subject_breakdown"`
	CompletionSeries []CompletionSeriesPoint `json:"completion_series"`
	PointsSeries     []PointsSeriesPoint     `json:"points_series"`
	WordSeries       []WordSeriesPoint       `json:"word_series"`
	Encouragement    string                  `json:"encouragement"`
}
