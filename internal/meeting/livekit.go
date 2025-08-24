package meeting

import (
	"time"

	"github.com/livekit/protocol/auth"
	"q7o/config"
)

type LiveKitService struct {
	cfg config.LiveKitConfig
}

func NewLiveKitService(cfg config.LiveKitConfig) *LiveKitService {
	return &LiveKitService{
		cfg: cfg,
	}
}

// GenerateMeetingToken generates a token for meeting participants
func (s *LiveKitService) GenerateMeetingToken(roomName, identity, name, role string) (string, error) {
	at := auth.NewAccessToken(s.cfg.APIKey, s.cfg.APISecret)

	canPublish := true
	canSubscribe := true
	canPublishData := true

	// Add additional permissions for host
	roomAdmin := false
	if role == "host" || role == "co-host" {
		roomAdmin = true
	}

	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           roomName,
		CanPublish:     &canPublish,
		CanSubscribe:   &canSubscribe,
		CanPublishData: &canPublishData,
		RoomAdmin:      roomAdmin,
	}

	// Set metadata for role
	metadata := map[string]string{
		"role": role,
	}

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(name).
		SetMetadata(string(metadata["role"])).
		SetValidFor(24 * time.Hour)

	return at.ToJWT()
}
