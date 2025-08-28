-- Create contacts table
CREATE TABLE IF NOT EXISTS contacts (
                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_call_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, contact_id),
    CHECK (user_id != contact_id)
    );

-- Create contact_requests table
CREATE TABLE IF NOT EXISTS contact_requests (
                                                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP,
    UNIQUE(sender_id, receiver_id),
    CHECK (sender_id != receiver_id)
    );

-- Indexes for contacts
CREATE INDEX idx_contacts_user_id ON contacts(user_id);
CREATE INDEX idx_contacts_contact_id ON contacts(contact_id);
CREATE INDEX idx_contacts_last_call ON contacts(user_id, last_call_at DESC NULLS LAST);

-- Indexes for contact_requests
CREATE INDEX idx_contact_requests_sender ON contact_requests(sender_id, status);
CREATE INDEX idx_contact_requests_receiver ON contact_requests(receiver_id, status);
CREATE INDEX idx_contact_requests_created ON contact_requests(created_at DESC);