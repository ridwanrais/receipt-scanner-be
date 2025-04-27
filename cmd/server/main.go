package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ridwanfathin/invoice-processor-service/internal/config"
	"github.com/ridwanfathin/invoice-processor-service/internal/database"
	"github.com/ridwanfathin/invoice-processor-service/internal/handler"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
	"github.com/ridwanfathin/invoice-processor-service/internal/server"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

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
		APIKey:         cfg.OpenRouterAPIKey,
		ModelID:        cfg.OpenRouterModelID,
		Timeout:        cfg.OpenRouterTimeout,
		SupabaseURL:    cfg.SupabaseURL,
		SupabaseBucket: cfg.SupabaseBucket,
		SupabaseAPIKey: cfg.SupabaseAPIKey,
	})

	// Initialize PostgreSQL database connection
	var db *database.PostgresDB
	var receiptRepo repository.ReceiptRepository
	var invoiceRepo repository.InvoiceRepository
	
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
	log.Println("Successfully connected to PostgreSQL database.")
	
	// For now, use the SupabaseRepository for invoices until we fully migrate
	invoiceRepo = repository.NewSupabaseRepository(openRouterClient)

	// Initialize services
	log.Println("Initializing services...")
	receiptService := service.NewReceiptService(receiptRepo, openRouterClient, cfg.MaxWorkers)
	
	// For backward compatibility, create the AI processor service
	processorService := service.NewAIProcessorService(openRouterClient, cfg.MaxWorkers)
	processorService.SetRepository(invoiceRepo)

	// Initialize handlers
	log.Println("Initializing API handlers...")
	receiptHandler := handler.NewReceiptHandler(receiptService)
	invoiceHandler := handler.NewInvoiceHandler(processorService)

	// Create and configure server
	log.Println("Configuring server...")
	appServer := server.NewServer(cfg, invoiceHandler)
	
	// Set the receipt handler and service
	appServer.SetReceiptHandler(receiptHandler)
	appServer.SetReceiptService(receiptService)
	
	// Register receipt API routes
	appServer.RegisterReceiptRoutes()
	
	// Set processor service in the server for clean shutdown
	appServer.SetProcessorService(processorService)

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
