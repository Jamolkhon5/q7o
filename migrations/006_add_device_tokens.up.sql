-- Device tokens table for push notifications
CREATE TABLE device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    device_type VARCHAR(20) NOT NULL CHECK (device_type IN ('ios', 'android')),
    push_type VARCHAR(20) NOT NULL CHECK (push_type IN ('fcm', 'apns', 'voip')),
    device_info TEXT,
    app_version VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Уникальность на комбинацию пользователь + токен + тип устройства
    UNIQUE(user_id, token, device_type)
);

-- Indexes для производительности
CREATE INDEX idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX idx_device_tokens_active ON device_tokens(is_active);
CREATE INDEX idx_device_tokens_push_type ON device_tokens(push_type);
CREATE INDEX idx_device_tokens_last_used ON device_tokens(last_used_at);

-- Comments для документации
COMMENT ON TABLE device_tokens IS 'Push notification tokens for mobile devices';
COMMENT ON COLUMN device_tokens.token IS 'FCM/APNs/VoIP push token';
COMMENT ON COLUMN device_tokens.device_type IS 'Mobile platform: ios or android';
COMMENT ON COLUMN device_tokens.push_type IS 'Push service: fcm, apns, or voip';
COMMENT ON COLUMN device_tokens.is_active IS 'Whether token is currently active';
COMMENT ON COLUMN device_tokens.last_used_at IS 'Last time token was used successfully';