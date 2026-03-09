package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/services"
)

func GetWeeklyStats(c *gin.Context) {
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

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -6) // 7 days inclusive

	var allDailyData []map[string]interface{}

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		tasks, err := services.GetTasksFromMD(familyID, userID, d)
		if err == nil && len(tasks) > 0 {
			allDailyData = append(allDailyData, map[string]interface{}{
				"date":  d.Format("2006-01-02"),
				"tasks": tasks,
			})
		}
	}

	if len(allDailyData) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No data found for the past week in the Markdown logs.",
			"data":    nil,
		})
		return
	}

	mlResponse, err := services.AnalyzeWeeklyStats(allDailyData)
	if err != nil {
		log.Printf("Agent Core unreachable: %v. Returning raw Markdown stats.", err)
		c.JSON(http.StatusOK, gin.H{
			"message":   "Agent Core unreachable. Returning raw statistics.",
			"raw_stats": allDailyData,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Weekly stats generated successfully",
		"raw_stats": allDailyData,
		"insights":  mlResponse,
	})
}
