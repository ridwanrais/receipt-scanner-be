-- Add password_hash column to users table for email/password authentication
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- Add comment to explain the column
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password for email/password authentication. NULL for OAuth-only users.';
