DROP INDEX IF EXISTS idx_contact_requests_created;
DROP INDEX IF EXISTS idx_contact_requests_receiver;
DROP INDEX IF EXISTS idx_contact_requests_sender;
DROP INDEX IF EXISTS idx_contacts_last_call;
DROP INDEX IF EXISTS idx_contacts_contact_id;
DROP INDEX IF EXISTS idx_contacts_user_id;

DROP TABLE IF EXISTS contact_requests;
DROP TABLE IF EXISTS contacts;