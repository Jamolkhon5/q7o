-- User settings table
CREATE TABLE user_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notifications_call BOOLEAN DEFAULT TRUE,
    notifications_meeting BOOLEAN DEFAULT TRUE,
    notifications_chat BOOLEAN DEFAULT TRUE,
    privacy VARCHAR(20) DEFAULT 'friends' CHECK (privacy IN ('public', 'friends', 'private')),
    theme VARCHAR(20) DEFAULT 'auto' CHECK (theme IN ('light', 'dark', 'auto')),
    language VARCHAR(5) DEFAULT 'en' CHECK (language IN ('en', 'ru')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- Index for user settings
CREATE INDEX idx_user_settings_user_id ON user_settings(user_id);

-- Trigger for updated_at
CREATE TRIGGER update_user_settings_updated_at BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE user_settings IS 'User application settings and preferences';
COMMENT ON COLUMN user_settings.notifications_call IS 'Enable/disable call notifications';
COMMENT ON COLUMN user_settings.notifications_meeting IS 'Enable/disable meeting notifications';
COMMENT ON COLUMN user_settings.notifications_chat IS 'Enable/disable chat notifications';
COMMENT ON COLUMN user_settings.privacy IS 'User privacy setting: public, friends, private';
COMMENT ON COLUMN user_settings.theme IS 'UI theme preference: light, dark, auto';
COMMENT ON COLUMN user_settings.language IS 'User interface language preference';