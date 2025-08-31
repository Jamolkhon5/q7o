package contact

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"q7o/internal/call"
	"q7o/internal/user"
)

type Service struct {
	repo     *Repository
	userRepo *user.Repository
	wsHub    *call.WSHub
}

func NewService(repo *Repository, userRepo *user.Repository, wsHub *call.WSHub) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		wsHub:    wsHub,
	}
}

func (s *Service) SendContactRequest(ctx context.Context, senderID uuid.UUID, dto *SendContactRequestDTO) (*ContactRequest, error) {
	receiverID, err := uuid.Parse(dto.ReceiverID)
	if err != nil {
		return nil, errors.New("invalid receiver ID")
	}

	// Check if trying to add self
	if senderID == receiverID {
		return nil, errors.New("cannot add yourself as contact")
	}

	// Check if receiver exists
	receiver, err := s.userRepo.FindByID(ctx, receiverID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if already contacts
	isContact, _ := s.repo.IsContact(ctx, senderID, receiverID)
	if isContact {
		return nil, errors.New("already in contacts")
	}

	// Очищаем старые отклоненные/принятые запросы перед проверкой
	s.repo.CleanupOldRequests(ctx, senderID, receiverID)

	// Check for existing pending request (in either direction)
	hasRequest, _ := s.repo.HasExistingRequest(ctx, senderID, receiverID)
	if hasRequest {
		return nil, errors.New("request already exists")
	}

	// Create request
	request := &ContactRequest{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Message:    dto.Message,
	}

	if err := s.repo.CreateContactRequest(ctx, request); err != nil {
		return nil, err
	}

	// Send WebSocket notification
	sender, _ := s.userRepo.FindByID(ctx, senderID)
	if s.wsHub != nil {
		notificationData, _ := json.Marshal(map[string]interface{}{
			"request_id":    request.ID.String(),
			"sender_id":     senderID.String(),
			"sender_name":   sender.Username,
			"receiver_name": receiver.Username, // ИСПОЛЬЗУЕМ receiver
			"message":       dto.Message,
		})

		signal := &call.CallSignal{
			Type:   "contact_request_received",
			FromID: senderID,
			ToID:   receiverID,
			Data:   notificationData,
		}
		s.wsHub.Broadcast() <- signal
	}

	return request, nil
}

func (s *Service) AcceptContactRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	request, err := s.repo.GetContactRequest(ctx, requestID)
	if err != nil {
		return errors.New("request not found")
	}

	// Verify the user is the receiver
	if request.ReceiverID != userID {
		return errors.New("unauthorized")
	}

	if request.Status != "pending" {
		return errors.New("request already processed")
	}

	// Create contact relationship
	if err := s.repo.CreateContact(ctx, request.SenderID, request.ReceiverID); err != nil {
		return err
	}

	// Удаляем запрос после принятия (для последовательности с отклонением)
	if err := s.repo.DeleteContactRequest(ctx, requestID); err != nil {
		return err
	}

	// Send WebSocket notification to sender
	if s.wsHub != nil {
		receiver, _ := s.userRepo.FindByID(ctx, userID)
		notificationData, _ := json.Marshal(map[string]interface{}{
			"request_id":    requestID.String(),
			"receiver_id":   userID.String(),
			"receiver_name": receiver.Username,
		})

		signal := &call.CallSignal{
			Type:   "contact_request_accepted",
			FromID: userID,
			ToID:   request.SenderID,
			Data:   notificationData,
		}
		s.wsHub.Broadcast() <- signal
	}

	return nil
}

func (s *Service) RejectContactRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	request, err := s.repo.GetContactRequest(ctx, requestID)
	if err != nil {
		return errors.New("request not found")
	}

	// Verify the user is the receiver
	if request.ReceiverID != userID {
		return errors.New("unauthorized")
	}

	if request.Status != "pending" {
		return errors.New("request already processed")
	}

	// Удаляем запрос вместо изменения статуса - это позволит отправлять повторные запросы
	if err := s.repo.DeleteContactRequest(ctx, requestID); err != nil {
		return err
	}

	// Send WebSocket notification to sender
	if s.wsHub != nil {
		notificationData, _ := json.Marshal(map[string]interface{}{
			"request_id": requestID.String(),
		})

		signal := &call.CallSignal{
			Type:   "contact_request_rejected",
			FromID: userID,
			ToID:   request.SenderID,
			Data:   notificationData,
		}
		s.wsHub.Broadcast() <- signal
	}

	return nil
}

func (s *Service) RemoveContact(ctx context.Context, userID, contactID uuid.UUID) error {
	// Check if they are contacts
	isContact, _ := s.repo.IsContact(ctx, userID, contactID)
	if !isContact {
		return errors.New("not in contacts")
	}

	// Delete contact relationship
	if err := s.repo.DeleteContact(ctx, userID, contactID); err != nil {
		return err
	}

	// Send WebSocket notification
	if s.wsHub != nil {
		notificationData, _ := json.Marshal(map[string]interface{}{
			"removed_by": userID.String(),
		})

		signal := &call.CallSignal{
			Type:   "contact_removed",
			FromID: userID,
			ToID:   contactID,
			Data:   notificationData,
		}
		s.wsHub.Broadcast() <- signal
	}

	return nil
}

func (s *Service) GetContacts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetContacts(ctx, userID, limit, offset)
}

func (s *Service) GetIncomingRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactRequestWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetIncomingRequests(ctx, userID, limit, offset)
}

func (s *Service) GetOutgoingRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ContactRequestWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetOutgoingRequests(ctx, userID, limit, offset)
}

func (s *Service) IsContact(ctx context.Context, userID, contactID uuid.UUID) (bool, error) {
	return s.repo.IsContact(ctx, userID, contactID)
}

func (s *Service) UpdateLastCallTime(ctx context.Context, userID, contactID uuid.UUID) error {
	// Check if they are contacts before updating
	isContact, _ := s.repo.IsContact(ctx, userID, contactID)
	if !isContact {
		return nil // Silently ignore if not contacts
	}
	return s.repo.UpdateLastCallTime(ctx, userID, contactID)
}
