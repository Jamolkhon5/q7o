-- Drop all tables (in reverse dependency order)
DROP TABLE IF EXISTS contact_requests CASCADE;
DROP TABLE IF EXISTS contacts CASCADE;
DROP TABLE IF EXISTS meeting_recordings CASCADE;
DROP TABLE IF EXISTS meeting_participants CASCADE;
DROP TABLE IF EXISTS meetings CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS calls CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop extensions (only if no other tables depend on them)
-- Note: Extensions are usually kept system-wide, but uncomment if needed
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";