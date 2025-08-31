package contact

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Contact methods
func (r *Repository) CreateContact(ctx context.Context, userID, contactID uuid.UUID) error {
	// Create bidirectional contact relationship
	query := `
		INSERT INTO contacts (user_id, contact_id, created_at) 
		VALUES 
			($1, $2, $3),
			($2, $1, $3)
		ON CONFLICT (user_id, contact_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, userID, contactID, time.Now())
	return err
}

func (r *Repository) DeleteContact(ctx context.Context, userID, contactID uuid.UUID) error {
	// Delete bidirectional relationship
	query := `
		DELETE FROM contacts 
		WHERE (user_id = $1 AND contact_id = $2) 
		   OR (user_id = $2 AND contact_id = $1)
	`
	_, err := r.db.ExecContext(ctx, query, userID, contactID)
	return err
}

func (r *Repository) IsContact(ctx context.Context, userID, contactID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM contacts 
			WHERE user_id = $1 AND contact_id = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID, contactID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetContacts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactWithUser, error) {
	query := `
		SELECT 
			c.id, c.contact_id, u.username, u.first_name, u.last_name, 
			u.email, u.avatar_url, u.status, c.last_call_at, c.created_at
		FROM contacts c
		INNER JOIN users u ON u.id = c.contact_id
		WHERE c.user_id = $1
		ORDER BY c.last_call_at DESC NULLS LAST, c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*ContactWithUser
	for rows.Next() {
		contact := &ContactWithUser{}
		err := rows.Scan(
			&contact.ID, &contact.ContactID, &contact.Username,
			&contact.FirstName, &contact.LastName, &contact.Email,
			&contact.AvatarURL, &contact.Status, &contact.LastCallAt,
			&contact.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (r *Repository) UpdateLastCallTime(ctx context.Context, userID, contactID uuid.UUID) error {
	query := `
		UPDATE contacts 
		SET last_call_at = $3
		WHERE (user_id = $1 AND contact_id = $2) 
		   OR (user_id = $2 AND contact_id = $1)
	`
	_, err := r.db.ExecContext(ctx, query, userID, contactID, time.Now())
	return err
}

// Contact Request methods
func (r *Repository) CreateContactRequest(ctx context.Context, req *ContactRequest) error {
	query := `
		INSERT INTO contact_requests (
			id, sender_id, receiver_id, status, message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	req.ID = uuid.New()
	req.Status = "pending"
	req.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.SenderID, req.ReceiverID,
		req.Status, req.Message, req.CreatedAt,
	)
	return err
}

func (r *Repository) GetContactRequest(ctx context.Context, requestID uuid.UUID) (*ContactRequest, error) {
	query := `
		SELECT id, sender_id, receiver_id, status, message, created_at, responded_at
		FROM contact_requests
		WHERE id = $1
	`

	req := &ContactRequest{}
	err := r.db.QueryRowContext(ctx, query, requestID).Scan(
		&req.ID, &req.SenderID, &req.ReceiverID,
		&req.Status, &req.Message, &req.CreatedAt, &req.RespondedAt,
	)
	return req, err
}

func (r *Repository) GetPendingRequest(ctx context.Context, senderID, receiverID uuid.UUID) (*ContactRequest, error) {
	query := `
		SELECT id, sender_id, receiver_id, status, message, created_at, responded_at
		FROM contact_requests
		WHERE sender_id = $1 AND receiver_id = $2 AND status = 'pending'
	`

	req := &ContactRequest{}
	err := r.db.QueryRowContext(ctx, query, senderID, receiverID).Scan(
		&req.ID, &req.SenderID, &req.ReceiverID,
		&req.Status, &req.Message, &req.CreatedAt, &req.RespondedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return req, err
}

func (r *Repository) UpdateRequestStatus(ctx context.Context, requestID uuid.UUID, status string) error {
	query := `
		UPDATE contact_requests 
		SET status = $2, responded_at = $3
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, requestID, status, time.Now())
	return err
}

func (r *Repository) GetIncomingRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactRequestWithUser, error) {
	query := `
		SELECT 
			cr.id, cr.sender_id, cr.receiver_id, u.username, u.first_name, 
			u.last_name, u.avatar_url, cr.status, cr.message, cr.created_at, cr.responded_at
		FROM contact_requests cr
		INNER JOIN users u ON u.id = cr.sender_id
		WHERE cr.receiver_id = $1 AND cr.status = 'pending'
		ORDER BY cr.created_at DESC
		LIMIT $2 OFFSET $3
	`

	return r.getRequestsWithUsers(ctx, query, userID, limit, offset, "incoming")
}

func (r *Repository) GetOutgoingRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactRequestWithUser, error) {
	query := `
		SELECT 
			cr.id, cr.sender_id, cr.receiver_id, u.username, u.first_name, 
			u.last_name, u.avatar_url, cr.status, cr.message, cr.created_at, cr.responded_at
		FROM contact_requests cr
		INNER JOIN users u ON u.id = cr.receiver_id
		WHERE cr.sender_id = $1 AND cr.status = 'pending'
		ORDER BY cr.created_at DESC
		LIMIT $2 OFFSET $3
	`

	return r.getRequestsWithUsers(ctx, query, userID, limit, offset, "outgoing")
}

func (r *Repository) getRequestsWithUsers(ctx context.Context, query string, userID uuid.UUID, limit, offset int, requestType string) ([]*ContactRequestWithUser, error) {
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ContactRequestWithUser
	for rows.Next() {
		req := &ContactRequestWithUser{RequestType: requestType}
		err := rows.Scan(
			&req.ID, &req.SenderID, &req.ReceiverID, &req.Username,
			&req.FirstName, &req.LastName, &req.AvatarURL,
			&req.Status, &req.Message, &req.CreatedAt, &req.RespondedAt,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, nil
}

func (r *Repository) HasExistingRequest(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM contact_requests 
			WHERE ((sender_id = $1 AND receiver_id = $2) 
			    OR (sender_id = $2 AND receiver_id = $1))
			  AND status = 'pending'
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID1, userID2).Scan(&exists)
	return exists, err
}

func (r *Repository) DeleteContactRequest(ctx context.Context, requestID uuid.UUID) error {
	query := `DELETE FROM contact_requests WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, requestID)
	return err
}
