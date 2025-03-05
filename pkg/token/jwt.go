package token

import (
	"fmt"
	"sso/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTManager struct {
	Secret          string
	TokenDuration   time.Duration
	RefreshDuration time.Duration
}

func NewJWTManager(secret string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		Secret:          secret,
		TokenDuration:   tokenDuration,
		RefreshDuration: tokenDuration * 2,
	}
}

func (m *JWTManager) Generate(userID uint) (models.TokenResponse, error) {
	expiresAt := time.Now().Add(m.TokenDuration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     expiresAt.Unix(),
	})

	tokenString, err := token.SignedString([]byte(m.Secret))
	if err != nil {
		return models.TokenResponse{}, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     time.Now().Add(m.RefreshDuration).Unix(),
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(m.Secret))
	if err != nil {
		return models.TokenResponse{}, err
	}

	return models.TokenResponse{
		Token:        tokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
	}, nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*jwt.Token, jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.Secret), nil
	})

	if err != nil {
		return nil, nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, nil, fmt.Errorf("invalid token")
	}

	return token, claims, nil
}
