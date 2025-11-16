package handler

import (
	"fmt"
	"mime/multipart"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/model"
)

// getPathParam retrieves a path parameter and validates it's not empty
func getPathParam(c *gin.Context, paramName string) (string, error) {
	value := c.Param(paramName)
	if value == "" {
		return "", fmt.Errorf("%s is required", paramName)
	}
	return value, nil
}

// getQueryInt retrieves an integer query parameter with a default value
func getQueryInt(c *gin.Context, paramName string, defaultValue int) (int, error) {
	valueStr := c.Query(paramName)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be an integer", paramName)
	}

	return value, nil
}

// getQueryString retrieves a string query parameter
func getQueryString(c *gin.Context, paramName string) string {
	return c.Query(paramName)
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: expected YYYY-MM-DD")
	}

	return date, nil
}

// getFormFile retrieves a file from multipart form data
func getFormFile(c *gin.Context, fieldName string) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := c.Request.FormFile(fieldName)
	if err != nil {
		return nil, nil, fmt.Errorf("no %s provided", fieldName)
	}
	return file, header, nil
}

// bindJSON binds JSON request body to a struct
func bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return fmt.Errorf("invalid JSON format: %v", err)
	}
	return nil
}

// validatePagination validates and returns pagination parameters
func validatePagination(page, limit int) error {
	if page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	return nil
}

// buildValidationErrors converts validation errors to ErrorDetail slice
func buildValidationErrors(errors map[string]string) []model.ErrorDetail {
	details := make([]model.ErrorDetail, 0, len(errors))
	for field, message := range errors {
		details = append(details, model.ErrorDetail{
			Field:   field,
			Message: message,
		})
	}
	return details
}
