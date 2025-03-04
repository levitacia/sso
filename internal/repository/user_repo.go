package repository

import "sso/internal/models"

type UserRepository interface {
	CreateUser(email string, password string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
}

type GormUserRepository struct {
	db *gorm.DB
}
