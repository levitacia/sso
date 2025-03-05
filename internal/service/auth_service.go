package service

import (
	"log"
	"net/http"
	"sso/internal/config"
	"sso/internal/middleware"
	"sso/internal/models"
	"sso/internal/repository"
	"sso/pkg/token"

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

	if err = db.AutoMigrate(&models.User{}); err != nil {
		log.Printf("Failed to migrate DB: %w", err)
		return nil, err
	}

	userRepo := repository.NewUserRepository(db)
	tokenManager := token.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)

	service := &SSOService{
		db:           db,
		config:       cfg,
		router:       mux.NewRouter(),
		userRepo:     userRepo,
		tokenManager: tokenManager,
	}

	service.SetupRoutes()

	return service, nil
}

func (s *SSOService) SetupRoutes() {
	authHandler := handlers.NewAuthHandler(s.userRepo, s.tokenManager)

	profileHandler := handlers.NewProfileHandler(s.userRepo)

	authMiddleware := middleware.NewAuthMiddleware(s.tokenManager)

	s.router.HandleFunc("/api/register", authHandler.Register).Methods("POST")
	s.router.HandleFunc("/api/login", authHandler.Login).Methods("POST")
	s.router.HandleFunc("/api/refresh", authHandler.RefreshToken).Methods("POST")
	s.router.HandleFunc("/api/verify", authHandler.VerifyToken).Methods("GET")

	protected := s.router.PathPrefix("/api/protected").Subrouter()
	protected.Use(authMiddleware.Authenticate)
	protected.HandleFunc("/profile", profileHandler.GetProfile).Methods("GET")
}

func (s *SSOService) Start() error {
	port := s.config.ServerPort
	if port == "" {
		port = "8080"
	}

	log.Printf("SSO Service starting on port %s", port)
	return http.ListenAndServe("localhost:"+port, s.router)
}
