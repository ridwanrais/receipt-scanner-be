package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ridwanfathin/invoice-ocr-service/internal/handler"
	"github.com/ridwanfathin/invoice-ocr-service/internal/ocr"
	"github.com/ridwanfathin/invoice-ocr-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-ocr-service/internal/service"
	"github.com/ridwanfathin/invoice-ocr-service/internal/util"
)

func main() {
	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Invoice OCR Service...")

	// Get the executable directory
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Warning: Could not determine executable path: %v", err)
	}
	
	// Determine project root directory
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	envPath := filepath.Join(projectRoot, ".env")
	
	// Load .env file if it exists
	if err := godotenv.Load(envPath); err != nil {
		// Try loading from current directory as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading .env file. Using environment variables.")
			log.Printf("Error details: %v", err)
		} else {
			log.Println("Loaded environment variables from current directory .env file")
		}
	} else {
		log.Printf("Loaded environment variables from %s", envPath)
		
		// Debug: Print loaded environment variables
		log.Printf("Debug - OPENROUTER_API_KEY: %s", os.Getenv("OPENROUTER_API_KEY"))
		log.Printf("Debug - SUPABASE_URL: %s", os.Getenv("SUPABASE_URL"))
		log.Printf("Debug - SUPABASE_BUCKET: %s", os.Getenv("SUPABASE_BUCKET"))
		log.Printf("Debug - SUPABASE_API_KEY: %s", os.Getenv("SUPABASE_API_KEY")[:10] + "...")
	}

	// Load configuration from environment variables
	port := getEnvInt("PORT", 8080)
	maxWorkers := getEnvInt("MAX_WORKERS", 5)
	useOpenRouter := getEnvBool("USE_OPENROUTER", true)
	openRouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	openRouterModelID := getEnvString("OPENROUTER_MODEL_ID", "meta-llama/llama-3.2-11b-vision-instruct:free")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseBucket := getEnvString("SUPABASE_BUCKET", "invoices")
	supabaseAPIKey := os.Getenv("SUPABASE_API_KEY")

	// Create image processor
	imageProcessor := util.NewImageProcessor()

	// Create OCR service based on configuration
	var ocrService service.OCRServiceInterface

	if useOpenRouter {
		log.Println("Using OpenRouter for OCR processing")

		// Check if API key is provided
		if openRouterAPIKey == "" {
			log.Println("Warning: No OpenRouter API key provided. API requests will fail.")
		}

		// Check if Supabase URL is provided
		if supabaseURL == "" {
			log.Println("Warning: No Supabase URL provided. Image uploads will fail.")
		}

		// Check if Supabase API key is provided
		if supabaseAPIKey == "" {
			log.Println("Warning: No Supabase API key provided. Image uploads will fail.")
		}

		// Create OpenRouter client
		openRouterConfig := &openrouter.Config{
			APIKey:         openRouterAPIKey,
			ModelID:        openRouterModelID,
			Timeout:        60 * time.Second,
			SupabaseURL:    supabaseURL,
			SupabaseBucket: supabaseBucket,
			SupabaseAPIKey: supabaseAPIKey,
		}
		openRouterClient := openrouter.NewClient(openRouterConfig)

		// Create OpenRouter OCR service
		ocrService = service.NewOpenRouterService(openRouterClient, maxWorkers)
	} else {
		log.Println("Using Tesseract for OCR processing")

		// Load Tesseract configuration
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

		// Create Tesseract OCR service
		ocrService = service.NewOCRService(ocrEngine, imageProcessor, maxWorkers)
	}

	// Ensure service is shut down properly
	defer func() {
		if shutdowner, ok := ocrService.(interface{ Shutdown() }); ok {
			shutdowner.Shutdown()
		}
	}()

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
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"ocr_engine": func() string {
				if useOpenRouter {
					return "openrouter"
				}
				return "tesseract"
			}(),
		})
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

// getEnvBool gets a boolean from an environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	valueStr = strings.ToLower(valueStr)
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

// getEnvString gets a string from an environment variable with a default value
func getEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
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
