package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/model"
)

// HTTP status codes as constants for consistency
const (
	StatusOK                  = http.StatusOK
	StatusCreated             = http.StatusCreated
	StatusNoContent           = http.StatusNoContent
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusNotFound            = http.StatusNotFound
	StatusConflict            = http.StatusConflict
	StatusUnprocessableEntity = http.StatusUnprocessableEntity
	StatusInternalServerError = http.StatusInternalServerError
)

// Common error messages
const (
	ErrInvalidInput       = "Invalid input format"
	ErrInvalidID          = "Invalid ID provided"
	ErrResourceNotFound   = "Resource not found"
	ErrInternalServer     = "Internal server error"
	ErrInvalidQueryParams = "Invalid query parameters"
	ErrFileUpload         = "Failed to upload file"
	ErrFileProcessing     = "Failed to process file"
	ErrDataExtraction     = "Unable to extract data"
)

// respondWithError sends a standardized error response
func respondWithError(c *gin.Context, statusCode int, message string, details ...model.ErrorDetail) {
	response := model.ErrorResponse{
		Status:  http.StatusText(statusCode),
		Message: message,
		Details: details,
	}
	c.JSON(statusCode, response)
}

// respondBadRequest sends a 400 Bad Request response
func respondBadRequest(c *gin.Context, message string, details ...model.ErrorDetail) {
	respondWithError(c, StatusBadRequest, message, details...)
}

// respondUnauthorized sends a 401 Unauthorized response
func respondUnauthorized(c *gin.Context, message string, details ...model.ErrorDetail) {
	respondWithError(c, StatusUnauthorized, message, details...)
}

// respondNotFound sends a 404 Not Found response
func respondNotFound(c *gin.Context, message string) {
	respondWithError(c, StatusNotFound, message)
}

// respondConflict sends a 409 Conflict response
func respondConflict(c *gin.Context, message string) {
	respondWithError(c, StatusConflict, message)
}

// respondUnprocessableEntity sends a 422 Unprocessable Entity response
func respondUnprocessableEntity(c *gin.Context, message string, details ...model.ErrorDetail) {
	respondWithError(c, StatusUnprocessableEntity, message, details...)
}

// respondInternalServerError sends a 500 Internal Server Error response
func respondInternalServerError(c *gin.Context, message string) {
	respondWithError(c, StatusInternalServerError, message)
}

// respondSuccess sends a standardized success response with data
func respondSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// respondCreated sends a 201 Created response with data
func respondCreated(c *gin.Context, data interface{}) {
	respondSuccess(c, StatusCreated, data)
}

// respondOK sends a 200 OK response with data
func respondOK(c *gin.Context, data interface{}) {
	respondSuccess(c, StatusOK, data)
}

// respondNoContent sends a 204 No Content response
func respondNoContent(c *gin.Context) {
	c.Status(StatusNoContent)
}

// newErrorDetail creates a new error detail
func newErrorDetail(field, message string) model.ErrorDetail {
	return model.ErrorDetail{
		Field:   field,
		Message: message,
	}
}

// newErrorDetails creates multiple error details
func newErrorDetails(errors map[string]string) []model.ErrorDetail {
	details := make([]model.ErrorDetail, 0, len(errors))
	for field, message := range errors {
		details = append(details, newErrorDetail(field, message))
	}
	return details
}
