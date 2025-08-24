-- Meetings table for conference rooms
CREATE TABLE meetings (
                          id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                          meeting_code VARCHAR(12) UNIQUE NOT NULL, -- Format: xxx-xxxx-xxx
                          room_name VARCHAR(255) UNIQUE NOT NULL,
                          host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          title VARCHAR(255),
                          description TEXT,
                          meeting_type VARCHAR(20) DEFAULT 'instant', -- instant, scheduled
                          scheduled_at TIMESTAMP,
                          max_participants INTEGER DEFAULT 100,
                          is_active BOOLEAN DEFAULT TRUE,
                          requires_auth BOOLEAN DEFAULT FALSE,
                          allow_guests BOOLEAN DEFAULT TRUE,
                          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                          ended_at TIMESTAMP,
                          expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours')
);

-- Meeting participants table
CREATE TABLE meeting_participants (
                                      id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                      meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
                                      user_id UUID REFERENCES users(id) ON DELETE SET NULL,
                                      guest_name VARCHAR(100), -- For non-authenticated users
                                      participant_role VARCHAR(20) DEFAULT 'participant', -- host, co-host, participant
                                      joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                      left_at TIMESTAMP,
                                      is_active BOOLEAN DEFAULT TRUE,
                                      audio_enabled BOOLEAN DEFAULT TRUE,
                                      video_enabled BOOLEAN DEFAULT TRUE,
                                      screen_sharing BOOLEAN DEFAULT FALSE,
                                      connection_quality VARCHAR(20), -- poor, fair, good, excellent
                                      UNIQUE(meeting_id, user_id)
);

-- Meeting recordings table (for future use)
CREATE TABLE meeting_recordings (
                                    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
                                    recording_url TEXT NOT NULL,
                                    recording_size BIGINT, -- in bytes
                                    duration INTEGER, -- in seconds
                                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_meetings_code ON meetings(meeting_code);
CREATE INDEX idx_meetings_host_id ON meetings(host_id);
CREATE INDEX idx_meetings_is_active ON meetings(is_active);
CREATE INDEX idx_meetings_expires_at ON meetings(expires_at);
CREATE INDEX idx_meeting_participants_meeting_id ON meeting_participants(meeting_id);
CREATE INDEX idx_meeting_participants_user_id ON meeting_participants(user_id);
CREATE INDEX idx_meeting_participants_is_active ON meeting_participants(is_active);