-- Очистка старых отклоненных и принятых запросов контактов
-- чтобы разрешить повторную отправку запросов

DELETE FROM contact_requests 
WHERE status IN ('rejected', 'accepted');

-- Добавляем индекс для быстрого поиска pending запросов
CREATE INDEX IF NOT EXISTS idx_contact_requests_pending 
ON contact_requests(sender_id, receiver_id, status) 
WHERE status = 'pending';