package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type LoginAttempt struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

type LogRepository interface {
	StoreLoginAttempt(attempt *LoginAttempt) error
	GetUserLogs(userID uint) ([]LoginAttempt, error)
	GetAllLogs() ([]LoginAttempt, error)
}

type RedisLogRepository struct {
	client *redis.Client
}

func NewRedisLogRepository(redisURL string) (*RedisLogRepository, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	return &RedisLogRepository{
		client: client,
	}, nil
}

func (r *RedisLogRepository) StoreLoginAttempt(attempt *LoginAttempt) error {
	ctx := context.Background()

	if attempt.Timestamp.IsZero() {
		attempt.Timestamp = time.Now()
	}

	data, err := json.Marshal(attempt)
	if err != nil {
		return err
	}

	// Store in Redis list
	// We'll use a list with key format "login_logs:{user_id}"
	key := fmt.Sprintf("login_logs:%d", attempt.UserID)

	if err := r.client.RPush(ctx, key, string(data)).Err(); err != nil {
		return err
	}

	if err := r.client.RPush(ctx, "login_logs:all", string(data)).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisLogRepository) GetUserLogs(userID uint) ([]LoginAttempt, error) {
	ctx := context.Background()
	key := fmt.Sprintf("login_logs:%d", userID)

	stringLogs, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	return parseLogStrings(stringLogs)
}

func (r *RedisLogRepository) GetAllLogs() ([]LoginAttempt, error) {
	ctx := context.Background()

	stringLogs, err := r.client.LRange(ctx, "login_logs:all", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	return parseLogStrings(stringLogs)
}

func parseLogStrings(stringLogs []string) ([]LoginAttempt, error) {
	logs := make([]LoginAttempt, 0, len(stringLogs))

	for _, logString := range stringLogs {
		var log LoginAttempt
		if err := json.Unmarshal([]byte(logString), &log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}
