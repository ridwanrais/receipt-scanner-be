package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService service.AuthService
	frontendURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService service.AuthService, frontendURL string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		frontendURL: frontendURL,
	}
}

// GoogleLogin initiates the Google OAuth flow
// @Summary Initiate Google OAuth login
// @Description Redirects to Google OAuth consent screen
// @Tags auth
// @Accept json
// @Produce json
// @Success 302 "Redirect to Google OAuth"
// @Router /v1/auth/google/login [get]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	// Generate random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		respondInternalServerError(c, "Failed to generate state")
		return
	}

	// Store state in session/cookie for validation (simplified for now)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// Get OAuth URL and redirect
	url := h.authService.GetGoogleOAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles the Google OAuth callback
// @Summary Handle Google OAuth callback
// @Description Processes Google OAuth callback and returns JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "OAuth authorization code"
// @Param state query string true "OAuth state parameter"
// @Success 200 {object} service.AuthResponse "Authentication successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/auth/google/callback [get]
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Get code and state from query params
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		respondBadRequest(c, "Authorization code is required")
		return
	}

	// Validate state (CSRF protection)
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != state {
		respondBadRequest(c, "Invalid state parameter")
		return
	}

	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Handle OAuth callback
	authResponse, err := h.authService.HandleGoogleCallback(c.Request.Context(), code)
	if err != nil {
		logError(c, "google_oauth_callback_failed", err, map[string]interface{}{
			"error_type": "oauth_error",
		})
		respondInternalServerError(c, "Failed to authenticate with Google")
		return
	}

	// For web clients, redirect to frontend with tokens in URL params (or use a different flow)
	// For mobile/API clients, return JSON response

	// Check if this is an API request (Accept header or query param)
	if c.GetHeader("Accept") == "application/json" || c.Query("response_type") == "json" {
		respondOK(c, authResponse)
		return
	}

	// Redirect to frontend with tokens (for web flow)
	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s&expires_in=%d",
		h.frontendURL,
		authResponse.AccessToken,
		authResponse.RefreshToken,
		authResponse.ExpiresIn,
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// RefreshToken generates a new access token from a refresh token
// @Summary Refresh access token
// @Description Generate a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} service.TokenPair "New tokens"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 401 {object} model.ErrorResponse "Invalid refresh token"
// @Router /v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := bindJSON(c, &req); err != nil {
		respondBadRequest(c, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		respondBadRequest(c, "Refresh token is required")
		return
	}

	// Refresh tokens
	tokens, err := h.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		respondUnauthorized(c, "Invalid or expired refresh token")
		return
	}

	respondOK(c, tokens)
}

// GetCurrentUser returns the current authenticated user
// @Summary Get current user
// @Description Get the currently authenticated user's information
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.User "User information"
// @Failure 401 {object} model.ErrorResponse "Unauthorized"
// @Router /v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		respondUnauthorized(c, "User not authenticated")
		return
	}

	// Get user details
	user, err := h.authService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		respondInternalServerError(c, "Failed to get user information")
		return
	}

	respondOK(c, user)
}

// GoogleMobileAuth handles mobile authentication with Google ID Token
// @Summary Authenticate with Google ID Token (Mobile)
// @Description Authenticate mobile app users using Google Sign-In ID Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body MobileAuthRequest true "Google ID Token from mobile SDK"
// @Success 200 {object} service.AuthResponse "Authentication successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 401 {object} model.ErrorResponse "Invalid ID token"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/auth/google/mobile [post]
func (h *AuthHandler) GoogleMobileAuth(c *gin.Context) {
	var req MobileAuthRequest
	if err := bindJSON(c, &req); err != nil {
		respondBadRequest(c, "Invalid request body")
		return
	}

	if req.IDToken == "" {
		respondBadRequest(c, "ID token is required")
		return
	}

	// Handle mobile authentication
	authResponse, err := h.authService.HandleGoogleMobileAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		logError(c, "google_mobile_auth_failed", err, map[string]interface{}{
			"error_type": "mobile_auth_error",
		})
		respondUnauthorized(c, "Failed to authenticate with Google")
		return
	}

	respondOK(c, authResponse)
}

// Logout handles user logout (mainly client-side token removal)
// @Summary Logout
// @Description Logout the current user (client should remove tokens)
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Logout successful"
// @Router /v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Since we're using stateless JWT, logout is mainly client-side
	// The client should remove the tokens from storage
	respondOK(c, gin.H{
		"message": "Logout successful",
	})
}

// Register handles user registration with email and password
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} service.AuthResponse "Registration successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 409 {object} model.ErrorResponse "User already exists"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := bindJSON(c, &req); err != nil {
		respondBadRequest(c, "Invalid request body")
		return
	}

	// Register user
	authResponse, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			respondConflict(c, "User with this email already exists")
			return
		}
		logError(c, "registration_failed", err, map[string]interface{}{
			"email": req.Email,
		})
		respondInternalServerError(c, "Failed to register user")
		return
	}

	respondCreated(c, authResponse)
}

// Login handles user login with email and password
// @Summary Login with email and password
// @Description Authenticate a user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} service.AuthResponse "Login successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 401 {object} model.ErrorResponse "Invalid credentials"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := bindJSON(c, &req); err != nil {
		respondBadRequest(c, "Invalid request body")
		return
	}

	// Login user
	authResponse, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			respondUnauthorized(c, "Invalid email or password")
			return
		}
		logError(c, "login_failed", err, map[string]interface{}{
			"email": req.Email,
		})
		respondInternalServerError(c, "Failed to login")
		return
	}

	respondOK(c, authResponse)
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	auth := router.Group("/v1/auth")
	{
		// Email/Password authentication
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)

		// Web OAuth flow (for future web support)
		auth.GET("/google/login", h.GoogleLogin)
		auth.GET("/google/callback", h.GoogleCallback)

		// Mobile authentication
		auth.POST("/google/mobile", h.GoogleMobileAuth)

		// Token management
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)

		// Protected route - requires auth middleware
		auth.GET("/me", authMiddleware, h.GetCurrentUser)
	}
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// MobileAuthRequest represents a mobile authentication request
type MobileAuthRequest struct {
	IDToken string `json:"idToken" binding:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// generateRandomState generates a random state string for OAuth
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
