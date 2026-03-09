package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/services"
)

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

type TaskGroupSummary struct {
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

func parseUintParam(value, field string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid unsigned integer", field)
	}

	return uint(parsed), nil
}

func parseAssignedDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now(), nil
	}

	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("assigned_date must be in YYYY-MM-DD format")
	}

	return parsed, nil
}

func normalizeTaskFields(subject, groupTitle, content string) (string, string, string) {
	normalizedSubject := strings.TrimSpace(subject)
	if normalizedSubject == "" {
		normalizedSubject = "未分类"
	}

	normalizedContent := strings.TrimSpace(content)
	normalizedGroupTitle := strings.TrimSpace(groupTitle)
	if normalizedGroupTitle == "" {
		normalizedGroupTitle = normalizedContent
	}

	return normalizedSubject, normalizedGroupTitle, normalizedContent
}

func mapParsedTasksToCreateReqs(familyID, assigneeID uint, assignedDate string, parsedTasks []services.ParsedTask) []CreateTaskReq {
	createdTasks := make([]CreateTaskReq, 0, len(parsedTasks))

	for _, task := range parsedTasks {
		subject, groupTitle, content := normalizeTaskFields(task.Subject, task.GroupTitle, task.Title)
		if content == "" {
			continue
		}

		createdTasks = append(createdTasks, CreateTaskReq{
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

func persistTasks(tasks []CreateTaskReq) error {
	for _, task := range tasks {
		assignedDate, err := parseAssignedDate(task.AssignedDate)
		if err != nil {
			return err
		}

		if err := services.SaveTaskWithGroupToMDAtDate(task.FamilyID, task.AssigneeID, task.Subject, task.GroupTitle, task.Content, assignedDate); err != nil {
			return err
		}
	}

	return nil
}

func buildTaskSummaries(tasks []services.MarkdownTask) ([]TaskGroupSummary, []HomeworkGroupSummary, map[string]interface{}) {
	groupMap := make(map[string]*TaskGroupSummary)
	homeworkMap := make(map[string]*HomeworkGroupSummary)
	subjectOrder := make([]string, 0)
	homeworkOrder := make([]string, 0)
	summary := map[string]interface{}{
		"total":     len(tasks),
		"completed": 0,
		"pending":   0,
		"status":    "empty",
	}

	for _, task := range tasks {
		subject := strings.TrimSpace(task.Subject)
		if subject == "" {
			subject = "未分类"
		}
		groupTitle := strings.TrimSpace(task.GroupTitle)
		if groupTitle == "" {
			groupTitle = strings.TrimSpace(task.Content)
		}

		subjectGroup, exists := groupMap[subject]
		if !exists {
			subjectGroup = &TaskGroupSummary{Subject: subject}
			groupMap[subject] = subjectGroup
			subjectOrder = append(subjectOrder, subject)
		}

		homeworkKey := subject + "\x00" + groupTitle
		homeworkGroup, exists := homeworkMap[homeworkKey]
		if !exists {
			homeworkGroup = &HomeworkGroupSummary{
				Subject:    subject,
				GroupTitle: groupTitle,
			}
			homeworkMap[homeworkKey] = homeworkGroup
			homeworkOrder = append(homeworkOrder, homeworkKey)
		}

		subjectGroup.Total++
		homeworkGroup.Total++
		if task.Completed {
			subjectGroup.Completed++
			homeworkGroup.Completed++
			summary["completed"] = summary["completed"].(int) + 1
		} else {
			subjectGroup.Pending++
			homeworkGroup.Pending++
			summary["pending"] = summary["pending"].(int) + 1
		}
	}

	groups := make([]TaskGroupSummary, 0, len(groupMap))
	for _, subject := range subjectOrder {
		group := groupMap[subject]
		switch {
		case group.Completed == 0:
			group.Status = "pending"
		case group.Completed == group.Total:
			group.Status = "completed"
		default:
			group.Status = "partial"
		}
		groups = append(groups, *group)
	}

	homeworkGroups := make([]HomeworkGroupSummary, 0, len(homeworkMap))
	for _, homeworkKey := range homeworkOrder {
		group := homeworkMap[homeworkKey]
		switch {
		case group.Completed == 0:
			group.Status = "pending"
		case group.Completed == group.Total:
			group.Status = "completed"
		default:
			group.Status = "partial"
		}
		homeworkGroups = append(homeworkGroups, *group)
	}

	switch {
	case len(tasks) == 0:
		summary["status"] = "empty"
	case summary["completed"].(int) == 0:
		summary["status"] = "pending"
	case summary["completed"].(int) == len(tasks):
		summary["status"] = "completed"
	default:
		summary["status"] = "partial"
	}

	return groups, homeworkGroups, summary
}

func respondWithTaskBoard(c *gin.Context, date time.Time, tasks []services.MarkdownTask, extra gin.H) {
	groups, homeworkGroups, summary := buildTaskSummaries(tasks)
	payload := gin.H{
		"date":            date.Format("2006-01-02"),
		"tasks":           tasks,
		"groups":          groups,
		"homework_groups": homeworkGroups,
		"summary":         summary,
	}

	for key, value := range extra {
		payload[key] = value
	}

	c.JSON(http.StatusOK, payload)
}

func CreateTask(c *gin.Context) {
	var req CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	req.Subject, req.GroupTitle, req.Content = normalizeTaskFields(req.Subject, req.GroupTitle, req.Content)
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty"})
		return
	}

	assignedDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.SaveTaskWithGroupToMDAtDate(req.FamilyID, req.AssigneeID, req.Subject, req.GroupTitle, req.Content, assignedDate); err != nil {
		log.Printf("Failed to append task to markdown: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store task in markdown"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task written to daily markdown successfully",
		"date":    assignedDate.Format("2006-01-02"),
		"task":    req,
	})
}

func ParseAndCreateTasks(c *gin.Context) {
	var req ParseTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	parsed, err := services.ParseParentInput(strings.TrimSpace(req.RawText))
	if err != nil {
		log.Printf("Failed to parse parent input via Agent Core: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse task text via Agent Core"})
		return
	}

	if parsed.Status != "success" || len(parsed.Data) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":          "Agent Core could not extract valid tasks",
			"agent_response": parsed,
		})
		return
	}

	assignedDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdTasks := mapParsedTasksToCreateReqs(req.FamilyID, req.AssigneeID, assignedDate.Format("2006-01-02"), parsed.Data)

	if len(createdTasks) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Agent Core returned tasks without usable content"})
		return
	}

	if req.AutoCreate {
		if err := persistTasks(createdTasks); err != nil {
			log.Printf("Failed to append parsed task to markdown: %v", err)
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

func ConfirmTasks(c *gin.Context) {
	var req ConfirmTasksReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	if len(req.Tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tasks cannot be empty"})
		return
	}

	assignedDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdTasks := make([]CreateTaskReq, 0, len(req.Tasks))
	for _, task := range req.Tasks {
		subject, groupTitle, content := normalizeTaskFields(task.Subject, task.GroupTitle, task.Title)
		if content == "" {
			continue
		}

		createdTasks = append(createdTasks, CreateTaskReq{
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

	if err := persistTasks(createdTasks); err != nil {
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

func ListTasks(c *gin.Context) {
	familyIDStr := c.Query("family_id")
	userIDStr := c.Query("user_id")

	if familyIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family_id and user_id query params are required"})
		return
	}

	familyID, err := parseUintParam(familyIDStr, "family_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := parseUintParam(userIDStr, "user_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetDate, err := parseAssignedDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, err := services.GetTasksFromMD(familyID, userID, targetDate)
	if err != nil {
		log.Printf("Error reading markdown tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks from markdown"})
		return
	}

	respondWithTaskBoard(c, targetDate, tasks, gin.H{})
}

func UpdateSingleTaskStatus(c *gin.Context) {
	var req UpdateTaskStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, updatedCount, err := services.UpdateTaskCompletionByID(req.FamilyID, req.AssigneeID, targetDate, req.TaskID, req.Completed)
	if err != nil {
		log.Printf("Failed to update single task status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		return
	}

	if updatedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	respondWithTaskBoard(c, targetDate, tasks, gin.H{
		"message":       "Single task status updated successfully",
		"updated_count": updatedCount,
	})
}

func UpdateTaskGroupStatus(c *gin.Context) {
	var req UpdateTaskGroupStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		tasks        []services.MarkdownTask
		updatedCount int
	)
	if strings.TrimSpace(req.GroupTitle) != "" {
		tasks, updatedCount, err = services.UpdateTaskCompletionByHomeworkGroup(
			req.FamilyID,
			req.AssigneeID,
			targetDate,
			req.Subject,
			req.GroupTitle,
			req.Completed,
		)
	} else {
		tasks, updatedCount, err = services.UpdateTaskCompletionBySubject(req.FamilyID, req.AssigneeID, targetDate, req.Subject, req.Completed)
	}
	if err != nil {
		log.Printf("Failed to update task group status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task group status"})
		return
	}

	if updatedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No tasks found for the provided subject"})
		return
	}

	respondWithTaskBoard(c, targetDate, tasks, gin.H{
		"message":       "Task group status updated successfully",
		"updated_count": updatedCount,
		"subject":       strings.TrimSpace(req.Subject),
		"group_title":   strings.TrimSpace(req.GroupTitle),
	})
}

func UpdateAllTasksStatus(c *gin.Context) {
	var req UpdateAllTasksStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	targetDate, err := parseAssignedDate(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, updatedCount, err := services.UpdateAllTasksCompletion(req.FamilyID, req.AssigneeID, targetDate, req.Completed)
	if err != nil {
		log.Printf("Failed to update all task statuses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update all task statuses"})
		return
	}

	respondWithTaskBoard(c, targetDate, tasks, gin.H{
		"message":       "All task statuses updated successfully",
		"updated_count": updatedCount,
	})
}
