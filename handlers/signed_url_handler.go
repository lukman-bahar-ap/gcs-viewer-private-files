package handlers

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gcs-viewer/utils"

	"github.com/gofiber/fiber/v2"
)

func SignedURLHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		bucketName := c.Query("bucket")
		objectName := c.Query("object")
		expiryStr := c.Query("expiry")

		if bucketName == "" {
			bucketName = os.Getenv("BUCKET_NAME")
		}
		if objectName == "" {
			objectName = os.Getenv("FILE_NAME")
		}

		if bucketName == "" || objectName == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Bucket name or object name is missing")
		}

		// Default expiry logic:
		// 1. Start with 15 minutes hardcoded default
		expiryDuration := 15 * time.Minute

		// 2. Override with ENV if present
		if envExpiry := os.Getenv("SIGNED_URL_EXPIRY"); envExpiry != "" {
			if d, err := time.ParseDuration(envExpiry); err == nil {
				expiryDuration = d
			} else {
				// Fallback to integer seconds if duration parse fails
				if s, err := strconv.Atoi(envExpiry); err == nil {
					expiryDuration = time.Duration(s) * time.Second
				}
			}
		}

		// 3. Override with Query Param if present
		if expiryStr != "" {
			// Try parsing as duration string (e.g. "30m", "1h")
			if d, err := time.ParseDuration(expiryStr); err == nil {
				expiryDuration = d
			} else {
				// Fallback: try parsing as simple integer seconds
				if s, err := strconv.Atoi(expiryStr); err == nil {
					expiryDuration = time.Duration(s) * time.Second
				}
			}
		}

		url, expiresAt, err := utils.GenerateSignedURL(c.Context(), bucketName, objectName, expiryDuration)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to generate signed URL: %v", err))
		}

		return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
			"url":        url,
			"expires_at": expiresAt,
			"status":     "success",
			"code":       200,
		})
	}
}
