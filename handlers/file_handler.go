package handlers

import (
	"context"
	"fmt"
	"gcs-viewer/utils"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"github.com/gofiber/fiber/v2"
)

func ViewFileHandler(client *storage.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		bucketName := os.Getenv("BUCKET_NAME")
		fileName := os.Getenv("FILE_NAME")

		if bucketName == "" || fileName == "" {
			return c.Status(fiber.StatusInternalServerError).SendString("Bucket name or file name is not set in environment variables")
		}

		// Use the client passed from main
		rc, err := client.Bucket(bucketName).Object(fileName).NewReader(context.Background())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to read file from GCS: " + err.Error())
		}
		defer rc.Close()

		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
		contentType, _ := utils.GetContentType(fileName)
		c.Set("Content-Type", contentType)

		if _, err := io.Copy(c.Response().BodyWriter(), rc); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to send file content: " + err.Error())
		}

		return nil
	}
}
