package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"gcs-viewer/utils"
	"io"
	"os"

	"net/http"

	"cloud.google.com/go/storage"
)

type MergeRequest struct {
	Sources []string `json:"sources"`
	Dest    string   `json:"dest"`
}

type MergeResponse struct {
	Message string `json:"message"`
}

func composeFiles(ctx context.Context, client *storage.Client, bucketName string, sources []string, dest string) error {
	if len(sources) > 32 {
		return fmt.Errorf("compose supports max 32 objects, got %d", len(sources))
	}

	var objs []*storage.ObjectHandle
	for _, src := range sources {
		objs = append(objs, client.Bucket(bucketName).Object(src))
	}

	destObj := client.Bucket(bucketName).Object(dest)
	composer := destObj.ComposerFrom(objs...)
	if _, err := composer.Run(ctx); err != nil {
		return fmt.Errorf("failed to compose: %w", err)
	}
	return nil
}

func recursiveCompose(ctx context.Context, client *storage.Client, bucketName string, sources []string, finalDest string, requestID string) error {
	const batchSize = 32
	var intermediates []string

	if len(sources) <= batchSize {
		if err := composeFiles(ctx, client, bucketName, sources, finalDest); err != nil {
			return nil
		}
		return nil
	}

	for i := 0; i < len(sources); i += batchSize {
		end := i + batchSize
		if end > len(sources) {
			end = len(sources)
		}
		batch := sources[i:end]
		intermediateName := fmt.Sprintf("tmp/%s/intermediate_%d", requestID, i/batchSize)

		if err := composeFiles(ctx, client, bucketName, batch, intermediateName); err != nil {
			return err
		}
		intermediates = append(intermediates, intermediateName)
	}

	var err error
	if len(intermediates) > 32 {
		err = recursiveCompose(ctx, client, bucketName, intermediates, finalDest, requestID)
	} else {
		err = composeFiles(ctx, client, bucketName, intermediates, finalDest)
	}

	// Cleanup intermediates kalau sukses
	if err == nil {
		for _, obj := range intermediates {
			_ = client.Bucket(bucketName).Object(obj).Delete(ctx)
		}
	}

	return err
}

func MergeHandler(client *storage.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MergeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if len(req.Sources) == 0 {
			http.Error(w, "sources cannot be empty", http.StatusBadRequest)
			return
		}

		bucketName := os.Getenv("BUCKET_NAME")
		ctx := r.Context()

		err := recursiveCompose(ctx, client, bucketName, req.Sources, req.Dest, "some-request-id") // TODO: Generate a proper request ID
		if err != nil {
			http.Error(w, fmt.Sprintf("merge failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Buka hasil merge
		rc, err := client.Bucket(bucketName).Object(req.Dest).NewReader(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read merged file: %v", err), http.StatusInternalServerError)
			return
		}
		defer rc.Close()

		// Set header supaya browser langsung download
		// w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", req.Dest))

		contentType, found := utils.GetContentType(req.Dest)
		if !found {
			http.Error(w, "Unsupported file type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", contentType)

		if _, err := io.Copy(w, rc); err != nil {
			http.Error(w, "Failed to send file content: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
