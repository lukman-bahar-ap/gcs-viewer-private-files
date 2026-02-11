package main

import (
	"context"
	"fmt"
	"gcs-viewer/handlers"
	"gcs-viewer/utils"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}
	ctx := context.Background()
	client := utils.GcsNewClient(ctx)
	defer client.Close()

	hostname := os.Getenv("HOST")

	// Initialize Fiber app
	app := fiber.New()

	// Enable CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Content-Type",
	}))

	// Register Routes
	app.Get("/view-file", handlers.ViewFileHandler(client))
	app.Post("/merge", handlers.MergeHandler(client))
	app.Get("/signed-url", handlers.SignedURLHandler())

	// Serve Static Files
	app.Static("/", "./index.html")

	// Graceful Shutdown
	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine
	go func() {
		fmt.Println("Server running at ", hostname, ":8080")
		if err := app.Listen(":8080"); err != nil {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	fmt.Println("\nShutting down server...")

	// Shutdown the app
	if err := app.Shutdown(); err != nil {
		fmt.Printf("Server Shutdown: %v\n", err)
	}
	fmt.Println("Server exiting")
}
