package main

import (
	"fmt"
	"log"

	"github.com/ridwanfathin/invoice-processor-service/internal/config"
	"github.com/ridwanfathin/invoice-processor-service/internal/handler"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
	"github.com/ridwanfathin/invoice-processor-service/internal/server"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

func main() {
	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize OpenRouter client for invoice processing
	openRouterClient := openrouter.NewClient(&openrouter.Config{
		APIKey:         cfg.OpenRouterAPIKey,
		ModelID:        cfg.OpenRouterModelID,
		Timeout:        cfg.OpenRouterTimeout,
		SupabaseURL:    cfg.SupabaseURL,
		SupabaseBucket: cfg.SupabaseBucket,
		SupabaseAPIKey: cfg.SupabaseAPIKey,
	})

	// Create invoice processor service
	log.Println("Creating AI-based invoice processor service...")
	processorService := service.NewAIProcessorService(openRouterClient, cfg.MaxWorkers)

	// Initialize repository
	log.Println("Initializing repository...")
	// Use SupabaseRepository instead of FileRepository to eliminate redundant storage
	repo := repository.NewSupabaseRepository(openRouterClient)

	// Set repository for processor service
	processorService.SetRepository(repo)

	// Create handler
	invoiceHandler := handler.NewInvoiceHandler(processorService)

	// Create and configure server
	log.Println("Configuring server...")
	appServer := server.NewServer(cfg, invoiceHandler)
	
	// Set processor service in the server for clean shutdown
	appServer.SetProcessorService(processorService)

	// Start server (blocking call)
	log.Printf("Starting server on port %d...", cfg.Port)
	if err := appServer.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	fmt.Println("Server shutdown complete")
}
