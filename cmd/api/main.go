package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Read environment (defaults to "development" when .env is not loaded externally)
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	// Set Gin mode based on environment
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// ---------------------------------------------------------------------------
	// Health-check route
	// Will be replaced by a structured router once internal packages are wired up.
	// ---------------------------------------------------------------------------
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	log.Printf("[worksphere-api] starting server — env=%s port=%s", appEnv, port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
