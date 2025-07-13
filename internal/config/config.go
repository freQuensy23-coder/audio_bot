package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config stores the application's configuration.
type Config struct {
	TelegramToken    string
	ElevenLabsAPIKey string
	FFMPEGPath       string
	Workers          int
	QueueSize        int

	// TDLib-related config
	TDLibApiID       int
	TDLibApiHash     string
	TDLibSession     string
	TDLibForwardTo   int64
	TDLibDownloadDir string
}

// Load reads configuration from environment variables and .env file.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	workers, err := strconv.Atoi(getEnv("WORKERS", "0"))
	if err != nil {
		log.Fatalf("Invalid WORKERS value: %v", err)
	}

	queueSize, err := strconv.Atoi(getEnv("QUEUE_SIZE", "100"))
	if err != nil {
		log.Fatalf("Invalid QUEUE_SIZE value: %v", err)
	}

	tdApiID, err := strconv.Atoi(getEnv("TD_API_ID", "0"))
	if err != nil {
		log.Fatalf("Invalid TD_API_ID: %v", err)
	}
	tdForwardTo, err := strconv.ParseInt(getEnv("TD_FORWARD_USER_ID", "0"), 10, 64)
	if err != nil {
		log.Fatalf("Invalid TD_FORWARD_USER_ID: %v", err)
	}

	return &Config{
		TelegramToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		ElevenLabsAPIKey: getEnv("ELEVENLABS_API_KEY", ""),
		FFMPEGPath:       getEnv("FFMPEG_PATH", "ffmpeg"),
		Workers:          workers,
		QueueSize:        queueSize,
		TDLibApiID:       tdApiID,
		TDLibApiHash:     getEnv("TD_API_HASH", ""),
		TDLibSession:     getEnv("TD_SESSION", ""),
		TDLibForwardTo:   tdForwardTo,
		TDLibDownloadDir: getEnv("TD_DOWNLOAD_DIR", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
