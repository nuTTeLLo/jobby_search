package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID    string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Email string `json:"email" gorm:"uniqueIndex;not null;type:varchar(255)"`
	// PasswordHash is left over from password logins, which OAuth replaced.
	// Kept (nullable) so existing rows migrate without losing the column.
	PasswordHash *string   `json:"-" gorm:"type:varchar(255)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type AuthResponse struct {
	Token string   `json:"token"`
	User  AuthUser `json:"user"`
}

type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
