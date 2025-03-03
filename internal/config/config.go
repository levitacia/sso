package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret     string
	JWTExpiration time.Duration
	DatabaseURL   string
	ServerPort    string
}

func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, err
	}

	return Config{
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTExpiration: time.Hour * 24,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		ServerPort:    os.Getenv("SERVER_PORT"),
	}, nil
}
