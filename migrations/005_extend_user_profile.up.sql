-- Add avatar URL field
ALTER TABLE users ADD COLUMN avatar_url TEXT;

-- Extend user profile with additional fields
ALTER TABLE users 
ADD COLUMN phone VARCHAR(20) UNIQUE,
ADD COLUMN bio TEXT,
ADD COLUMN date_of_birth DATE,
ADD COLUMN location VARCHAR(100),
ADD COLUMN timezone VARCHAR(50);

-- Create indexes for better performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_location ON users(location) WHERE location IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN users.phone IS 'User phone number in international format';
COMMENT ON COLUMN users.bio IS 'User biography/description (max 500 chars)';
COMMENT ON COLUMN users.date_of_birth IS 'User date of birth';
COMMENT ON COLUMN users.location IS 'User location/city';
COMMENT ON COLUMN users.timezone IS 'User timezone (IANA format)';