package main

import (
	"fmt"
	"gcs-viewer/handlers"
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
	hostname := os.Getenv("HOST")
	http.HandleFunc("/view-file", handlers.ViewFileHandler)
	fmt.Println("Server running at ",hostname,":8080")

	http.ListenAndServe(":8080", nil)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

}
