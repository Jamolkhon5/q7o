package call

import (
	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"q7o/config"
	"time"
)

type LiveKitService struct {
	cfg config.LiveKitConfig
}

func NewLiveKitService(cfg config.LiveKitConfig) *LiveKitService {
	return &LiveKitService{
		cfg: cfg,
	}
}

// GenerateToken генерирует токен для участника звонка
func (s *LiveKitService) GenerateToken(roomName string, userID uuid.UUID, username string, participantRole string) (string, error) {
	at := auth.NewAccessToken(s.cfg.APIKey, s.cfg.APISecret)

	canPublish := true
	canSubscribe := true
	canPublishData := true

	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           roomName,
		CanPublish:     &canPublish,
		CanSubscribe:   &canSubscribe,
		CanPublishData: &canPublishData,
	}

	// ИСПРАВЛЕНИЕ: Используем простой identity как в meetings - только userID
	// Роль сохраняем в metadata
	identity := userID.String()

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(username).
		SetMetadata(participantRole). // Сохраняем роль в metadata
		SetValidFor(24 * time.Hour)

	return at.ToJWT()
}

func (s *LiveKitService) CreateRoom(roomName string) error {
	// Room auto-creates on first participant join with our config
	return nil
}
