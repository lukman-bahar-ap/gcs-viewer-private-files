package handlers

import (
	"context"
	"fmt"
	"gcs-viewer/utils"
	"io"
	"os"

	"time"

	"cloud.google.com/go/storage"
	"github.com/gofiber/fiber/v2"
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

	// Cleanup intermediates
	if err == nil {
		for _, obj := range intermediates {
			_ = client.Bucket(bucketName).Object(obj).Delete(ctx)
		}
	}

	return err
}

func MergeHandler(client *storage.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req MergeRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if len(req.Sources) == 0 || req.Dest == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Sources and destination must be provided")
		}

		// Use the client passed from main
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Increased timeout for merge
		defer cancel()

		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			return c.Status(fiber.StatusInternalServerError).SendString("Bucket name is not set in environment variables")
		}

		// Use a unique request ID (or just a random string) for intermediates to avoid collision
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())
		err := recursiveCompose(ctx, client, bucketName, req.Sources, req.Dest, requestID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to merge files: " + err.Error())
		}

		// Open the merged file
		rc, err := client.Bucket(bucketName).Object(req.Dest).NewReader(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to read merged file: " + err.Error())
		}
		defer rc.Close()

		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", req.Dest))
		contentType, _ := utils.GetContentType(req.Dest)
		c.Set("Content-Type", contentType)

		if _, err := io.Copy(c.Response().BodyWriter(), rc); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to send file content: " + err.Error())
		}

		return nil
	}
}
