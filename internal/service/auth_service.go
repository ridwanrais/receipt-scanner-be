package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
	"github.com/ridwanfathin/invoice-processor-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Common errors
var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

// AuthService handles authentication operations
type AuthService interface {
	// OAuth operations
	GetGoogleOAuthURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (*AuthResponse, error)

	// Mobile authentication
	HandleGoogleMobileAuth(ctx context.Context, idToken string) (*AuthResponse, error)

	// Email/Password authentication
	Register(ctx context.Context, email, password, name string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)

	// JWT operations
	GenerateTokens(userID string) (*TokenPair, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	RefreshAccessToken(refreshToken string) (*TokenPair, error)

	// User operations
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}

// AuthResponse contains authentication response data
type AuthResponse struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"` // seconds
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// authService implements AuthService
type authService struct {
	userRepo              repository.UserRepository
	googleOAuthConfig     *oauth2.Config
	googleClientIDAndroid string
	googleClientIDIOS     string
	jwtSecret             []byte
	jwtAccessExpiration   time.Duration
	jwtRefreshExpiration  time.Duration
}

// AuthServiceConfig holds configuration for auth service
type AuthServiceConfig struct {
	UserRepo              repository.UserRepository
	GoogleClientID        string
	GoogleClientSecret    string
	GoogleRedirectURL     string
	GoogleClientIDAndroid string
	GoogleClientIDIOS     string
	JWTSecret             string
	JWTAccessExpiration   time.Duration
	JWTRefreshExpiration  time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(config AuthServiceConfig) AuthService {
	googleOAuthConfig := &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &authService{
		userRepo:              config.UserRepo,
		googleOAuthConfig:     googleOAuthConfig,
		googleClientIDAndroid: config.GoogleClientIDAndroid,
		googleClientIDIOS:     config.GoogleClientIDIOS,
		jwtSecret:             []byte(config.JWTSecret),
		jwtAccessExpiration:   config.JWTAccessExpiration,
		jwtRefreshExpiration:  config.JWTRefreshExpiration,
	}
}

// GetGoogleOAuthURL generates the Google OAuth URL
func (s *authService) GetGoogleOAuthURL(state string) string {
	return s.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleGoogleCallback processes the Google OAuth callback
func (s *authService) HandleGoogleCallback(ctx context.Context, code string) (*AuthResponse, error) {
	// Exchange code for token
	token, err := s.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	googleUser, err := s.getGoogleUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if OAuth provider exists
	oauthProvider, err := s.userRepo.GetOAuthProvider(ctx, "google", googleUser.ID)

	var user *domain.User

	if err != nil {
		// New user - create user and OAuth provider
		user = &domain.User{
			Email:         googleUser.Email,
			Name:          googleUser.Name,
			PictureURL:    googleUser.Picture,
			EmailVerified: googleUser.VerifiedEmail,
			IsActive:      true,
		}

		if err := s.userRepo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Create OAuth provider record
		provider := &domain.OAuthProvider{
			UserID:         user.ID,
			Provider:       "google",
			ProviderUserID: googleUser.ID,
			ProviderEmail:  googleUser.Email,
			ProviderData: map[string]interface{}{
				"locale":      googleUser.Locale,
				"given_name":  googleUser.GivenName,
				"family_name": googleUser.FamilyName,
			},
		}

		if err := s.userRepo.CreateOAuthProvider(ctx, provider); err != nil {
			return nil, fmt.Errorf("failed to create OAuth provider: %w", err)
		}
	} else {
		// Existing user - get user details
		user, err = s.userRepo.GetUserByID(ctx, oauthProvider.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Update user info from Google if changed
		if user.Name != googleUser.Name || user.PictureURL != googleUser.Picture {
			user.Name = googleUser.Name
			user.PictureURL = googleUser.Picture
			user.EmailVerified = googleUser.VerifiedEmail

			if err := s.userRepo.UpdateUser(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to update user: %w", err)
			}
		}
	}

	// Generate JWT tokens
	tokens, err := s.GenerateTokens(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// getGoogleUserInfo retrieves user information from Google
func (s *authService) getGoogleUserInfo(ctx context.Context, accessToken string) (*domain.GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"https://www.googleapis.com/oauth2/v2/userinfo",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo domain.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// HandleGoogleMobileAuth handles mobile authentication with Google ID Token
func (s *authService) HandleGoogleMobileAuth(ctx context.Context, idToken string) (*AuthResponse, error) {
	// Verify the ID token with Google
	googleUser, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Check if OAuth provider exists
	oauthProvider, err := s.userRepo.GetOAuthProvider(ctx, "google", googleUser.ID)

	var user *domain.User

	if err != nil {
		// New user - create user and OAuth provider
		user = &domain.User{
			Email:         googleUser.Email,
			Name:          googleUser.Name,
			PictureURL:    googleUser.Picture,
			EmailVerified: googleUser.VerifiedEmail,
			IsActive:      true,
		}

		if err := s.userRepo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Create OAuth provider record
		provider := &domain.OAuthProvider{
			UserID:         user.ID,
			Provider:       "google",
			ProviderUserID: googleUser.ID,
			ProviderEmail:  googleUser.Email,
			ProviderData: map[string]interface{}{
				"locale":      googleUser.Locale,
				"given_name":  googleUser.GivenName,
				"family_name": googleUser.FamilyName,
			},
		}

		if err := s.userRepo.CreateOAuthProvider(ctx, provider); err != nil {
			return nil, fmt.Errorf("failed to create OAuth provider: %w", err)
		}
	} else {
		// Existing user - get user details
		user, err = s.userRepo.GetUserByID(ctx, oauthProvider.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Update user info from Google if changed
		if user.Name != googleUser.Name || user.PictureURL != googleUser.Picture {
			user.Name = googleUser.Name
			user.PictureURL = googleUser.Picture
			user.EmailVerified = googleUser.VerifiedEmail

			if err := s.userRepo.UpdateUser(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to update user: %w", err)
			}
		}
	}

	// Generate JWT tokens
	tokens, err := s.GenerateTokens(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// Register creates a new user with email and password
func (s *authService) Register(ctx context.Context, email, password, name string) (*AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create new user
	user := &domain.User{
		Email:         email,
		Name:          name,
		PasswordHash:  string(hashedPassword),
		EmailVerified: false,
		IsActive:      true,
	}

	if err := s.userRepo.CreateUserWithPassword(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT tokens
	tokens, err := s.GenerateTokens(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// Login authenticates a user with email and password
func (s *authService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	// Get user by email with password hash
	user, err := s.userRepo.GetUserByEmailWithPassword(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user has a password (might be OAuth-only user)
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT tokens
	tokens, err := s.GenerateTokens(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// verifyGoogleIDToken verifies a Google ID token and returns user info
func (s *authService) verifyGoogleIDToken(ctx context.Context, idToken string) (*domain.GoogleUserInfo, error) {
	// Call Google's tokeninfo endpoint to verify the token
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid ID token: %s", string(body))
	}

	// Parse the token info response
	var tokenInfo struct {
		Sub           string `json:"sub"` // User ID
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"` // "true" or "false" as string
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
		Aud           string `json:"aud"` // Client ID
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, err
	}

	// Verify the audience (client ID) matches one of our mobile clients
	validAudience := tokenInfo.Aud == s.googleClientIDAndroid || tokenInfo.Aud == s.googleClientIDIOS
	if !validAudience {
		return nil, fmt.Errorf("invalid audience: expected Android or iOS client ID, got %s", tokenInfo.Aud)
	}

	// Convert to GoogleUserInfo
	emailVerified := tokenInfo.EmailVerified == "true"

	return &domain.GoogleUserInfo{
		ID:            tokenInfo.Sub,
		Email:         tokenInfo.Email,
		VerifiedEmail: emailVerified,
		Name:          tokenInfo.Name,
		GivenName:     tokenInfo.GivenName,
		FamilyName:    tokenInfo.FamilyName,
		Picture:       tokenInfo.Picture,
		Locale:        tokenInfo.Locale,
	}, nil
}

// GenerateTokens generates access and refresh tokens
func (s *authService) GenerateTokens(userID string) (*TokenPair, error) {
	// Get user to include email in claims
	user, err := s.userRepo.GetUserByID(context.Background(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate access token
	accessClaims := &Claims{
		UserID: userID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtAccessExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &Claims{
		UserID: userID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtRefreshExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.jwtAccessExpiration.Seconds()),
	}, nil
}

// ValidateAccessToken validates and parses an access token
func (s *authService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshAccessToken generates a new access token from a refresh token
func (s *authService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := s.ValidateAccessToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new token pair
	return s.GenerateTokens(claims.UserID)
}

// GetUserByID retrieves a user by ID
func (s *authService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}
