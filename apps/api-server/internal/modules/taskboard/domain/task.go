package domain

type Task struct {
	TaskID                 int     `json:"task_id"`
	RawLine                string  `json:"raw_line"`
	Completed              bool    `json:"completed"`
	Status                 string  `json:"status"`
	Subject                string  `json:"subject"`
	GroupTitle             string  `json:"group_title"`
	Content                string  `json:"content"`
	TaskType               string  `json:"task_type,omitempty"`
	ReferenceTitle         string  `json:"reference_title,omitempty"`
	ReferenceAuthor        string  `json:"reference_author,omitempty"`
	ReferenceText          string  `json:"reference_text,omitempty"`
	ReferenceSource        string  `json:"reference_source,omitempty"`
	HideReferenceFromChild bool    `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string  `json:"analysis_mode,omitempty"`
	IsCurrentSessionItem   bool    `json:"is_current_session_item,omitempty"`
	PriorityWeight         int     `json:"priority_weight,omitempty"`
	InterruptionRiskScore  float64 `json:"interruption_risk_score,omitempty"`
	AssignedSequence       int     `json:"assigned_sequence,omitempty"`
}

type GroupSummary struct {
	Subject   string `json:"subject"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Pending   int    `json:"pending"`
	Status    string `json:"status"`
}

type HomeworkGroupSummary struct {
	Subject    string `json:"subject"`
	GroupTitle string `json:"group_title"`
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Pending    int    `json:"pending"`
	Status     string `json:"status"`
}

type Summary struct {
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Pending   int    `json:"pending"`
	Status    string `json:"status"`
}

type Board struct {
	Date                 string                 `json:"date"`
	Tasks                []Task                 `json:"tasks"`
	Groups               []GroupSummary         `json:"groups"`
	HomeworkGroups       []HomeworkGroupSummary `json:"homework_groups"`
	Summary              Summary                `json:"summary"`
	LaunchRecommendation *LaunchRecommendation  `json:"launch_recommendation,omitempty"`
}

type LaunchRecommendation struct {
	ReasonCode     string `json:"reason_code"`
	GroupID        string `json:"group_id"`
	ItemID         *int   `json:"item_id"`
	WhyRecommended string `json:"why_recommended,omitempty"`
}

type CreateTaskInput struct {
	FamilyID               uint   `json:"family_id"`
	AssigneeID             uint   `json:"assignee_id"`
	Subject                string `json:"subject"`
	GroupTitle             string `json:"group_title,omitempty"`
	Content                string `json:"content"`
	AssignedDate           string `json:"assigned_date,omitempty"`
	TaskType               string `json:"task_type,omitempty"`
	ReferenceTitle         string `json:"reference_title,omitempty"`
	ReferenceAuthor        string `json:"reference_author,omitempty"`
	ReferenceText          string `json:"reference_text,omitempty"`
	ReferenceSource        string `json:"reference_source,omitempty"`
	HideReferenceFromChild bool   `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string `json:"analysis_mode,omitempty"`
}
