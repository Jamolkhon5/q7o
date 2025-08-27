package call

import (
	"fmt"
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

// GenerateToken теперь принимает роль участника для создания уникального identity
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

	// Создаем уникальный identity для каждого участника
	// Формат: userID::role::roomName - это гарантирует уникальность
	identity := fmt.Sprintf("%s::%s::%s", userID.String(), participantRole, roomName)

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(username).
		SetValidFor(24 * time.Hour)

	return at.ToJWT()
}

func (s *LiveKitService) CreateRoom(roomName string) error {
	// Room auto-creates on first participant join with our config
	return nil
}
