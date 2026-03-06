package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/models"
)

type CreateTaskReq struct {
	FamilyID   uint   `json:"family_id" binding:"required"`
	AssigneeID uint   `json:"assignee_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Subject    string `json:"subject"`
	RawText    string `json:"raw_text"`
}

func CreateTask(c *gin.Context) {
	var req CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	task := models.Task{
		FamilyID:    req.FamilyID,
		AssigneeID:  req.AssigneeID,
		Title:       req.Title,
		Subject:     req.Subject,
		RawText:     req.RawText,
		Status:      "pending",
		PointsValue: 1,
	}

	if result := models.DB.Create(&task); result.Error != nil {
		log.Printf("Failed to create task: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"task":    task,
	})
}

func ListTasks(c *gin.Context) {
	familyID := c.Query("family_id")
	if familyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family_id query param is required"})
		return
	}

	var tasks []models.Task
	if err := models.DB.Where("family_id = ?", familyID).Order("created_at desc").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
	})
}
