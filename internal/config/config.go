package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret             string
	JWTExpiration         time.Duration
	DatabaseURL           string
	RedisURL              string
	ServerPort            string
	GoogleCredentialsPath string
	GoogleDriveLogFolder  string
	LogExportInterval     time.Duration
}

func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, err
	}

	logExportInterval := time.Hour
	if val := os.Getenv("LOG_EXPORT_INTERVAL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			logExportInterval = duration
		}
	}

	return Config{
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTExpiration:         time.Hour * 24,
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		RedisURL:              os.Getenv("REDIS_URL"),
		ServerPort:            os.Getenv("SERVER_PORT"),
		GoogleCredentialsPath: os.Getenv("GOOGLE_CREDENTIALS_PATH"),
		GoogleDriveLogFolder:  os.Getenv("GOOGLE_DRIVE_LOG_FOLDER"),
		LogExportInterval:     logExportInterval,
	}, nil
}
