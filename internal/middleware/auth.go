package middleware

import (
	"context"
	"log"
	"net/http"
	"sso/pkg/token"
	"strings"
)

type AuthMiddleware struct {
	tokenManager *token.JWTManager
}

func NewAuthMiddleware(tokenManager *token.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{tokenManager}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			log.Println("Token is empty")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		_, claims, err := m.tokenManager.ValidateToken(tokenString)
		if err != nil {
			log.Println("Token not verify")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if claims["type"] != "access" {
			http.Error(w, "Invalid token type", http.StatusUnauthorized)
			return
		}

		userID := uint(claims["user_id"].(float64))
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
