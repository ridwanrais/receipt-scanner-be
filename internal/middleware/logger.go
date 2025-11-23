package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// sensitiveFields contains patterns for fields that should be redacted
var sensitiveFields = []string{
	"password",
	"token",
	"api_key",
	"apikey",
	"api-key",
	"secret",
	"authorization",
	"auth",
	"bearer",
	"key",
	"credential",
	"access_token",
	"refresh_token",
	"session",
	"cookie",
}

// sensitiveHeaderPatterns contains regex patterns for sensitive headers
var sensitiveHeaderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)authorization`),
	regexp.MustCompile(`(?i)api[-_]?key`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)bearer`),
	regexp.MustCompile(`(?i)cookie`),
	regexp.MustCompile(`(?i)session`),
}

// responseWriter is a custom response writer to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerConfig holds configuration for the logger middleware
type LoggerConfig struct {
	Format string // "json" or "pretty"
	Level  string // "debug", "info", "warn", "error"
}

// RequestResponseLogger creates a middleware that logs all API requests and responses
func RequestResponseLogger(config LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Read and store request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore the body for the next handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create custom response writer to capture response
		responseBodyWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseBodyWriter

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Build log entry
		logEntry := buildLogEntry(c, requestBody, responseBodyWriter.body.Bytes(), latency)

		// Output log based on format
		if config.Format == "pretty" {
			printPrettyLog(logEntry)
		} else {
			printJSONLog(logEntry)
		}
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    string              `json:"timestamp"`
	Method       string              `json:"method"`
	Path         string              `json:"path"`
	StatusCode   int                 `json:"status_code"`
	Latency      string              `json:"latency"`
	ClientIP     string              `json:"client_ip"`
	UserAgent    string              `json:"user_agent"`
	RequestID    string              `json:"request_id,omitempty"`
	Headers      map[string]string   `json:"headers"`
	QueryParams  map[string][]string `json:"query_params,omitempty"`
	RequestBody  interface{}         `json:"request_body,omitempty"`
	ResponseBody interface{}         `json:"response_body,omitempty"`
	Error        string              `json:"error,omitempty"`
}

// buildLogEntry constructs a log entry from request and response data
func buildLogEntry(c *gin.Context, requestBody, responseBody []byte, latency time.Duration) LogEntry {
	entry := LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		Method:      c.Request.Method,
		Path:        c.Request.URL.Path,
		StatusCode:  c.Writer.Status(),
		Latency:     latency.String(),
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		Headers:     redactHeaders(c.Request.Header),
		QueryParams: c.Request.URL.Query(),
	}

	// Add request ID if available
	if requestID := c.GetString("request_id"); requestID != "" {
		entry.RequestID = requestID
	}

	// Parse and redact request body
	if len(requestBody) > 0 {
		entry.RequestBody = parseAndRedactBody(requestBody)
	}

	// Parse and redact response body
	if len(responseBody) > 0 {
		entry.ResponseBody = parseAndRedactBody(responseBody)
	}

	// Add error if present
	if len(c.Errors) > 0 {
		entry.Error = c.Errors.String()
	}

	return entry
}

// redactHeaders redacts sensitive headers
func redactHeaders(headers map[string][]string) map[string]string {
	redacted := make(map[string]string)
	for key, values := range headers {
		if isSensitiveHeader(key) {
			redacted[key] = "[REDACTED]"
		} else {
			redacted[key] = strings.Join(values, ", ")
		}
	}
	return redacted
}

// isSensitiveHeader checks if a header name is sensitive
func isSensitiveHeader(headerName string) bool {
	for _, pattern := range sensitiveHeaderPatterns {
		if pattern.MatchString(headerName) {
			return true
		}
	}
	return false
}

// parseAndRedactBody parses JSON body and redacts sensitive fields
func parseAndRedactBody(body []byte) interface{} {
	// Try to parse as JSON
	var jsonBody interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		// If not JSON, return truncated string
		bodyStr := string(body)
		if len(bodyStr) > 1000 {
			bodyStr = bodyStr[:1000] + "... (truncated)"
		}
		return bodyStr
	}

	// Redact sensitive fields
	redactSensitiveFields(jsonBody)
	return jsonBody
}

// redactSensitiveFields recursively redacts sensitive fields in JSON data
func redactSensitiveFields(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if isSensitiveField(key) {
				v[key] = "[REDACTED]"
			} else {
				redactSensitiveFields(value)
			}
		}
	case []interface{}:
		for _, item := range v {
			redactSensitiveFields(item)
		}
	}
}

// isSensitiveField checks if a field name is sensitive
func isSensitiveField(fieldName string) bool {
	lowerField := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerField, sensitive) {
			return true
		}
	}
	return false
}

// printJSONLog outputs the log entry as JSON
func printJSONLog(entry LogEntry) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf(`{"error": "failed to marshal log entry: %v"}%s`, err, "\n")
		return
	}
	fmt.Println(string(jsonBytes))
}

// printPrettyLog outputs the log entry in a human-readable format
func printPrettyLog(entry LogEntry) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("🕐 Timestamp: %s\n", entry.Timestamp)
	fmt.Printf("📍 %s %s\n", entry.Method, entry.Path)
	fmt.Printf("📊 Status: %d | ⏱️  Latency: %s\n", entry.StatusCode, entry.Latency)
	fmt.Printf("🌐 Client IP: %s\n", entry.ClientIP)

	if entry.RequestID != "" {
		fmt.Printf("🔖 Request ID: %s\n", entry.RequestID)
	}

	// Print headers
	if len(entry.Headers) > 0 {
		fmt.Println("\n📋 Headers:")
		for key, value := range entry.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// Print query params
	if len(entry.QueryParams) > 0 {
		fmt.Println("\n🔍 Query Parameters:")
		for key, values := range entry.QueryParams {
			fmt.Printf("  %s: %v\n", key, values)
		}
	}

	// Print request body
	if entry.RequestBody != nil {
		fmt.Println("\n📤 Request Body:")
		prettyPrintJSON(entry.RequestBody)
	}

	// Print response body
	if entry.ResponseBody != nil {
		fmt.Println("\n📥 Response Body:")
		prettyPrintJSON(entry.ResponseBody)
	}

	// Print error if present
	if entry.Error != "" {
		fmt.Printf("\n❌ Error: %s\n", entry.Error)
	}

	fmt.Println(strings.Repeat("=", 80))
}

// prettyPrintJSON prints JSON data in a formatted way
func prettyPrintJSON(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "  ", "  ")
	if err != nil {
		fmt.Printf("  %v\n", data)
		return
	}
	fmt.Printf("  %s\n", string(jsonBytes))
}
