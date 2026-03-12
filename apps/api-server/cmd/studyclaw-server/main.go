package main

import (
	"log"

	"github.com/nikkofu/studyclaw/api-server/config"
	"github.com/nikkofu/studyclaw/api-server/internal/app"
	httpapi "github.com/nikkofu/studyclaw/api-server/internal/interfaces/http"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/logging"
)

func main() {
	config.LoadConfig()
	if err := logging.Init(); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}

	log.Println("StudyClaw Go Server starting...")
	container := app.NewContainer()
	router := httpapi.SetupRouter(container)

	port := config.GetEnv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
