package main

import (
	"log"

	"yourproject/internal/config"
	"yourproject/internal/service"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Инициализация сервиса
	ssoService, err := service.NewSSOService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSO service: %v", err)
	}

	// Запуск сервера
	if err := ssoService.Start(); err != nil {
		log.Fatalf("Failed to start SSO service: %v", err)
	}
}