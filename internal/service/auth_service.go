package service

import (
	"log"
	"sso/internal/config"
	"sso/internal/repository"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type SSOService struct {
	db           *gorm.DB
	config       config.Config
	router       *mux.Router
	userRepo     repository.UserRepository
	tokenManager *token.JWTManager
}

func NewSSOService(cfg config.Config) (*SSOService, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to DB: %w", err)
		return nil, err
	}

	//migration
	//

	service := &SSOService{
		db:     db,
		config: cfg,
		router: mux.NewRouter(),
	}
	return service, nil
}

func (s *SSOService) SetupRoutes() {

}

func (s *SSOService) Start() {
	port := s.config.ServerPort
	if port == "" {
		port = "8080"
	}

	log.Printf("SSO Service starting on port %s", port)
}
