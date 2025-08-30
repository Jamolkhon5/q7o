-- Add extended user profile fields
ALTER TABLE users 
ADD COLUMN phone VARCHAR(20) UNIQUE,
ADD COLUMN bio TEXT,
ADD COLUMN date_of_birth DATE,
ADD COLUMN location VARCHAR(100),
ADD COLUMN timezone VARCHAR(50);

-- Create indexes for better query performance
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_location ON users(location) WHERE location IS NOT NULL;

-- Add constraints
ALTER TABLE users ADD CONSTRAINT check_bio_length CHECK (length(bio) <= 500);
ALTER TABLE users ADD CONSTRAINT check_location_length CHECK (length(location) <= 100);
ALTER TABLE users ADD CONSTRAINT check_timezone_length CHECK (length(timezone) <= 50);

-- Update the updated_at trigger to include new fields
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';