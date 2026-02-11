package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"cloud.google.com/go/storage"
)

func GcsNewClient(ctx context.Context) *storage.Client {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create GCS client: %v", err)
	}
	return client
}

type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func GetGCSCredentials() (string, []byte, error) {
	credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credFile == "" {
		return "", nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is not set")
	}

	data, err := os.ReadFile(credFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read credential file: %w", err)
	}

	var key ServiceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return "", nil, fmt.Errorf("failed to parse credential file: %w", err)
	}

	return key.ClientEmail, []byte(key.PrivateKey), nil
}

func GenerateSignedURL(ctx context.Context, bucket, object string, expiry time.Duration) (string, time.Time, error) {
	// 1. Try Local Mode (GOOGLE_APPLICATION_CREDENTIALS)
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		clientEmail, privateKey, err := GetGCSCredentials()
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to get local credentials: %w", err)
		}

		expiresAt := time.Now().Add(expiry)
		url, err := storage.SignedURL(bucket, object, &storage.SignedURLOptions{
			GoogleAccessID: clientEmail,
			PrivateKey:     privateKey,
			Method:         "GET",
			Expires:        expiresAt,
		})
		return url, expiresAt, err
	}

	// 2. Try Cloud Run Mode (Metadata + IAM)
	// Get Service Account Email from Metadata Server
	email, err := metadata.Email("default")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get default service account email from metadata: %w", err)
	}

	// Create IAM Credentials Client
	c, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create iam credentials client: %w", err)
	}
	defer c.Close()

	// Define SignBytes function using IAM SignBlob
	signBytes := func(b []byte) ([]byte, error) {
		req := &credentialspb.SignBlobRequest{
			Name:    fmt.Sprintf("projects/-/serviceAccounts/%s", email),
			Payload: b,
		}
		resp, err := c.SignBlob(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.SignedBlob, nil
	}

	// Generate Signed URL
	expiresAt := time.Now().Add(expiry)
	url, err := storage.SignedURL(bucket, object, &storage.SignedURLOptions{
		GoogleAccessID: email,
		SignBytes:      signBytes,
		Method:         "GET",
		Expires:        expiresAt,
	})
	return url, expiresAt, err
}
