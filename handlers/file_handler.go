package handlers

import (
	"context"
	"gcs-viewer/utils"
	"io"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
)

func ViewFileHandler(client *storage.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := os.Getenv("BUCKET_NAME")
		fileName := os.Getenv("FILE_NAME")

		if bucketName == "" || fileName == "" {
			http.Error(w, "Bucket name or file name is not set in environment variables", http.StatusInternalServerError)
			return
		}

		ctx := context.Background()
		rc, err := client.Bucket(bucketName).Object(fileName).NewReader(ctx)
		if err != nil {
			http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rc.Close()

		contentType, found := utils.GetContentType(fileName)
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
