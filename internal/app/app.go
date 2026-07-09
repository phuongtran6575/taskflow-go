package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/job"

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

	// --- Background Jobs ---
	jobDispatcher := job.NewDispatcher(db)
	jobDispatcher.StartDailyJobs()
	jobDispatcher.StartTaskDueSoonCron()

	// --- HTTP Server ---
	r := gin.Default()
	api := r.Group("/api/v1")
	container.SetupRoutes(api)

	port := ":8080"
	if len(addr) > 0 {
		port = addr[0]
	}

	log.Printf("Server starting on %s", port)

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	container.WSHub.Stop()
	log.Println("Server exited")
}
