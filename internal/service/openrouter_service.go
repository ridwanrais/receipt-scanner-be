package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ridwanfathin/invoice-ocr-service/internal/model"
	"github.com/ridwanfathin/invoice-ocr-service/internal/openrouter"
)

// OpenRouterService implements the OCRServiceInterface using OpenRouter
type OpenRouterService struct {
	client      *openrouter.Client
	maxWorkers  int
	workerQueue chan struct{}
}

// NewOpenRouterService creates a new OpenRouter service
func NewOpenRouterService(client *openrouter.Client, maxWorkers int) *OpenRouterService {
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 workers
	}

	return &OpenRouterService{
		client:      client,
		maxWorkers:  maxWorkers,
		workerQueue: make(chan struct{}, maxWorkers),
	}
}

// ProcessInvoice processes an invoice image using OpenRouter
func (s *OpenRouterService) ProcessInvoice(ctx context.Context, request *model.OCRRequest) (*model.OCRResponse, error) {
	// Initialize the response
	response := &model.OCRResponse{}

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
		return nil, ctx.Err()
	}

	// Process the invoice
	invoice, err := s.client.ExtractInvoiceData(request.File)
	if err != nil {
		log.Printf("OpenRouter extraction failed: %v", err)
		response.Error = fmt.Sprintf("OpenRouter extraction failed: %v", err)
		return response, nil
	}

	// Set the invoice in the response
	response.Invoice = invoice
	return response, nil
}

// ProcessInvoiceBatch processes multiple invoice images in parallel
func (s *OpenRouterService) ProcessInvoiceBatch(ctx context.Context, requests []*model.OCRRequest) ([]*model.OCRResponse, error) {
	var wg sync.WaitGroup
	responses := make([]*model.OCRResponse, len(requests))

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req *model.OCRRequest) {
			defer wg.Done()
			resp, err := s.ProcessInvoice(ctx, req)
			if err != nil {
				resp = &model.OCRResponse{
					Error: fmt.Sprintf("Failed to process invoice: %v", err),
				}
			}
			responses[idx] = resp
		}(i, request)
	}

	wg.Wait()
	return responses, nil
}
