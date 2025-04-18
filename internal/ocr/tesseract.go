package ocr

import (
	"errors"
	"fmt"
	"time"

	"github.com/otiai10/gosseract/v2"
)

// Config holds configuration for the OCR engine
type Config struct {
	Languages       []string
	PageSegMode     gosseract.PageSegMode
	MaxRetries      int
	RetryDelay      time.Duration
	TesseractConfig map[string]string
}

// DefaultConfig returns a default configuration for the OCR engine
func DefaultConfig() *Config {
	return &Config{
		Languages:   []string{"eng"},
		PageSegMode: gosseract.PSM_AUTO,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		TesseractConfig: map[string]string{
			"tessedit_char_whitelist": "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz.,;:$%&()[]{}+-*/=@#\"'<>!?",
		},
	}
}

// OCREngine represents a Tesseract OCR engine
type OCREngine struct {
	client *gosseract.Client
	config *Config
}

// NewOCREngine creates a new OCR engine with the given configuration
func NewOCREngine(config *Config) *OCREngine {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &OCREngine{
		client: gosseract.NewClient(),
		config: config,
	}
}

// Close releases resources used by the OCR engine
func (e *OCREngine) Close() error {
	return e.client.Close()
}

// ExtractText extracts text from an image with retry mechanism
func (e *OCREngine) ExtractText(imageData []byte) (string, error) {
	var lastErr error
	
	for i := 0; i < e.config.MaxRetries; i++ {
		text, err := e.extractTextOnce(imageData)
		if err == nil {
			return text, nil
		}
		
		lastErr = err
		time.Sleep(e.config.RetryDelay)
	}
	
	return "", fmt.Errorf("failed to extract text after %d attempts: %w", e.config.MaxRetries, lastErr)
}

// extractTextOnce performs a single OCR attempt
func (e *OCREngine) extractTextOnce(imageData []byte) (string, error) {
	if err := e.client.SetImageFromBytes(imageData); err != nil {
		return "", fmt.Errorf("failed to set image: %w", err)
	}
	
	if err := e.client.SetLanguage(e.config.Languages...); err != nil {
		return "", fmt.Errorf("failed to set language: %w", err)
	}
	
	if err := e.client.SetPageSegMode(e.config.PageSegMode); err != nil {
		return "", fmt.Errorf("failed to set page segmentation mode: %w", err)
	}
	
	for key, value := range e.config.TesseractConfig {
		if err := e.client.SetVariable(gosseract.SettableVariable(key), value); err != nil {
			return "", fmt.Errorf("failed to set variable %s: %w", key, err)
		}
	}
	
	text, err := e.client.Text()
	if err != nil {
		return "", fmt.Errorf("failed to extract text: %w", err)
	}
	
	if text == "" {
		return "", errors.New("no text extracted from image")
	}
	
	return text, nil
}

// SetLanguages sets the languages for OCR
func (e *OCREngine) SetLanguages(languages ...string) {
	e.config.Languages = languages
}

// SetPageSegMode sets the page segmentation mode
func (e *OCREngine) SetPageSegMode(mode gosseract.PageSegMode) {
	e.config.PageSegMode = mode
}
