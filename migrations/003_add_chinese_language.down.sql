-- Откат: возвращаем ограничение только для английского и русского языков

-- Сначала устанавливаем всем пользователям с китайским языком английский по умолчанию
UPDATE user_settings SET language = 'en' WHERE language = 'zh';

-- Удаляем текущее ограничение
ALTER TABLE user_settings
DROP CONSTRAINT IF EXISTS user_settings_language_check;

-- Возвращаем старое ограничение без китайского языка
ALTER TABLE user_settings
    ADD CONSTRAINT user_settings_language_check
        CHECK (language IN ('en', 'ru'));

-- Возвращаем старый комментарий
COMMENT ON COLUMN user_settings.language IS 'User interface language preference';