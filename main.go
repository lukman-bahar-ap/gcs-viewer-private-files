package main

import (
	"context"
	"fmt"
	"gcs-viewer/handlers"
	"gcs-viewer/utils"
	"net/http"
	"os"

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
	defer client.Close() // hanya sekali saat server mati

	hostname := os.Getenv("HOST")
	http.HandleFunc("/view-file", handlers.ViewFileHandler(client))
	http.HandleFunc("/merge", handlers.MergeHandler(client))
	fmt.Println("Server running at ", hostname, ":8080")

	http.ListenAndServe(":8080", nil)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

}
