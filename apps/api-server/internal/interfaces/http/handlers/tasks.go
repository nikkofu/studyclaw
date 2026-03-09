package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	taskparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/taskparse"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type TaskHandler struct {
	taskboard *taskboardapp.Service
	parser    *taskparse.Service
}

type CreateTaskReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	AssigneeID   uint   `json:"assignee_id" binding:"required"`
	Subject      string `json:"subject"`
	GroupTitle   string `json:"group_title,omitempty"`
	Content      string `json:"content" binding:"required"`
	AssignedDate string `json:"assigned_date,omitempty"`
}

type ParseTaskReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	AssigneeID   uint   `json:"assignee_id" binding:"required"`
	RawText      string `json:"raw_text" binding:"required"`
	AutoCreate   bool   `json:"auto_create"`
	AssignedDate string `json:"assigned_date,omitempty"`
}

type ConfirmTaskItem struct {
	Subject     string   `json:"subject"`
	GroupTitle  string   `json:"group_title,omitempty"`
	Title       string   `json:"title"`
	Confidence  float64  `json:"confidence,omitempty"`
	NeedsReview bool     `json:"needs_review,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

type ConfirmTasksReq struct {
	FamilyID     uint              `json:"family_id" binding:"required"`
	AssigneeID   uint              `json:"assignee_id" binding:"required"`
	Tasks        []ConfirmTaskItem `json:"tasks" binding:"required"`
	AssignedDate string            `json:"assigned_date,omitempty"`
}

type UpdateTaskStatusReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	AssigneeID   uint   `json:"assignee_id" binding:"required"`
	TaskID       int    `json:"task_id" binding:"required"`
	Completed    bool   `json:"completed"`
	AssignedDate string `json:"assigned_date,omitempty"`
}

type UpdateTaskGroupStatusReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	AssigneeID   uint   `json:"assignee_id" binding:"required"`
	Subject      string `json:"subject" binding:"required"`
	GroupTitle   string `json:"group_title,omitempty"`
	Completed    bool   `json:"completed"`
	AssignedDate string `json:"assigned_date,omitempty"`
}

type UpdateAllTasksStatusReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	AssigneeID   uint   `json:"assignee_id" binding:"required"`
	Completed    bool   `json:"completed"`
	AssignedDate string `json:"assigned_date,omitempty"`
}

func NewTaskHandler(taskboard *taskboardapp.Service, parser *taskparse.Service) *TaskHandler {
	return &TaskHandler{taskboard: taskboard, parser: parser}
}

func mapParsedTasksToCreateReqs(familyID, assigneeID uint, assignedDate string, parsedTasks []taskparse.ParsedTask) []taskboarddomain.CreateTaskInput {
	createdTasks := make([]taskboarddomain.CreateTaskInput, 0, len(parsedTasks))

	for _, task := range parsedTasks {
		subject, groupTitle, content := taskboardapp.NormalizeTaskFields(task.Subject, task.GroupTitle, task.Title)
		if content == "" {
			continue
		}

		createdTasks = append(createdTasks, taskboarddomain.CreateTaskInput{
			FamilyID:     familyID,
			AssigneeID:   assigneeID,
			Subject:      subject,
			GroupTitle:   groupTitle,
			Content:      content,
			AssignedDate: assignedDate,
		})
	}

	return createdTasks
}

func respondWithBoard(c *gin.Context, board taskboarddomain.Board, extra gin.H) {
	payload := gin.H{
		"date":            board.Date,
		"tasks":           board.Tasks,
		"groups":          board.Groups,
		"homework_groups": board.HomeworkGroups,
		"summary":         board.Summary,
	}

	for key, value := range extra {
		payload[key] = value
	}

	c.JSON(http.StatusOK, payload)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	assignedDate, err := h.taskboard.CreateTask(taskboarddomain.CreateTaskInput{
		FamilyID:     req.FamilyID,
		AssigneeID:   req.AssigneeID,
		Subject:      req.Subject,
		GroupTitle:   req.GroupTitle,
		Content:      req.Content,
		AssignedDate: req.AssignedDate,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot be empty") || strings.Contains(err.Error(), "YYYY-MM-DD") {
			status = http.StatusBadRequest
		}
		errorCode := "internal_error"
		details := any(nil)
		if status == http.StatusBadRequest {
			errorCode = "invalid_request"
			if strings.Contains(err.Error(), "YYYY-MM-DD") {
				errorCode = "invalid_date"
				details = gin.H{"field": "assigned_date"}
			}
		}
		respondError(c, status, errorCode, err.Error(), details)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task written to daily markdown successfully",
		"date":    assignedDate.Format("2006-01-02"),
		"task":    req,
	})
}

func (h *TaskHandler) ParseAndCreateTasks(c *gin.Context) {
	var req ParseTaskReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	parsed, err := h.parser.Parse(context.Background(), strings.TrimSpace(req.RawText))
	if err != nil {
		log.Printf("Failed to parse parent input via Go parser: %v", err)
		respondError(c, http.StatusBadGateway, "parser_unavailable", "Failed to parse task text", nil)
		return
	}

	if parsed.Status != "success" || len(parsed.Data) == 0 {
		respondError(c, http.StatusUnprocessableEntity, "tasks_not_extractable", "Agent workflow could not extract valid tasks", gin.H{
			"agent_response": parsed,
		})
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	createdTasks := mapParsedTasksToCreateReqs(req.FamilyID, req.AssigneeID, assignedDate.Format("2006-01-02"), parsed.Data)
	if len(createdTasks) == 0 {
		respondError(c, http.StatusUnprocessableEntity, "tasks_not_extractable", "Parser returned tasks without usable content", nil)
		return
	}

	if req.AutoCreate {
		if _, err := h.taskboard.CreateTasks(createdTasks); err != nil {
			log.Printf("Failed to persist parsed tasks: %v", err)
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to store parsed tasks in markdown", nil)
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Tasks parsed successfully",
		"parsed_count": len(createdTasks),
		"parser_mode":  parsed.ParserMode,
		"analysis":     parsed.Analysis,
		"auto_created": req.AutoCreate,
		"date":         assignedDate.Format("2006-01-02"),
		"tasks":        parsed.Data,
	})
}

func (h *TaskHandler) ConfirmTasks(c *gin.Context) {
	var req ConfirmTasksReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	if len(req.Tasks) == 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "tasks cannot be empty", gin.H{
			"field": "tasks",
		})
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	createdTasks := make([]taskboarddomain.CreateTaskInput, 0, len(req.Tasks))
	for _, task := range req.Tasks {
		subject, groupTitle, content := taskboardapp.NormalizeTaskFields(task.Subject, task.GroupTitle, task.Title)
		if content == "" {
			continue
		}

		createdTasks = append(createdTasks, taskboarddomain.CreateTaskInput{
			FamilyID:     req.FamilyID,
			AssigneeID:   req.AssigneeID,
			Subject:      subject,
			GroupTitle:   groupTitle,
			Content:      content,
			AssignedDate: assignedDate.Format("2006-01-02"),
		})
	}

	if len(createdTasks) == 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "No valid tasks were provided for confirmation", gin.H{
			"field": "tasks",
		})
		return
	}

	if _, err := h.taskboard.CreateTasks(createdTasks); err != nil {
		log.Printf("Failed to persist confirmed tasks: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to store confirmed tasks in markdown", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Confirmed tasks written to daily markdown successfully",
		"created_count": len(createdTasks),
		"date":          assignedDate.Format("2006-01-02"),
		"tasks":         createdTasks,
	})
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	queryValues, ok := requireQueryParams(c, "family_id", "user_id")
	if !ok {
		return
	}

	familyID, ok := parseUintQueryParam(c, "family_id", queryValues["family_id"])
	if !ok {
		return
	}

	userID, ok := parseUintQueryParam(c, "user_id", queryValues["user_id"])
	if !ok {
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "date", c.Query("date"))
	if !ok {
		return
	}

	board, err := h.taskboard.ListBoard(familyID, userID, targetDate)
	if err != nil {
		log.Printf("Error reading markdown tasks: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to fetch tasks from markdown", nil)
		return
	}

	respondWithBoard(c, board, gin.H{})
}

func (h *TaskHandler) UpdateSingleTaskStatus(c *gin.Context) {
	var req UpdateTaskStatusReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	result, err := h.taskboard.UpdateTaskStatusByID(req.FamilyID, req.AssigneeID, targetDate, req.TaskID, req.Completed)
	if err != nil {
		log.Printf("Failed to update single task status: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to update task status", nil)
		return
	}
	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "task_not_found", "Task not found", gin.H{
			"task_id": req.TaskID,
		})
		return
	}
	if result.UpdatedCount == 0 {
		respondError(c, http.StatusConflict, "status_unchanged", "Task status is already "+statusLabel(req.Completed), gin.H{
			"task_id": req.TaskID,
			"status":  statusLabel(req.Completed),
		})
		return
	}

	respondWithBoard(c, result.Board, gin.H{
		"message":       "Single task status updated successfully",
		"updated_count": result.UpdatedCount,
	})
}

func (h *TaskHandler) UpdateTaskGroupStatus(c *gin.Context) {
	var req UpdateTaskGroupStatusReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	if strings.TrimSpace(req.Subject) == "" {
		respondError(c, http.StatusBadRequest, "missing_required_fields", "Required fields are missing", gin.H{
			"fields": []string{"subject"},
		})
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	result, err := h.taskboard.UpdateTaskStatusByGroup(req.FamilyID, req.AssigneeID, targetDate, req.Subject, req.GroupTitle, req.Completed)
	if err != nil {
		log.Printf("Failed to update task group status: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to update task group status", nil)
		return
	}
	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "task_group_not_found", "No tasks found for the provided subject", gin.H{
			"subject":     strings.TrimSpace(req.Subject),
			"group_title": strings.TrimSpace(req.GroupTitle),
		})
		return
	}
	if result.UpdatedCount == 0 {
		respondError(c, http.StatusConflict, "status_unchanged", "Matched tasks are already "+statusLabel(req.Completed), gin.H{
			"subject":     strings.TrimSpace(req.Subject),
			"group_title": strings.TrimSpace(req.GroupTitle),
			"status":      statusLabel(req.Completed),
		})
		return
	}

	respondWithBoard(c, result.Board, gin.H{
		"message":       "Task group status updated successfully",
		"updated_count": result.UpdatedCount,
		"subject":       strings.TrimSpace(req.Subject),
		"group_title":   strings.TrimSpace(req.GroupTitle),
	})
}

func (h *TaskHandler) UpdateAllTasksStatus(c *gin.Context) {
	var req UpdateAllTasksStatusReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	result, err := h.taskboard.UpdateAllTaskStatuses(req.FamilyID, req.AssigneeID, targetDate, req.Completed)
	if err != nil {
		log.Printf("Failed to update all task statuses: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to update all task statuses", nil)
		return
	}
	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "task_not_found", "No tasks found to update", nil)
		return
	}
	if result.UpdatedCount == 0 {
		respondError(c, http.StatusConflict, "status_unchanged", "All tasks are already "+statusLabel(req.Completed), gin.H{
			"status": statusLabel(req.Completed),
		})
		return
	}

	respondWithBoard(c, result.Board, gin.H{
		"message":       "All task statuses updated successfully",
		"updated_count": result.UpdatedCount,
	})
}

func statusLabel(completed bool) string {
	if completed {
		return "completed"
	}
	return "pending"
}
