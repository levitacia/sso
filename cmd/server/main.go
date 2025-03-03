package main

import (
	"log"

	"sso/internal/config"
	"sso/internal/service"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ssoService, err := service.NewSSOService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	if err := ssoService.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
