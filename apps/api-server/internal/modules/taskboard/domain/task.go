package domain

type Task struct {
	TaskID                 int    `json:"task_id"`
	RawLine                string `json:"raw_line"`
	Completed              bool   `json:"completed"`
	Status                 string `json:"status"`
	Subject                string `json:"subject"`
	GroupTitle             string `json:"group_title"`
	Content                string `json:"content"`
	TaskType               string `json:"task_type,omitempty"`
	ReferenceTitle         string `json:"reference_title,omitempty"`
	ReferenceAuthor        string `json:"reference_author,omitempty"`
	ReferenceText          string `json:"reference_text,omitempty"`
	HideReferenceFromChild bool   `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string `json:"analysis_mode,omitempty"`
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
	Date           string                 `json:"date"`
	Tasks          []Task                 `json:"tasks"`
	Groups         []GroupSummary         `json:"groups"`
	HomeworkGroups []HomeworkGroupSummary `json:"homework_groups"`
	Summary        Summary                `json:"summary"`
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
	HideReferenceFromChild bool   `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string `json:"analysis_mode,omitempty"`
}
