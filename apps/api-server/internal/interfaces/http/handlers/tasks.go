package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
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
		c.JSON(status, gin.H{"error": err.Error()})
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	parsed, err := h.parser.Parse(context.Background(), strings.TrimSpace(req.RawText))
	if err != nil {
		log.Printf("Failed to parse parent input via Go parser: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse task text"})
		return
	}

	if parsed.Status != "success" || len(parsed.Data) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":          "Agent workflow could not extract valid tasks",
			"agent_response": parsed,
		})
		return
	}

	assignedDate, err := taskboardapp.ParseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdTasks := mapParsedTasksToCreateReqs(req.FamilyID, req.AssigneeID, assignedDate.Format("2006-01-02"), parsed.Data)
	if len(createdTasks) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Parser returned tasks without usable content"})
		return
	}

	if req.AutoCreate {
		if _, err := h.taskboard.CreateTasks(createdTasks); err != nil {
			log.Printf("Failed to persist parsed tasks: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store parsed tasks in markdown"})
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	if len(req.Tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tasks cannot be empty"})
		return
	}

	assignedDate, err := taskboardapp.ParseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid tasks were provided for confirmation"})
		return
	}

	if _, err := h.taskboard.CreateTasks(createdTasks); err != nil {
		log.Printf("Failed to persist confirmed tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store confirmed tasks in markdown"})
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
	familyIDStr := c.Query("family_id")
	userIDStr := c.Query("user_id")

	if familyIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family_id and user_id query params are required"})
		return
	}

	familyID, err := strconv.ParseUint(familyIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family_id must be a valid unsigned integer"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid unsigned integer"})
		return
	}

	targetDate, err := taskboardapp.ParseAssignedDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	board, err := h.taskboard.ListBoard(uint(familyID), uint(userID), targetDate)
	if err != nil {
		log.Printf("Error reading markdown tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks from markdown"})
		return
	}

	respondWithBoard(c, board, gin.H{})
}

func (h *TaskHandler) UpdateSingleTaskStatus(c *gin.Context) {
	var req UpdateTaskStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := taskboardapp.ParseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	board, updatedCount, err := h.taskboard.UpdateTaskStatusByID(req.FamilyID, req.AssigneeID, targetDate, req.TaskID, req.Completed)
	if err != nil {
		log.Printf("Failed to update single task status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		return
	}
	if updatedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	respondWithBoard(c, board, gin.H{
		"message":       "Single task status updated successfully",
		"updated_count": updatedCount,
	})
}

func (h *TaskHandler) UpdateTaskGroupStatus(c *gin.Context) {
	var req UpdateTaskGroupStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := taskboardapp.ParseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	board, updatedCount, err := h.taskboard.UpdateTaskStatusByGroup(req.FamilyID, req.AssigneeID, targetDate, req.Subject, req.GroupTitle, req.Completed)
	if err != nil {
		log.Printf("Failed to update task group status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task group status"})
		return
	}
	if updatedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No tasks found for the provided subject"})
		return
	}

	respondWithBoard(c, board, gin.H{
		"message":       "Task group status updated successfully",
		"updated_count": updatedCount,
		"subject":       strings.TrimSpace(req.Subject),
		"group_title":   strings.TrimSpace(req.GroupTitle),
	})
}

func (h *TaskHandler) UpdateAllTasksStatus(c *gin.Context) {
	var req UpdateAllTasksStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := taskboardapp.ParseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	board, updatedCount, err := h.taskboard.UpdateAllTaskStatuses(req.FamilyID, req.AssigneeID, targetDate, req.Completed)
	if err != nil {
		log.Printf("Failed to update all task statuses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update all task statuses"})
		return
	}

	respondWithBoard(c, board, gin.H{
		"message":       "All task statuses updated successfully",
		"updated_count": updatedCount,
	})
}
