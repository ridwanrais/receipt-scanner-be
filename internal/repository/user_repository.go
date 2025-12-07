package repository

import (
	"context"

	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *domain.User) error
	CreateUserWithPassword(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByEmailWithPassword(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error

	// OAuth provider operations
	CreateOAuthProvider(ctx context.Context, provider *domain.OAuthProvider) error
	GetOAuthProvider(ctx context.Context, providerName, providerUserID string) (*domain.OAuthProvider, error)
	GetOAuthProvidersByUserID(ctx context.Context, userID string) ([]domain.OAuthProvider, error)
	UpdateOAuthProvider(ctx context.Context, provider *domain.OAuthProvider) error
}
