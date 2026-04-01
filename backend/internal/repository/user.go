package repository

import (
	"errors"

	"job-tracker-backend/internal/domain"
	appErrors "job-tracker-backend/pkg/errors"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdatePasswordHash(id, hash string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}
