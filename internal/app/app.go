package app

import (
	"fmt"
	"log"

	"TaskFlow-Go/internal/database"

	"github.com/gin-gonic/gin"
)

func Run(addr ...string) {
	// --- Database ---
	cfg := database.LoadConfig()
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// --- Container (DI) ---
	container := NewContainer(db)

	// --- HTTP Server ---
	r := gin.Default()
	api := r.Group("/api/v1")
	container.SetupRoutes(api)

	port := ":8080"
	if len(addr) > 0 {
		port = addr[0]
	}

	log.Printf("Server starting on %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	_ = fmt.Sprintf
}
