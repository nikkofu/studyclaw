package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	weeklyinsights "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/weeklyinsights"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
)

type StatsHandler struct {
	taskboard *taskboardapp.Service
	weekly    *weeklyinsights.Service
}

func NewStatsHandler(taskboard *taskboardapp.Service, weekly *weeklyinsights.Service) *StatsHandler {
	return &StatsHandler{taskboard: taskboard, weekly: weekly}
}

func (h *StatsHandler) GetWeeklyStats(c *gin.Context) {
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

	endDate, ok := parseOptionalDateOrAbort(c, "end_date", c.Query("end_date"))
	if !ok {
		return
	}
	startDate := endDate.AddDate(0, 0, -6)

	allDailyData := make([]map[string]interface{}, 0)
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		tasks, err := h.taskboard.ListTasks(familyID, userID, date)
		if err == nil && len(tasks) > 0 {
			allDailyData = append(allDailyData, map[string]interface{}{
				"date":  date.Format("2006-01-02"),
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

	insights, err := h.weekly.Generate(context.Background(), allDailyData)
	if err != nil {
		log.Printf("Weekly insight workflow failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message":   "Weekly insight workflow failed. Returning raw statistics.",
			"raw_stats": allDailyData,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Weekly stats generated successfully",
		"raw_stats": allDailyData,
		"insights":  insights,
	})
}
