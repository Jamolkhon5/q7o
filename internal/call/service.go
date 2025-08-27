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
	wsHub    *WSHub
}

func NewService(repo *Repository, userRepo *user.Repository, cfg config.LiveKitConfig, redis *redis.Client, wsHub *WSHub) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		livekit:  NewLiveKitService(cfg),
		redis:    redis,
		cfg:      cfg,
		wsHub:    wsHub,
	}
}

func (s *Service) GetLiveKitURL() string {
	if s.cfg.PublicHost != "" {
		return s.cfg.PublicHost
	}
	if s.cfg.Host != "" {
		return s.cfg.Host
	}
	return "ws://10.0.2.2:7880"
}

func (s *Service) GenerateCallToken(roomName string, userID uuid.UUID, username string) (string, error) {
	// Для обычной генерации токена используем роль "participant"
	return s.livekit.GenerateToken(roomName, userID, username, "participant")
}

func (s *Service) InitiateCall(ctx context.Context, callerID, calleeID uuid.UUID, callType string) (*Call, string, error) {
	// Проверяем существование пользователей
	caller, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return nil, "", errors.New("caller not found")
	}

	callee, err := s.userRepo.FindByID(ctx, calleeID)
	if err != nil {
		return nil, "", errors.New("callee not found")
	}

	// Проверяем доступность
	if caller.Status == "busy" {
		return nil, "", errors.New("caller is already in a call")
	}

	if callee.Status == "busy" {
		return nil, "", errors.New("user is busy")
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
		return nil, "", err
	}

	// Генерируем токен для звонящего с ролью "caller"
	callerToken, err := s.livekit.GenerateToken(roomName, callerID, caller.Username, "caller")
	if err != nil {
		return nil, "", err
	}

	// Генерируем токен для получателя с ролью "callee" и сохраняем в Redis
	calleeToken, err := s.livekit.GenerateToken(roomName, calleeID, callee.Username, "callee")
	if err != nil {
		return nil, "", err
	}

	// Сохраняем токен получателя в Redis
	tokenKey := "call:token:" + call.ID.String()
	s.redis.Set(ctx, tokenKey, calleeToken, 5*time.Minute)

	// Сохраняем информацию о звонке в Redis для быстрого доступа
	s.storeCallInCache(ctx, call)

	// Устанавливаем таймаут на ответ (60 секунд)
	go s.handleCallTimeout(call.ID, 60*time.Second)

	// Обновляем статус caller
	s.userRepo.UpdateStatus(ctx, callerID, "calling")

	// Обновляем статус звонка на "ringing" после отправки уведомления
	call.Status = "ringing"
	s.repo.UpdateStatus(ctx, call.ID, "ringing", nil, nil)

	return call, callerToken, nil
}

func (s *Service) AnswerCall(ctx context.Context, callID, userID uuid.UUID) (*Call, string, error) {
	call, err := s.repo.FindByID(ctx, callID)
	if err != nil {
		return nil, "", err
	}

	if call.CalleeID != userID {
		return nil, "", errors.New("unauthorized")
	}

	if call.Status != "initiated" && call.Status != "ringing" {
		return nil, "", errors.New("call already processed")
	}

	// Получаем сохраненный токен из Redis
	tokenKey := "call:token:" + callID.String()
	token, err := s.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		// Если токена нет в Redis, генерируем новый с ролью "callee"
		callee, _ := s.userRepo.FindByID(ctx, userID)
		token, err = s.livekit.GenerateToken(call.RoomName, userID, callee.Username, "callee")
		if err != nil {
			return nil, "", err
		}
	}

	now := time.Now()
	call.Status = "answered"
	call.AnsweredAt = &now

	if err := s.repo.UpdateStatus(ctx, callID, "answered", &now, nil); err != nil {
		return nil, "", err
	}

	// Обновляем статусы пользователей
	s.userRepo.UpdateStatus(ctx, call.CallerID, "busy")
	s.userRepo.UpdateStatus(ctx, call.CalleeID, "busy")

	// Обновляем кеш
	s.storeCallInCache(ctx, call)

	// Удаляем токен из Redis после использования
	s.redis.Del(ctx, tokenKey)

	return call, token, nil
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

	// Очищаем токен из Redis
	tokenKey := "call:token:" + callID.String()
	s.redis.Del(ctx, tokenKey)

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

	// Очищаем кеш и токен
	s.clearCallFromCache(ctx, call.ID)
	tokenKey := "call:token:" + callID.String()
	s.redis.Del(ctx, tokenKey)

	return call, nil
}

func (s *Service) GetCall(ctx context.Context, callID uuid.UUID) (*Call, error) {
	if call := s.getCallFromCache(ctx, callID); call != nil {
		return call, nil
	}
	return s.repo.FindByID(ctx, callID)
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

		// Отправляем сигнал о пропущенном звонке
		if s.wsHub != nil {
			signal := &CallSignal{
				Type:   "missed",
				FromID: call.CalleeID,
				ToID:   call.CallerID,
				CallID: call.ID.String(),
			}
			s.wsHub.broadcast <- signal
		}

		// Очищаем кеш и токен
		s.clearCallFromCache(ctx, callID)
		tokenKey := "call:token:" + callID.String()
		s.redis.Del(ctx, tokenKey)
	}
}

func (s *Service) storeCallInCache(ctx context.Context, call *Call) {
	key := "call:" + call.ID.String()
	s.redis.HSet(ctx, key,
		"room_name", call.RoomName,
		"caller_id", call.CallerID.String(),
		"callee_id", call.CalleeID.String(),
		"caller_name", call.CallerName,
		"callee_name", call.CalleeName,
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
		ID:         callID,
		RoomName:   data["room_name"],
		CallerID:   callerID,
		CalleeID:   calleeID,
		CallerName: data["caller_name"],
		CalleeName: data["callee_name"],
		CallType:   data["call_type"],
		Status:     data["status"],
	}
}
