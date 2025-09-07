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
	repo       *Repository
	config     config.PushConfig
	client     *http.Client
	firebaseV1 *FirebaseV1Service
}

func NewService(repo *Repository, cfg config.PushConfig) *Service {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	var firebaseV1 *FirebaseV1Service
	if cfg.FirebaseCredentialsPath != "" {
		fbService, err := NewFirebaseV1Service(cfg.FirebaseCredentialsPath, cfg.FirebaseProjectID)
		if err != nil {
			log.Printf("Failed to initialize Firebase V1: %v", err)
		} else {
			firebaseV1 = fbService
			log.Printf("Firebase V1 initialized successfully")
		}
	}

	return &Service{
		repo:       repo,
		config:     cfg,
		client:     client,
		firebaseV1: firebaseV1,
	}
}

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

func (s *Service) SendCallNotification(ctx context.Context, userID uuid.UUID, callData *CallPushData) error {
	tokens, err := s.repo.GetActiveTokensForUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to get user tokens: %v, userID: %s", err, userID)
		return err
	}

	if len(tokens) == 0 {
		log.Printf("WARN: No active tokens found for user: %s", userID)
		return fmt.Errorf("no active push tokens for user %s", userID)
	}

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

			if s.isInvalidTokenError(err) {
				s.repo.DeactivateToken(ctx, token.UserID, token.Token)
			}
		} else {
			successCount++
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

func (s *Service) sendFCMNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	if s.firebaseV1 != nil {
		return s.firebaseV1.SendCallNotification(ctx, token.Token, callData)
	}
	return fmt.Errorf("Firebase V1 not initialized")
}

func (s *Service) sendAPNsNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	message := APNsMessage{
		DeviceToken: token.Token,
	}

	message.Payload.APS.Alert.Title = fmt.Sprintf("Входящий %s звонок",
		map[string]string{"audio": "аудио", "video": "видео"}[callData.CallType])
	message.Payload.APS.Alert.Body = fmt.Sprintf("От %s", callData.CallerName)
	message.Payload.APS.Sound = "default"
	message.Payload.APS.ContentAvailable = 1
	message.Payload.APS.Badge = 1

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

	endpoint := "https://api.push.apple.com/3/device/" + token.Token
	if s.config.APNsSandbox {
		endpoint = "https://api.sandbox.push.apple.com/3/device/" + token.Token
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create APNs request: %w", err)
	}

	req.Header.Set("authorization", "bearer "+s.config.APNsAuthToken)
	req.Header.Set("apns-topic", s.config.APNsBundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
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

func (s *Service) sendVoIPPushNotification(ctx context.Context, token *DeviceToken, callData *CallPushData) error {
	message := VoIPPushMessage{
		DeviceToken: token.Token,
	}

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

	endpoint := "https://api.push.apple.com/3/device/" + token.Token
	if s.config.APNsSandbox {
		endpoint = "https://api.sandbox.push.apple.com/3/device/" + token.Token
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create VoIP request: %w", err)
	}

	req.Header.Set("authorization", "bearer "+s.config.APNsVoIPAuthToken)
	req.Header.Set("apns-topic", s.config.APNsVoIPBundleID)
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

func (s *Service) isInvalidTokenError(err error) bool {
	errorStr := err.Error()

	if bytes.Contains([]byte(errorStr), []byte("NotRegistered")) ||
		bytes.Contains([]byte(errorStr), []byte("InvalidRegistration")) {
		return true
	}

	if bytes.Contains([]byte(errorStr), []byte("BadDeviceToken")) ||
		bytes.Contains([]byte(errorStr), []byte("Unregistered")) {
		return true
	}

	return false
}

func (s *Service) DeactivateToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.DeactivateToken(ctx, userID, token)
}

func (s *Service) CleanupOldTokens(ctx context.Context) error {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	return s.repo.CleanupOldTokens(ctx, cutoff)
}
