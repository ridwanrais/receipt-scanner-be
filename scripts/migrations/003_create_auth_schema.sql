-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    picture_url TEXT,
    email_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create oauth_providers table
CREATE TABLE IF NOT EXISTS oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'github', 'facebook', etc.
    provider_user_id VARCHAR(255) NOT NULL, -- Provider's unique user ID
    provider_email VARCHAR(255), -- Email from provider (may differ from primary)
    provider_data JSONB, -- Flexible storage for provider-specific metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure one account per provider per user
    UNIQUE(user_id, provider),
    -- Ensure provider user ID is unique per provider
    UNIQUE(provider, provider_user_id)
);

-- Create indexes on oauth_providers for faster lookups
CREATE INDEX IF NOT EXISTS idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_provider ON oauth_providers(provider);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_provider_user_id ON oauth_providers(provider, provider_user_id);

-- Add user_id column to receipts table
ALTER TABLE receipts 
ADD COLUMN IF NOT EXISTS user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

-- Create index on user_id for faster user-specific queries
CREATE INDEX IF NOT EXISTS idx_receipts_user_id ON receipts(user_id);

-- Add triggers for updated_at timestamp on users
CREATE TRIGGER update_users_modtime
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

-- Add triggers for updated_at timestamp on oauth_providers
CREATE TRIGGER update_oauth_providers_modtime
BEFORE UPDATE ON oauth_providers
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

-- Add comments to explain the tables
COMMENT ON TABLE users IS 'Core user information, provider-agnostic';
COMMENT ON TABLE oauth_providers IS 'OAuth provider accounts linked to users, supports multiple providers';
COMMENT ON COLUMN oauth_providers.provider IS 'OAuth provider name (google, github, facebook, etc.)';
COMMENT ON COLUMN oauth_providers.provider_user_id IS 'Unique user ID from the OAuth provider';
COMMENT ON COLUMN oauth_providers.provider_data IS 'Flexible JSONB storage for provider-specific metadata';
COMMENT ON COLUMN receipts.user_id IS 'User who owns this receipt';
