package domain

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	PictureURL    string    `json:"pictureUrl,omitempty"`
	EmailVerified bool      `json:"emailVerified"`
	IsActive      bool      `json:"isActive"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// OAuthProvider represents an OAuth provider linked to a user
type OAuthProvider struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"userId"`
	Provider       string                 `json:"provider"` // 'google', 'github', etc.
	ProviderUserID string                 `json:"providerUserId"`
	ProviderEmail  string                 `json:"providerEmail,omitempty"`
	ProviderData   map[string]interface{} `json:"providerData,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// GoogleUserInfo represents user information from Google OAuth
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
