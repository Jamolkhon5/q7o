package push

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"q7o/config"
)

type Service struct {
	repo   *Repository
	config config.PushConfig
	client *http.Client
}

func NewService(repo *Repository, cfg config.PushConfig) *Service {
	// Настраиваем HTTP клиент с таймаутами
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	return &Service{
		repo:   repo,
		config: cfg,
		client: client,
	}
}

// RegisterDeviceToken регистрирует токен устройства
func (s *Service) RegisterDeviceToken(ctx context.Context, userID uuid.UUID, req *RegisterTokenRequest) error {
	token := &DeviceToken{
		ID:         uuid.New(),
		UserID:     userID,
		Token:      req.Token,
		DeviceType: req.DeviceType,
		PushType:   req.PushType,
		DeviceInfo: req.DeviceInfo,
		AppVersion: req.AppVersion,
		IsActive:   true,
		LastUsedAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	return s.repo.RegisterDeviceToken(ctx, token)
}

// SendCallNotification отправляет push уведомление о входящем звонке
func (s *Service) SendCallNotification(ctx context.Context, userID uuid.UUID, callData *CallPushData) error {
	// Получаем активные токены пользователя
	tokens, err := s.repo.GetActiveTokensForUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to get user tokens: %v, userID: %s", err, userID)
		return err
	}

	if len(tokens) == 0 {
		log.Printf("WARN: No active tokens found for user: %s", userID)
		return fmt.Errorf("no active push tokens for user %s", userID)
	}

	// Отправляем на все активные устройства
	var lastError error
	successCount := 0

	for _, token := range tokens {
		var err error

		switch token.PushType {
		case "fcm":
			err = s.sendFCMNotification(ctx, token, callData)
		case "apns":
			err = s.sendAPNsNotification(ctx, token, callData)
		case "voip":
			err = s.sendVoIPPushNotification(ctx, token, callData)
		default:
			log.Printf("WARN: Unknown push type: %s", token.PushType)
			continue
		}

		if err != nil {
			log.Printf("ERROR: Failed to send push notification: %v, tokenID: %s, pushType: %s",
				err, token.ID, token.PushType)
			lastError = err

			// Деактивируем токен если он недействителен
			if s.isInvalidTokenError(err) {
				s.repo.DeactivateToken(ctx, token.UserID, token.Token)
			}
		} else {
			successCount++
			// Обновляем время использования успешного токена
			s.repo.UpdateTokenUsage(ctx, token.Token)
		}
	}

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to send push notifications: %w", lastError)
	}

	log.Printf("INFO: Push notifications sent - userID: %s, successCount: %d, totalTokens: %d",
		userID, successCount, len(tokens))

	return nil
}

// sendFCMNotification отправляет FCM уведомление (Android)
func (s *Service) sendFCMNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	message := FCMMessage{
		To: token.Token,
		Data: struct {
			Type       string `json:"type"`
			CallID     string `json:"call_id"`
			CallerID   string `json:"caller_id"`
			CallerName string `json:"caller_name"`
			CallType   string `json:"call_type"`
			RoomName   string `json:"room_name"`
			Token      string `json:"token,omitempty"`
		}{
			Type:       "call_incoming",
			CallID:     callData.CallID,
			CallerID:   callData.CallerID,
			CallerName: callData.CallerName,
			CallType:   callData.CallType,
			RoomName:   callData.RoomName,
			Token:      callData.Token,
		},
		Priority:         "high", // Высокий приоритет для звонков
		ContentAvailable: true,   // Пробуждает приложение в фоне
		TimeToLive:       60,     // 1 минута на доставку
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create FCM request: %w", err)
	}

	req.Header.Set("Authorization", "key="+s.config.FCMServerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send FCM request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FCM request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	log.Printf("INFO: FCM notification sent successfully, tokenID: %s", token.ID)
	return nil
}

