-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                       username VARCHAR(50) UNIQUE NOT NULL,
                       email VARCHAR(255) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL,
                       email_verified BOOLEAN DEFAULT FALSE,
                       email_verification_code VARCHAR(6),
                       email_verification_expires TIMESTAMP,
                       avatar_url TEXT,
                       status VARCHAR(20) DEFAULT 'offline', -- online, offline, busy, away
                       last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Calls table
CREATE TABLE calls (
                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                       room_name VARCHAR(255) NOT NULL,
                       caller_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                       callee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                       call_type VARCHAR(20) NOT NULL, -- audio, video
                       status VARCHAR(20) NOT NULL, -- initiated, ringing, answered, ended, missed, rejected
                       started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       answered_at TIMESTAMP,
                       ended_at TIMESTAMP,
                       duration INTEGER DEFAULT 0, -- in seconds
                       recording_url TEXT,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table (for refresh tokens)
CREATE TABLE sessions (
                          id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                          user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          refresh_token TEXT NOT NULL,
                          device_info TEXT,
                          ip_address VARCHAR(45),
                          expires_at TIMESTAMP NOT NULL,
                          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_calls_caller_id ON calls(caller_id);
CREATE INDEX idx_calls_callee_id ON calls(callee_id);
CREATE INDEX idx_calls_status ON calls(status);
CREATE INDEX idx_calls_created_at ON calls(created_at DESC);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();