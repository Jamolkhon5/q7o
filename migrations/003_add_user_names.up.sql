-- Add first_name and last_name columns to users table
ALTER TABLE users
    ADD COLUMN first_name VARCHAR(100) NOT NULL DEFAULT '',
ADD COLUMN last_name VARCHAR(100) NOT NULL DEFAULT '';

-- Remove default after adding columns
ALTER TABLE users
    ALTER COLUMN first_name DROP DEFAULT,
ALTER COLUMN last_name DROP DEFAULT;

-- Create indexes for better search performance
CREATE INDEX idx_users_first_name ON users(first_name);
CREATE INDEX idx_users_last_name ON users(last_name);
CREATE INDEX idx_users_full_name ON users(first_name, last_name);

-- Update existing users with placeholder values (optional, remove in production)
UPDATE users
SET first_name = split_part(username, '_', 1),
    last_name = COALESCE(split_part(username, '_', 2), '')
WHERE first_name = '';