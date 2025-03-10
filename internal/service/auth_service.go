package service

import (
	"log"
	"net/http"
	"sso/internal/config"
	"sso/internal/handlers"
	"sso/internal/middleware"
	"sso/internal/models"
	"sso/internal/repository"
	"sso/pkg/token"

	gohandlers "github.com/gorilla/handlers"
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

	router := mux.NewRouter()

	service := &SSOService{
		db:           db,
		config:       cfg,
		router:       router,
		userRepo:     userRepo,
		tokenManager: tokenManager,
	}

	service.SetupRoutes()

	return service, nil
}

func (s *SSOService) SetupRoutes() {
	corsHandler := gohandlers.CORS(
		gohandlers.AllowedOrigins([]string{"*"}),
		gohandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gohandlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Origin", "Authorization"}),
		gohandlers.AllowCredentials(),
	)

	authHandler := handlers.NewAuthHandler(s.userRepo, s.tokenManager)

	profileHandler := handlers.NewProfileHandler(s.userRepo)

	authMiddleware := middleware.NewAuthMiddleware(s.tokenManager)

	s.router.HandleFunc("/api/register", authHandler.Register).Methods("POST")
	s.router.HandleFunc("/api/login", authHandler.Login).Methods("POST")
	s.router.HandleFunc("/api/refresh", authHandler.RefreshToken).Methods("POST")
	s.router.HandleFunc("/api/verify", authHandler.VerifyToken).Methods("GET")

	// CORS issue
	protected := s.router.PathPrefix("/api/protected").Subrouter()
	protected.Use(corsHandler, authMiddleware.Authenticate)
	protected.HandleFunc("/profile", profileHandler.GetProfile).Methods("GET")
}

func (s *SSOService) Start() error {
	port := s.config.ServerPort
	if port == "" {
		port = "8080"
	}

	corsHandler := gohandlers.CORS(
		gohandlers.AllowedOrigins([]string{"*"}),
		gohandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gohandlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Origin", "Authorization"}),
		gohandlers.AllowCredentials(),
	)

	log.Printf("SSO Service starting on port %s", port)
	return http.ListenAndServe("localhost:"+port, corsHandler(s.router))
}