// sendAPNsNotification отправляет APNs уведомление (iOS)
func (s *Service) sendAPNsNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	message := APNsMessage{
		DeviceToken: token.Token,
	}

	// Настраиваем payload для обычного push (не VoIP)
	message.Payload.APS.Alert.Title = fmt.Sprintf("Входящий %s звонок",
		map[string]string{"audio": "аудио", "video": "видео"}[callData.CallType])
	message.Payload.APS.Alert.Body = fmt.Sprintf("От %s", callData.CallerName)
	message.Payload.APS.Sound = "default"
	message.Payload.APS.ContentAvailable = 1
	message.Payload.APS.Badge = 1

	// Добавляем данные звонка
	message.Payload.CallID = callData.CallID
	message.Payload.CallerID = callData.CallerID
	message.Payload.CallerName = callData.CallerName
	message.Payload.CallType = callData.CallType
	message.Payload.RoomName = callData.RoomName
	message.Payload.Token = callData.Token

	jsonData, err := json.Marshal(message.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal APNs message: %w", err)
	}

	// Используем HTTP/2 APNs endpoint
	endpoint := "https://api.push.apple.com/3/device/" + token.Token
	if s.config.APNsSandbox {
		endpoint = "https://api.sandbox.push.apple.com/3/device/" + token.Token
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create APNs request: %w", err)
	}

	// Настраиваем заголовки APNs
	req.Header.Set("authorization", "bearer "+s.config.APNsAuthToken)
	req.Header.Set("apns-topic", s.config.APNsBundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10") // Высокий приоритет
	req.Header.Set("content-type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send APNs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("APNs request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	log.Printf("INFO: APNs notification sent successfully, tokenID: %s", token.ID)
	return nil
}

// sendVoIPPushNotification отправляет VoIP push уведомление (iOS)
func (s *Service) sendVoIPPushNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	message := VoIPPushMessage{
		DeviceToken: token.Token,
	}

	// VoIP push содержит только данные, без alert
	message.Payload.CallID = callData.CallID
	message.Payload.CallerID = callData.CallerID
	message.Payload.CallerName = callData.CallerName
	message.Payload.CallType = callData.CallType
	message.Payload.RoomName = callData.RoomName
	message.Payload.Token = callData.Token

	jsonData, err := json.Marshal(message.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal VoIP message: %w", err)
	}

	// VoIP push endpoint
	endpoint := "https://api.push.apple.com/3/device/" + token.Token
	if s.config.APNsSandbox {
		endpoint = "https://api.sandbox.push.apple.com/3/device/" + token.Token
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create VoIP request: %w", err)
	}

	// VoIP push заголовки
	req.Header.Set("authorization", "bearer "+s.config.APNsVoIPAuthToken)
	req.Header.Set("apns-topic", s.config.APNsVoIPBundleID) // Обычно .voip суффикс
	req.Header.Set("apns-push-type", "voip")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("content-type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send VoIP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("VoIP request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	log.Printf("INFO: VoIP push notification sent successfully, tokenID: %s", token.ID)
	return nil
}

// isInvalidTokenError проверяет, является ли ошибка признаком недействительного токена
func (s *Service) isInvalidTokenError(err error) bool {
	errorStr := err.Error()

	// FCM invalid token indicators
	if bytes.Contains([]byte(errorStr), []byte("NotRegistered")) ||
		bytes.Contains([]byte(errorStr), []byte("InvalidRegistration")) {
		return true
	}

	// APNs invalid token indicators
	if bytes.Contains([]byte(errorStr), []byte("BadDeviceToken")) ||
		bytes.Contains([]byte(errorStr), []byte("Unregistered")) {
		return true
	}

	return false
}

// DeactivateToken деактивирует токен устройства
func (s *Service) DeactivateToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.DeactivateToken(ctx, userID, token)
}

// CleanupOldTokens очищает старые токены
func (s *Service) CleanupOldTokens(ctx context.Context) error {
	cutoff := time.Now().Add(-30 * 24 * time.Hour) // 30 дней назад
	return s.repo.CleanupOldTokens(ctx, cutoff)
}
