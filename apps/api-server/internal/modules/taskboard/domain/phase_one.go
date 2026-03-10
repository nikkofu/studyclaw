package domain

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
)

type TaskItem struct {
	TaskID      int      `json:"task_id"`
	Subject     string   `json:"subject"`
	GroupTitle  string   `json:"group_title"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Type        string   `json:"type,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	NeedsReview bool     `json:"needs_review"`
	Notes       []string `json:"notes,omitempty"`
	Completed   bool     `json:"completed"`
	Status      string   `json:"status"`
	PointsValue int      `json:"points_value"`
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
	SessionID      string    `json:"session_id"`
	WordListID     string    `json:"word_list_id"`
	FamilyID       uint      `json:"family_id"`
	ChildID        uint      `json:"child_id"`
	AssignedDate   string    `json:"assigned_date"`
	Status         string    `json:"status"`
	CurrentIndex   int       `json:"current_index"`
	TotalItems     int       `json:"total_items"`
	PlayedCount    int       `json:"played_count"`
	CompletedItems int       `json:"completed_items"`
	CurrentItem    *WordItem `json:"current_item,omitempty"`
	StartedAt      string    `json:"started_at"`
	UpdatedAt      string    `json:"updated_at"`
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
