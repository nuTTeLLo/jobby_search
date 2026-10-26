package repository

import (
	"errors"

	"job-tracker-backend/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// Get returns the user's profile, or an empty one if they have never saved it.
func (r *ProfileRepository) Get(userID string) (*domain.Profile, error) {
	var profile domain.Profile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.Profile{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Save creates or replaces the user's profile.
func (r *ProfileRepository) Save(profile *domain.Profile) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"fit_profile", "updated_at"}),
	}).Create(profile).Error
}
