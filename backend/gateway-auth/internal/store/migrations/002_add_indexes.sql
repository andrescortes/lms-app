-- Add index for refresh token cleanup queries
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Add index for user updated_at (used in password change)
CREATE INDEX IF NOT EXISTS idx_users_updated_at ON users(updated_at);