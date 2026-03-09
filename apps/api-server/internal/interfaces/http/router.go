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
	pointsHandler := handlers.NewPointsHandler()
	taskHandler := handlers.NewTaskHandler(container.Taskboard, container.TaskParse)
	statsHandler := handlers.NewStatsHandler(container.Taskboard, container.Weekly)

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
		apiV1.POST("/points/update", pointsHandler.UpdatePoints)
		apiV1.GET("/stats/weekly", statsHandler.GetWeeklyStats)
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
