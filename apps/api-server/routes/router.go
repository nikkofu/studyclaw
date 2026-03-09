package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/handlers"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware())

	// Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	apiV1 := r.Group("/api/v1")
	{
		// Auth
		apiV1.POST("/auth/login", handlers.Login)

		// Tasks
		apiV1.POST("/tasks", handlers.CreateTask)
		apiV1.POST("/tasks/parse", handlers.ParseAndCreateTasks)
		apiV1.POST("/tasks/confirm", handlers.ConfirmTasks)
		apiV1.PATCH("/tasks/status/item", handlers.UpdateSingleTaskStatus)
		apiV1.PATCH("/tasks/status/group", handlers.UpdateTaskGroupStatus)
		apiV1.PATCH("/tasks/status/all", handlers.UpdateAllTasksStatus)
		apiV1.GET("/tasks", handlers.ListTasks)

		// Points
		apiV1.POST("/points/update", handlers.UpdatePoints)

		// Stats
		apiV1.GET("/stats/weekly", handlers.GetWeeklyStats)
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
