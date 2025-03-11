package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"sso/internal/models"
	"sso/internal/repository"
	"sso/pkg/token"
)

type AuthHandler struct {
	userRepo     repository.UserRepository
	logRepo      repository.LogRepository
	tokenManager *token.JWTManager
}

func NewAuthHandler(userRepo repository.UserRepository, logRepo repository.LogRepository, tokenManager *token.JWTManager) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		logRepo:      logRepo,
		tokenManager: tokenManager,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existingUser, err := h.userRepo.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	user, err := h.userRepo.CreateUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	h.logRepo.StoreLoginAttempt(&repository.LoginAttempt{
		UserID:    user.ID,
		Email:     user.Email,
		Success:   true,
		IP:        getIP(r),
		UserAgent: r.UserAgent(),
	})

	tokenResp, err := h.tokenManager.Generate(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tokenResp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		h.logRepo.StoreLoginAttempt(&repository.LoginAttempt{
			Email:     req.Email,
			Success:   false,
			IP:        getIP(r),
			UserAgent: r.UserAgent(),
		})

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.logRepo.StoreLoginAttempt(&repository.LoginAttempt{
			UserID:    user.ID,
			Email:     user.Email,
			Success:   false,
			IP:        getIP(r),
			UserAgent: r.UserAgent(),
		})

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	h.logRepo.StoreLoginAttempt(&repository.LoginAttempt{
		UserID:    user.ID,
		Email:     user.Email,
		Success:   true,
		IP:        getIP(r),
		UserAgent: r.UserAgent(),
	})

	tokenResp, err := h.tokenManager.Generate(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResp)
}

func getIP(r *http.Request) string {
	// Try X-Forwarded-For header first (for proxies)
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return strings.Split(ip, ",")[0]
	}

	// Try X-Real-IP header next
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, claims, err := h.tokenManager.ValidateToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if claims["type"] != "refresh" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	userID := uint(claims["user_id"].(float64))
	tokenResp, err := h.tokenManager.Generate(userID)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResp)
}

func (h *AuthHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	_, claims, err := h.tokenManager.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userID := uint(claims["user_id"].(float64))

	response := models.TokenVerifyResponse{
		Valid:  true,
		UserID: userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
