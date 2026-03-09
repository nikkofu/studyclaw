package main

import (
	"log"

	"github.com/nikkofu/studyclaw/api-server/config"
	"github.com/nikkofu/studyclaw/api-server/routes"
)

func main() {
	log.Println("StudyClaw API Server starting...")

	// 1. Load Configurations
	config.LoadConfig()

	// 2. Setup Routes
	r := routes.SetupRouter()

	// 3. Start Server
	port := config.GetEnv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
