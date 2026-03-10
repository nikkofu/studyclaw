package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/internal/app"
	"github.com/nikkofu/studyclaw/api-server/internal/interfaces/http/handlers"
)

func SetupRouter(container *app.Container) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware())

	authHandler := handlers.NewAuthHandler()
	internalAgentHandler := handlers.NewAgentInternalHandler(container.TaskParse, container.Weekly)
	taskHandler := handlers.NewTaskHandler(container.Taskboard, container.PhaseOne, container.TaskParse)
	dailyAssignmentHandler := handlers.NewDailyAssignmentHandler(container.PhaseOne, container.TaskParse)
	pointsHandler := handlers.NewPointsHandler(container.PhaseOne)
	wordsHandler := handlers.NewWordsHandler(container.PhaseOne)
	statsHandler := handlers.NewStatsHandler(container.PhaseOne)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	apiV1 := r.Group("/api/v1")
	{
		apiV1.POST("/auth/login", authHandler.Login)
		apiV1.POST("/internal/parse", internalAgentHandler.Parse)
		apiV1.POST("/internal/analyze/weekly", internalAgentHandler.AnalyzeWeekly)
		apiV1.POST("/tasks", taskHandler.CreateTask)
		apiV1.POST("/tasks/parse", taskHandler.ParseAndCreateTasks)
		apiV1.POST("/tasks/confirm", taskHandler.ConfirmTasks)
		apiV1.PATCH("/tasks/status/item", taskHandler.UpdateSingleTaskStatus)
		apiV1.PATCH("/tasks/status/group", taskHandler.UpdateTaskGroupStatus)
		apiV1.PATCH("/tasks/status/all", taskHandler.UpdateAllTasksStatus)
		apiV1.GET("/tasks", taskHandler.ListTasks)
		apiV1.POST("/daily-assignments/drafts/parse", dailyAssignmentHandler.ParseDraft)
		apiV1.POST("/daily-assignments/publish", dailyAssignmentHandler.Publish)
		apiV1.GET("/daily-assignments", dailyAssignmentHandler.GetDailyAssignment)
		apiV1.POST("/points/update", pointsHandler.UpdatePoints)
		apiV1.POST("/points/ledger", pointsHandler.CreateLedgerEntry)
		apiV1.GET("/points/ledger", pointsHandler.ListLedger)
		apiV1.GET("/points/balance", pointsHandler.GetBalance)
		apiV1.POST("/word-lists", wordsHandler.UpsertWordList)
		apiV1.GET("/word-lists", wordsHandler.GetWordList)
		apiV1.POST("/dictation-sessions/start", wordsHandler.StartDictationSession)
		apiV1.GET("/dictation-sessions/:session_id", wordsHandler.GetDictationSession)
		apiV1.POST("/dictation-sessions/:session_id/replay", wordsHandler.ReplayDictationSession)
		apiV1.POST("/dictation-sessions/:session_id/next", wordsHandler.NextDictationSession)
		apiV1.GET("/stats/daily", statsHandler.GetDailyStats)
		apiV1.GET("/stats/weekly", statsHandler.GetWeeklyStats)
		apiV1.GET("/stats/monthly", statsHandler.GetMonthlyStats)
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
