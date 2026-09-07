package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"gh-pin-repo-go/internal/api"
	"gh-pin-repo-go/internal/cache"
	"gh-pin-repo-go/internal/service"
)

// @title           GitHub Pinned Repositories API
// @version         1.0
// @description     High-performance, production-ready RESTful API in Golang to query GitHub pinned repositories with Swagger UI.
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.url     https://github.com
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /
func main() {
	// Maximize CPU utilization for high concurrency
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Read environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cacheTTLMin := 10
	if ttlStr := os.Getenv("CACHE_TTL_MINUTES"); ttlStr != "" {
		if val, err := strconv.Atoi(ttlStr); err == nil && val > 0 {
			cacheTTLMin = val
		}
	}

	// Initialize sharded in-memory cache
	cacheInstance := cache.New(time.Duration(cacheTTLMin) * time.Minute)
	defer cacheInstance.Close()

	// Initialize scraper service and handler
	pinService := service.NewPinService(cacheInstance, time.Duration(cacheTTLMin)*time.Minute)
	handler := api.NewHandler(pinService)

	// Setup routes
	router := api.SetupRouter(handler)

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Run server in goroutine
	go func() {
		log.Printf("🚀 Server is running on http://localhost:%s", port)
		log.Printf("📖 Swagger UI available at: http://localhost:%s/swagger/index.html", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}

	fmt.Println("Server exited cleanly.")
}
