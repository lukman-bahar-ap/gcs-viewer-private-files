package utils

import (
	"context"
	"log"

	"cloud.google.com/go/storage"
)

func GcsNewClient(ctx context.Context) *storage.Client {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create GCS client: %v", err)
	}
	return client
}