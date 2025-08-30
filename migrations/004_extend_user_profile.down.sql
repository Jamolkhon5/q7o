-- Remove extended user profile fields
DROP INDEX IF EXISTS idx_users_phone;
DROP INDEX IF EXISTS idx_users_location;

ALTER TABLE users 
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS date_of_birth,
DROP COLUMN IF EXISTS location,
DROP COLUMN IF EXISTS timezone;