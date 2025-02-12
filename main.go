package main

import (
	"fmt"
	"net/http"
	"os"

	"shuto-api/config"
	"shuto-api/handler"
	"shuto-api/utils"

	_ "shuto-api/docs"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title						shuto API
// @description			API for processing and transforming images
// @BasePath				/v2
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and API key.
func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if err := utils.InitLogger(logLevel); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	utils.Info("Initializing VIPS library")
	vips.LoggingSettings(func(messageDomain string, messageLevel vips.LogLevel, message string) {
		switch messageLevel {
		case vips.LogLevelError:
			utils.Error("VIPS error", "vips_message", message)
		case vips.LogLevelWarning:
			utils.Warn("VIPS warning", "vips_message", message)
		case vips.LogLevelInfo:
			utils.Info("VIPS info", "vips_message", message)
		}
	}, vips.LogLevelWarning)

	vips.Startup(nil)
	defer vips.Shutdown()

	if err := godotenv.Load(); err != nil {
		utils.Warn("No .env file found, using environment variables")
	} else {
		utils.Debug("Loaded environment variables from .env file")
	}

	utils.Info("Initializing services")
	imageUtils := utils.NewImageUtils()
	executor := utils.NewCommandExecutor()
	configManager := config.NewDomainConfigManager(&config.FileConfigLoader{}, "config/domains.yaml")
	rclone := utils.NewRclone(executor, configManager)

	// Wrap handlers with CORS middleware
	http.HandleFunc("/"+config.ApiVersion+"/image/", utils.CORSMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handler.ImageHandler(w, r, imageUtils, rclone, configManager)
	}))
	http.HandleFunc("/"+config.ApiVersion+"/list/", utils.CORSMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handler.ListHandler(w, r, imageUtils, rclone, configManager)
	}))
	http.HandleFunc("/"+config.ApiVersion+"/download/", utils.CORSMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handler.DownloadHandler(w, r, imageUtils, rclone, configManager)
	}))

	// Serve Swagger UI with CORS
	http.HandleFunc("/docs/", utils.CORSMiddleware(httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		utils.Debug("No PORT environment variable found, using default", "port", "8080")
	}

	utils.Info("Starting server", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		utils.Fatal("Server failed to start", "error", err)
	}
}
