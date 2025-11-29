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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the HTTP server for the receipt scanning service
type Server struct {
	router         *gin.Engine
	httpServer     *http.Server
	receiptHandler *handler.ReceiptHandler
	receiptService service.ReceiptService
	config         *config.Config
}

// NewServer creates and configures a new server instance
func NewServer(cfg *config.Config) *Server {
	// Create router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestResponseLogger(middleware.LoggerConfig{
		Format: cfg.LogFormat,
		Level:  cfg.LogLevel,
	}))

	// Create server
	server := &Server{
		router: router,
		config: cfg,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      router,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
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

// GetRouter returns the gin router instance
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// setupRoutes configures all application routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API documentation endpoints
	// Access the Swagger UI at http://localhost:8080/api-docs/index.html
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler)
	s.router.GET("/api-docs/*any", swaggerHandler)

	s.router.GET("/api-docs", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/api-docs/index.html")
	})
}

// RegisterReceiptRoutes registers the receipt API routes
// This must be called after SetReceiptHandler
// Deprecated: Routes are now registered directly in main.go with auth middleware
func (s *Server) RegisterReceiptRoutes() {
	// This method is kept for backwards compatibility but is no longer used
	// Routes are registered in main.go to allow auth middleware injection
	log.Println("Note: RegisterReceiptRoutes is deprecated, routes should be registered in main.go")
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

	// Shutdown receipt service if available (if it implements a shutdown method)
	if shutdownable, ok := s.receiptService.(interface{ Shutdown() }); ok {
		shutdownable.Shutdown()
	}

	// Shutdown server
	return s.httpServer.Shutdown(ctx)
}
