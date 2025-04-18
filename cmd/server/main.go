package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-ocr-service/internal/handler"
	"github.com/ridwanfathin/invoice-ocr-service/internal/ocr"
	"github.com/ridwanfathin/invoice-ocr-service/internal/service"
	"github.com/ridwanfathin/invoice-ocr-service/internal/util"
	"strings"
)

func main() {
	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Invoice OCR Service...")

	// Load configuration from environment variables
	port := getEnvInt("PORT", 8080)
	maxWorkers := getEnvInt("MAX_WORKERS", 5)
	languages := getEnvStringSlice("OCR_LANGUAGES", []string{"eng"})

	// Create OCR engine
	ocrConfig := &ocr.Config{
		Languages:   languages,
		PageSegMode: 3, // PSM_AUTO
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}
	ocrEngine := ocr.NewOCREngine(ocrConfig)
	defer ocrEngine.Close()

	// Create image processor
	imageProcessor := util.NewImageProcessor()

	// Create OCR service
	ocrService := service.NewOCRService(ocrEngine, imageProcessor, maxWorkers)
	defer ocrService.Shutdown()

	// Create HTTP handler
	ocrHandler := handler.NewOCRHandler(ocrService)

	// Set up Gin router
	router := gin.Default()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	
	// Register routes
	ocrHandler.RegisterRoutes(router)
	
	// Add health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

// getEnvInt gets an integer from an environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}
	
	return value
}

// getEnvStringSlice gets a string slice from a comma-separated environment variable
func getEnvStringSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	return strings.Split(valueStr, ",")
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}
