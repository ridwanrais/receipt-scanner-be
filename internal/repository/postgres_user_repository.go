package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridwanfathin/invoice-processor-service/internal/domain"
)

// PostgresUserRepository implements UserRepository using PostgreSQL
type PostgresUserRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{db: db}
}

// CreateUser creates a new user in the database (for OAuth users without password)
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, name, picture_url, email_verified, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.Email,
		user.Name,
		user.PictureURL,
		user.EmailVerified,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// CreateUserWithPassword creates a new user with password hash in the database
func (r *PostgresUserRepository) CreateUserWithPassword(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, name, password_hash, picture_url, email_verified, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.PictureURL,
		user.EmailVerified,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user with password: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID
func (r *PostgresUserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, email, name, picture_url, email_verified, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PictureURL,
		&user.EmailVerified,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email
func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, picture_url, email_verified, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PictureURL,
		&user.EmailVerified,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByEmailWithPassword retrieves a user by their email including password hash
func (r *PostgresUserRepository) GetUserByEmailWithPassword(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, COALESCE(password_hash, ''), picture_url, email_verified, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.PictureURL,
		&user.EmailVerified,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email with password: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user
func (r *PostgresUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET name = $1, picture_url = $2, email_verified = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.PictureURL,
		user.EmailVerified,
		user.IsActive,
		user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// CreateOAuthProvider creates a new OAuth provider record
func (r *PostgresUserRepository) CreateOAuthProvider(ctx context.Context, provider *domain.OAuthProvider) error {
	// Convert provider data to JSON
	providerDataJSON, err := json.Marshal(provider.ProviderData)
	if err != nil {
		return fmt.Errorf("failed to marshal provider data: %w", err)
	}

	query := `
		INSERT INTO oauth_providers (user_id, provider, provider_user_id, provider_email, provider_data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRow(
		ctx,
		query,
		provider.UserID,
		provider.Provider,
		provider.ProviderUserID,
		provider.ProviderEmail,
		providerDataJSON,
	).Scan(&provider.ID, &provider.CreatedAt, &provider.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}

	return nil
}

// GetOAuthProvider retrieves an OAuth provider by provider name and provider user ID
func (r *PostgresUserRepository) GetOAuthProvider(ctx context.Context, providerName, providerUserID string) (*domain.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, provider_email, provider_data, created_at, updated_at
		FROM oauth_providers
		WHERE provider = $1 AND provider_user_id = $2
	`

	provider := &domain.OAuthProvider{}
	var providerDataJSON []byte

	err := r.db.QueryRow(ctx, query, providerName, providerUserID).Scan(
		&provider.ID,
		&provider.UserID,
		&provider.Provider,
		&provider.ProviderUserID,
		&provider.ProviderEmail,
		&providerDataJSON,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	// Unmarshal provider data
	if len(providerDataJSON) > 0 {
		if err := json.Unmarshal(providerDataJSON, &provider.ProviderData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider data: %w", err)
		}
	}

	return provider, nil
}

// GetOAuthProvidersByUserID retrieves all OAuth providers for a user
func (r *PostgresUserRepository) GetOAuthProvidersByUserID(ctx context.Context, userID string) ([]domain.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, provider_email, provider_data, created_at, updated_at
		FROM oauth_providers
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth providers: %w", err)
	}
	defer rows.Close()

	var providers []domain.OAuthProvider
	for rows.Next() {
		var provider domain.OAuthProvider
		var providerDataJSON []byte

		err := rows.Scan(
			&provider.ID,
			&provider.UserID,
			&provider.Provider,
			&provider.ProviderUserID,
			&provider.ProviderEmail,
			&providerDataJSON,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OAuth provider: %w", err)
		}

		// Unmarshal provider data
		if len(providerDataJSON) > 0 {
			if err := json.Unmarshal(providerDataJSON, &provider.ProviderData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal provider data: %w", err)
			}
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// UpdateOAuthProvider updates an existing OAuth provider
func (r *PostgresUserRepository) UpdateOAuthProvider(ctx context.Context, provider *domain.OAuthProvider) error {
	// Convert provider data to JSON
	providerDataJSON, err := json.Marshal(provider.ProviderData)
	if err != nil {
		return fmt.Errorf("failed to marshal provider data: %w", err)
	}

	query := `
		UPDATE oauth_providers
		SET provider_email = $1, provider_data = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING updated_at
	`

	err = r.db.QueryRow(
		ctx,
		query,
		provider.ProviderEmail,
		providerDataJSON,
		provider.ID,
	).Scan(&provider.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update OAuth provider: %w", err)
	}

	return nil
}
