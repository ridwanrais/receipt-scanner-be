package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/ridwanfathin/invoice-processor-service/docs"
	"github.com/ridwanfathin/invoice-processor-service/internal/config"
	"github.com/ridwanfathin/invoice-processor-service/internal/currency"
	"github.com/ridwanfathin/invoice-processor-service/internal/database"
	"github.com/ridwanfathin/invoice-processor-service/internal/handler"
	"github.com/ridwanfathin/invoice-processor-service/internal/middleware"
	"github.com/ridwanfathin/invoice-processor-service/internal/mlxclient"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
	"github.com/ridwanfathin/invoice-processor-service/internal/server"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
	"github.com/ridwanfathin/invoice-processor-service/internal/storage"
)

// @title Receipt Scanner API
// @version 1.0
// @description API for scanning and managing receipts with AI-powered data extraction
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Create a context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize OpenRouter client for receipt processing
	openRouterClient := openrouter.NewClient(&openrouter.Config{
		APIKey:            cfg.OpenRouterAPIKey,
		ModelID:           cfg.OpenRouterModelID,
		Timeout:           cfg.OpenRouterTimeout,
		S3Endpoint:        cfg.SupabaseS3Endpoint,
		S3AccessKeyID:     cfg.SupabaseAccessKeyID,
		S3AccessKeySecret: cfg.SupabaseAccessKeySecret,
		SupabaseBucket:    cfg.SupabaseBucket,
		S3Region:          cfg.SupabaseRegion,
	})

	// Initialize S3 uploader for image storage
	var s3Uploader *storage.S3Uploader
	if cfg.SupabaseS3Endpoint != "" {
		log.Println("Initializing S3 uploader...")
		var err error
		s3Uploader, err = storage.NewS3Uploader(&storage.Config{
			Endpoint:        cfg.SupabaseS3Endpoint,
			AccessKeyID:     cfg.SupabaseAccessKeyID,
			AccessKeySecret: cfg.SupabaseAccessKeySecret,
			Bucket:          cfg.SupabaseBucket,
			Region:          cfg.SupabaseRegion,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize S3 uploader: %v", err)
		}
	}

	// Initialize MLX client if enabled
	var mlxClient *mlxclient.Client
	if cfg.UseMLXService {
		log.Println("MLX service is enabled, initializing MLX client...")
		mlxClient = mlxclient.NewClient(&mlxclient.Config{
			BaseURL: cfg.MLXServiceURL,
			Timeout: cfg.MLXTimeout,
		})
		log.Printf("MLX client initialized with URL: %s", cfg.MLXServiceURL)
	}

	// Initialize PostgreSQL database connection
	var db *database.PostgresDB
	var receiptRepo repository.ReceiptRepository
	var userRepo repository.UserRepository

	// Require database connection - exit if not available
	if cfg.PostgresDBURL == "" {
		log.Fatalf("Error: PostgreSQL database URL not configured. Please set POSTGRES_DB_URL environment variable")
	}

	log.Println("Initializing database connection...")
	db, err = database.NewPostgresDB()
	if err != nil {
		log.Fatalf("Error: Failed to connect to database: %v", err)
	}

	defer db.Close()
	receiptRepo = repository.NewPostgresReceiptRepository(db.GetPool())
	userRepo = repository.NewPostgresUserRepository(db.GetPool())
	log.Println("Successfully connected to PostgreSQL database.")

	// Initialize services
	log.Println("Initializing services...")
	receiptService := service.NewReceiptService(receiptRepo, openRouterClient, mlxClient, s3Uploader, cfg.UseMLXService, cfg.MaxWorkers)

	authService := service.NewAuthService(service.AuthServiceConfig{
		UserRepo:              userRepo,
		GoogleClientID:        cfg.GoogleClientIDWeb,
		GoogleClientSecret:    cfg.GoogleClientSecretWeb,
		GoogleRedirectURL:     cfg.GoogleRedirectURLWeb,
		GoogleClientIDAndroid: cfg.GoogleClientIDAndroid,
		GoogleClientIDIOS:     cfg.GoogleClientIDIOS,
		JWTSecret:             cfg.JWTSecret,
		JWTAccessExpiration:   cfg.JWTAccessExpiration,
		JWTRefreshExpiration:  cfg.JWTRefreshExpiration,
	})

	// Initialize currency client
	log.Println("Initializing currency client...")
	currencyClient := currency.NewClient()

	// Initialize handlers
	log.Println("Initializing API handlers...")
	receiptHandler := handler.NewReceiptHandler(receiptService)
	authHandler := handler.NewAuthHandler(authService, cfg.FrontendURL)
	currencyHandler := handler.NewCurrencyHandler(currencyClient)
	analyticsHandler := handler.NewAnalyticsHandler(receiptRepo, currencyClient)

	// Create and configure server
	log.Println("Configuring server...")
	appServer := server.NewServer(cfg)

	// Set the receipt handler and service
	appServer.SetReceiptHandler(receiptHandler)
	appServer.SetReceiptService(receiptService)

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(authService)

	// Register API routes
	receiptHandler.RegisterRoutes(appServer.GetRouter(), authMiddleware)
	authHandler.RegisterRoutes(appServer.GetRouter(), authMiddleware)
	currencyHandler.RegisterCurrencyRoutes(appServer.GetRouter().Group("/v1"))
	analyticsHandler.RegisterAnalyticsRoutes(appServer.GetRouter().Group("/v1"), authMiddleware)

	// Start server in a goroutine so we can handle shutdown gracefully
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on port %d...", cfg.Port)
		serverErr <- appServer.Start()
	}()

	// Wait for either server error or context cancellation
	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case <-ctx.Done():
		// Shutdown requested, stop the server gracefully
		log.Println("Shutting down server...")
		if err := appServer.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	fmt.Println("Server shutdown complete")
}
