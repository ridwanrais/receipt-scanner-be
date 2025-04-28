package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/config"
	"github.com/ridwanfathin/invoice-processor-service/internal/handler"
	"github.com/ridwanfathin/invoice-processor-service/internal/middleware"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

// Server represents the HTTP server for the invoice processing service
type Server struct {
	router         *gin.Engine
	httpServer     *http.Server
	invoiceHandler *handler.InvoiceHandler
	receiptHandler *handler.ReceiptHandler
	processor      service.InvoiceProcessorServicer
	receiptService service.ReceiptService
	config         *config.Config
}

// NewServer creates and configures a new server instance
func NewServer(cfg *config.Config, invoiceHandler *handler.InvoiceHandler) *Server {
	// Create router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// Create server
	server := &Server{
		router:         router,
		invoiceHandler: invoiceHandler,
		config:         cfg,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
		},
	}

	// Configure routes
	server.setupRoutes()

	return server
}

// SetReceiptHandler sets the receipt handler for the server
func (s *Server) SetReceiptHandler(receiptHandler *handler.ReceiptHandler) {
	s.receiptHandler = receiptHandler
}

// SetReceiptService sets the receipt service for the server
func (s *Server) SetReceiptService(receiptService service.ReceiptService) {
	s.receiptService = receiptService
}

// setupRoutes configures all application routes
func (s *Server) setupRoutes() {
	// Register invoice processing routes with the provided handler
	s.invoiceHandler.RegisterRoutes(s.router)

	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"processor_type": "ai", // We're using AI processor exclusively in the new architecture
		})
	})
}

// RegisterReceiptRoutes registers the receipt API routes
// This must be called after SetReceiptHandler
func (s *Server) RegisterReceiptRoutes() {
	if s.receiptHandler == nil {
		log.Println("Warning: Receipt handler is nil, skipping receipt routes registration")
		return
	}
	s.receiptHandler.RegisterRoutes(s.router)
}

// Start begins listening for requests and handles graceful shutdown
func (s *Server) Start() error {
	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %d", s.config.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the invoice processor service
	if s.processor != nil {
		s.processor.Shutdown()
	}

	// Shutdown server gracefully
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited gracefully")
	return nil
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown processor service if available
	if s.processor != nil {
		s.processor.Shutdown()
	}

	// Shutdown receipt service if available (if it implements a shutdown method)
	if shutdownable, ok := s.receiptService.(interface{ Shutdown() }); ok {
		shutdownable.Shutdown()
	}

	// Shutdown server
	return s.httpServer.Shutdown(ctx)
}

// SetProcessorService sets the invoice processor service reference for shutdown purposes
func (s *Server) SetProcessorService(processor service.InvoiceProcessorServicer) {
	s.processor = processor
}
