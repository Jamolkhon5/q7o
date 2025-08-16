package call

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"q7o/config"
	"q7o/internal/common/utils"
	"q7o/internal/user"
)

type Service struct {
	repo     *Repository
	userRepo *user.Repository
	livekit  *LiveKitService
	redis    *redis.Client
	cfg      config.LiveKitConfig
}

func NewService(repo *Repository, userRepo *user.Repository, cfg config.LiveKitConfig, redis *redis.Client) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		livekit:  NewLiveKitService(cfg),
		redis:    redis,
		cfg:      cfg,
	}
}

func (s *Service) GetLiveKitURL() string {
	// Для мобильных клиентов возвращаем публичный URL
	// В production это должен быть wss://your-domain.com
	if s.cfg.Host != "" {
		return s.cfg.Host
	}
	return "ws://10.0.2.2:7880" // Для Android эмулятора
}

func (s *Service) GenerateCallToken(roomName string, userID uuid.UUID, username string) (string, error) {
	return s.livekit.GenerateToken(roomName, userID, username)
}

func (s *Service) InitiateCall(ctx context.Context, callerID, calleeID uuid.UUID, callType string) (*Call, error) {
	// Проверяем существование пользователей
	caller, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return nil, errors.New("caller not found")
	}

	callee, err := s.userRepo.FindByID(ctx, calleeID)
	if err != nil {
		return nil, errors.New("callee not found")
	}

	// Проверяем доступность
	if caller.Status == "busy" {
		return nil, errors.New("caller is already in a call")
	}

	if callee.Status == "busy" {
		return nil, errors.New("user is busy")
	}

	// Генерируем уникальное имя комнаты
	roomName := utils.GenerateRoomName()

	// Создаем запись о звонке
	call := &Call{
		ID:         uuid.New(),
		RoomName:   roomName,
		CallerID:   callerID,
		CalleeID:   calleeID,
		CallerName: caller.Username,
		CalleeName: callee.Username,
		CallType:   callType,
		Status:     "initiated",
		StartedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, call); err != nil {
		return nil, err
	}

	// Сохраняем информацию о звонке в Redis для быстрого доступа
	s.storeCallInCache(ctx, call)

	// Устанавливаем таймаут на ответ (60 секунд)
	go s.handleCallTimeout(call.ID, 60*time.Second)

	// Обновляем статус caller
	s.userRepo.UpdateStatus(ctx, callerID, "calling")

	return call, nil
}

func (s *Service) AnswerCall(ctx context.Context, callID, userID uuid.UUID) (*Call, error) {
	call, err := s.repo.FindByID(ctx, callID)
	if err != nil {
		return nil, err
	}

	if call.CalleeID != userID {
		return nil, errors.New("unauthorized")
	}

	if call.Status != "initiated" && call.Status != "ringing" {
		return nil, errors.New("call already processed")
	}

	now := time.Now()
	call.Status = "answered"
	call.AnsweredAt = &now

	if err := s.repo.UpdateStatus(ctx, callID, "answered", &now, nil); err != nil {
		return nil, err
	}

	// Обновляем статусы пользователей
	s.userRepo.UpdateStatus(ctx, call.CallerID, "busy")
	s.userRepo.UpdateStatus(ctx, call.CalleeID, "busy")

	return call, nil
}

func (s *Service) RejectCall(ctx context.Context, callID, userID uuid.UUID) error {
	call, err := s.repo.FindByID(ctx, callID)
	if err != nil {
		return err
	}

	if call.CalleeID != userID {
		return errors.New("unauthorized")
	}

	if call.Status != "initiated" && call.Status != "ringing" {
		return errors.New("call already processed")
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, callID, "rejected", nil, &now); err != nil {
		return err
	}

	// Обновляем статусы пользователей
	s.userRepo.UpdateStatus(ctx, call.CallerID, "online")
	s.userRepo.UpdateStatus(ctx, call.CalleeID, "online")

	// Очищаем кеш
	s.clearCallFromCache(ctx, callID)

	return nil
}

func (s *Service) EndCall(ctx context.Context, callID, userID uuid.UUID) (*Call, error) {
	call, err := s.repo.FindByID(ctx, callID)
	if err != nil {
		return nil, err
	}

	if call.CallerID != userID && call.CalleeID != userID {
		return nil, errors.New("unauthorized")
	}

	if call.Status == "ended" {
		return call, nil
	}

	now := time.Now()

	// Вычисляем длительность если звонок был отвечен
	if call.AnsweredAt != nil {
		duration := int(now.Sub(*call.AnsweredAt).Seconds())
		call.Duration = duration
		s.repo.UpdateDuration(ctx, callID, duration)
	}

	call.Status = "ended"
	call.EndedAt = &now

	if err := s.repo.UpdateStatus(ctx, callID, "ended", nil, &now); err != nil {
		return nil, err
	}

	// Обновляем статусы пользователей
	s.userRepo.UpdateStatus(ctx, call.CallerID, "online")
	s.userRepo.UpdateStatus(ctx, call.CalleeID, "online")

	// Очищаем кеш
	s.clearCallFromCache(ctx, call.ID)

	return call, nil
}

func (s *Service) GetCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Call, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetUserCalls(ctx, userID, limit, offset)
}

func (s *Service) handleCallTimeout(callID uuid.UUID, timeout time.Duration) {
	time.Sleep(timeout)

	ctx := context.Background()
	call, err := s.repo.FindByID(ctx, callID)
	if err != nil {
		return
	}

	// Если звонок все еще не отвечен, помечаем как пропущенный
	if call.Status == "initiated" || call.Status == "ringing" {
		now := time.Now()
		s.repo.UpdateStatus(ctx, callID, "missed", nil, &now)

		// Обновляем статусы пользователей
		s.userRepo.UpdateStatus(ctx, call.CallerID, "online")
		s.userRepo.UpdateStatus(ctx, call.CalleeID, "online")

		// Очищаем кеш
		s.clearCallFromCache(ctx, callID)
	}
}

func (s *Service) storeCallInCache(ctx context.Context, call *Call) {
	key := "call:" + call.ID.String()
	s.redis.HSet(ctx, key,
		"room_name", call.RoomName,
		"caller_id", call.CallerID.String(),
		"callee_id", call.CalleeID.String(),
		"call_type", call.CallType,
		"status", call.Status,
	)
	s.redis.Expire(ctx, key, 5*time.Minute)
}

func (s *Service) clearCallFromCache(ctx context.Context, callID uuid.UUID) {
	key := "call:" + callID.String()
	s.redis.Del(ctx, key)
}

func (s *Service) getCallFromCache(ctx context.Context, callID uuid.UUID) *Call {
	key := "call:" + callID.String()
	data, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil || len(data) == 0 {
		return nil
	}

	callerID, _ := uuid.Parse(data["caller_id"])
	calleeID, _ := uuid.Parse(data["callee_id"])

	return &Call{
		ID:       callID,
		RoomName: data["room_name"],
		CallerID: callerID,
		CalleeID: calleeID,
		CallType: data["call_type"],
		Status:   data["status"],
	}
}
