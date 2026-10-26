package domain

import "time"

// MaxFitProfileLength caps the fit profile. It is sent with every posting the
// daily scrape scores, so a runaway paste would cost on every call.
const MaxFitProfileLength = 20000

// Profile holds per-user settings that are not about any one job. For now that is
// only the fit profile: a free-text description of the user's experience and what
// they want from a role, which the daily scrape gives the model scoring each
// discovered posting.
type Profile struct {
	UserID     string     `json:"user_id" gorm:"primaryKey;type:varchar(36)"`
	FitProfile string     `json:"fit_profile" gorm:"type:text"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type ProfileInput struct {
	FitProfile string `json:"fit_profile"`
}
