package handlers

import (
	"encoding/json"
	"net/http"

	"sso/internal/repository"
)

type LogHandler struct {
	userRepo repository.UserRepository
	logRepo  repository.LogRepository
}

func NewLogHandler(userRepo repository.UserRepository, logRepo repository.LogRepository) *LogHandler {
	return &LogHandler{
		userRepo: userRepo,
		logRepo:  logRepo,
	}
}

func (h *LogHandler) GetUserLogs(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var logs []repository.LoginAttempt

	if user.Email == "admin" {
		logs, err = h.logRepo.GetAllLogs()
	} else {
		logs, err = h.logRepo.GetUserLogs(userID)
	}

	if err != nil {
		http.Error(w, "Failed to retrieve logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
