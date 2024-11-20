package main

import (
	"fmt"
	"gcs-viewer/handlers"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	http.HandleFunc("/view-file", handlers.ViewFileHandler)
	fmt.Println("Server running at http://localhost:8080")

	http.ListenAndServe(":8080", nil)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

}
