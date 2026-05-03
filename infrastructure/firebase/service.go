package firebase

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"google.golang.org/api/option"
)

func NewFCMClient(ctx context.Context) (*messaging.Client, error) {
	var opts []option.ClientOption

	firebaseCredFile := "firebase-dev.json"
	if domain.IsProdEnv() {
		firebaseCredFile = "/secrets/firebase-prod.json"
	}

	if _, err := os.Stat(firebaseCredFile); err == nil {
		opts = append(opts, option.WithCredentialsFile(firebaseCredFile))
	} else {
		fmt.Printf("Warning: Firebase credentials file '%s' not found. Ensure it exists in the root directory.\n", firebaseCredFile)
	}

	firebaseApp, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to init Firebase App: %w", err)
	}

	messagingClient, err := firebaseApp.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init Firebase messaging: %w", err)
	}

	return messagingClient, nil
}
