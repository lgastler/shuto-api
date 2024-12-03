package main

import (
	"log"
	"net/http"
	"os"

	"shuto-api/handler"
	"shuto-api/utils"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize the Vips library
	vips.Startup(nil)
	defer vips.Shutdown()

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize services and utilities
	imageUtils := utils.NewImageUtils()
	executor := utils.NewCommandExecutor()
	rclone := utils.NewRclone(executor)

	// Create handler functions
	imageHandler := func(w http.ResponseWriter, r *http.Request) {
		handler.ImageHandler(w, r, imageUtils, rclone)
	}

	listHandler := func(w http.ResponseWriter, r *http.Request) {
		handler.ListHandler(w, r, rclone)
	}

	// Register routes
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/list/", listHandler)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}