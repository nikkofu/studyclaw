package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/handlers"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

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
		apiV1.GET("/tasks", handlers.ListTasks)
	}

	return r
}
