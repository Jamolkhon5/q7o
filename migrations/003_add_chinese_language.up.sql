-- Добавление поддержки китайского языка в настройки пользователя

-- Удаляем старое ограничение
ALTER TABLE user_settings
DROP CONSTRAINT IF EXISTS user_settings_language_check;

-- Добавляем новое ограничение с поддержкой китайского языка
ALTER TABLE user_settings
    ADD CONSTRAINT user_settings_language_check
        CHECK (language IN ('en', 'ru', 'zh'));

-- Обновляем комментарий
COMMENT ON COLUMN user_settings.language IS 'User interface language preference: en (English), ru (Russian), zh (Chinese)';