package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "Unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "Unauthorized",
				"message": "Invalid authorization header format. Expected 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "Unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user information in context for handlers to use
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)

		// Continue to next handler
		c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that validates JWT tokens but doesn't require them
// Useful for endpoints that work differently for authenticated vs unauthenticated users
func OptionalAuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without setting user context
			c.Next()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without setting user context
			c.Next()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			// Invalid token, continue without setting user context
			c.Next()
			return
		}

		// Set user information in context
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)

		c.Next()
	}
}
