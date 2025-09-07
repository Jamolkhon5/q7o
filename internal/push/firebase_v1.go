package push

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebaseV1Service struct {
	client    *messaging.Client
	projectID string
}

func NewFirebaseV1Service(credentialsPath string, projectID string) (*FirebaseV1Service, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Messaging client: %v", err)
	}

	return &FirebaseV1Service{
		client:    client,
		projectID: projectID,
	}, nil
}

func (s *FirebaseV1Service) SendCallNotification(ctx context.Context, token string, callData *CallPushData) error {
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"type":        "call_incoming",
			"call_id":     callData.CallID,
			"caller_id":   callData.CallerID,
			"caller_name": callData.CallerName,
			"call_type":   callData.CallType,
			"room_name":   callData.RoomName,
			"token":       callData.Token,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Title:     fmt.Sprintf("Входящий %s звонок", callData.CallType),
				Body:      fmt.Sprintf("От %s", callData.CallerName),
				ChannelID: "calls",
				Priority:  messaging.PriorityMax,
			},
		},
	}

	response, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	log.Printf("Successfully sent message: %s", response)
	return nil
}
