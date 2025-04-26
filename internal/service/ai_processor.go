package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ridwanfathin/invoice-processor-service/internal/model"
	"github.com/ridwanfathin/invoice-processor-service/internal/openrouter"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
)

// AIProcessorService implements the InvoiceProcessorServicer interface using OpenRouter AI
type AIProcessorService struct {
	client      *openrouter.Client
	maxWorkers  int
	workerQueue chan struct{}
	repository  repository.InvoiceRepository
}

// NewAIProcessorService creates a new AI-based invoice processor service
func NewAIProcessorService(client *openrouter.Client, maxWorkers int) *AIProcessorService {
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 workers
	}

	return &AIProcessorService{
		client:      client,
		maxWorkers:  maxWorkers,
		workerQueue: make(chan struct{}, maxWorkers),
	}
}

// SetRepository sets the repository for the service
func (s *AIProcessorService) SetRepository(repo repository.InvoiceRepository) {
	s.repository = repo
}

// ProcessInvoice processes an invoice image using OpenRouter AI
func (s *AIProcessorService) ProcessInvoice(ctx context.Context, request *model.InvoiceProcessingRequest) (*model.InvoiceProcessingResponse, error) {
	// Initialize the response
	response := &model.InvoiceProcessingResponse{}

	// Acquire a worker from the pool
	select {
	case s.workerQueue <- struct{}{}:
		// Worker acquired, continue processing
		defer func() {
			// Release the worker back to the pool
			<-s.workerQueue
		}()
	case <-ctx.Done():
		// Context cancelled while waiting for a worker
		return nil, &InvoiceProcessingError{
			Op:  "acquire_worker",
			Err: ctx.Err(),
		}
	}

	// Store the image if repository is available
	if s.repository != nil {
		_, err := s.repository.StoreImage(ctx, request.File)
		if err != nil {
			// Log the error but continue with processing
			log.Printf("Error storing image: %v", err)
		}
	}

	// Process the invoice
	domainInvoice, err := s.client.ExtractInvoiceData(request.File)
	if err != nil {
		log.Printf("AI extraction failed: %v", err)
		response.Error = fmt.Sprintf("AI extraction failed: %v", err)
		return response, nil
	}

	// Store the invoice if repository is available
	if s.repository != nil && domainInvoice.InvoiceNumber != "" {
		if err := s.repository.StoreInvoice(ctx, domainInvoice); err != nil {
			// Log the error but continue
			log.Printf("Error storing invoice: %v", err)
		}
	}

	// Convert domain model to DTO
	invoiceDTO := &model.InvoiceDTO{}
	invoiceDTO.FromDomain(domainInvoice)

	// Set the invoice in the response
	response.Invoice = invoiceDTO
	return response, nil
}

// ProcessInvoiceBatch processes multiple invoice images in parallel
func (s *AIProcessorService) ProcessInvoiceBatch(ctx context.Context, requests []*model.InvoiceProcessingRequest) ([]*model.InvoiceProcessingResponse, error) {
	var wg sync.WaitGroup
	responses := make([]*model.InvoiceProcessingResponse, len(requests))

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req *model.InvoiceProcessingRequest) {
			defer wg.Done()
			resp, err := s.ProcessInvoice(ctx, req)
			if err != nil {
				resp = &model.InvoiceProcessingResponse{
					Error: fmt.Sprintf("Failed to process invoice: %v", err),
				}
			}
			responses[idx] = resp
		}(i, request)
	}

	wg.Wait()
	return responses, nil
}

// Shutdown implements the shutdown method from InvoiceProcessorServicer interface
func (s *AIProcessorService) Shutdown() {
	// Clean up any resources if needed
	close(s.workerQueue)
}
