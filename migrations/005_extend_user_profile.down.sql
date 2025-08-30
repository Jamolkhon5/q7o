-- Drop extended profile fields
ALTER TABLE users 
DROP COLUMN IF EXISTS phone,
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS date_of_birth,
DROP COLUMN IF EXISTS location,
DROP COLUMN IF EXISTS timezone;

-- Drop indexes
DROP INDEX CONCURRENTLY IF EXISTS idx_users_phone;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_location;

-- Drop avatar URL field
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;