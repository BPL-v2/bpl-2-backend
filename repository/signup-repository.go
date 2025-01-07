package repository

import (
	"time"

	"gorm.io/gorm"
)

type ExpectedPlayTime string

const (
	VeryLow  ExpectedPlayTime = "VERY_LOW"
	Low      ExpectedPlayTime = "LOW"
	Medium   ExpectedPlayTime = "MEDIUM"
	High     ExpectedPlayTime = "HIGH"
	VeryHigh ExpectedPlayTime = "VERY_HIGH"
	Extreme  ExpectedPlayTime = "EXTREME"
	NoLife   ExpectedPlayTime = "NO_LIFE"
)

type Signup struct {
	ID               int              `gorm:"primaryKey"`
	EventID          int              `gorm:"not null;references:event(id)"`
	UserID           int              `gorm:"not null;references:event(id)"`
	Timestamp        time.Time        `gorm:"not null"`
	User             *User            `gorm:"foreignKey:UserID;references:ID"`
	ExpectedPlayTime ExpectedPlayTime `gorm:"not null"`
}

type SignupRepository struct {
	DB *gorm.DB
}

func NewSignupRepository(db *gorm.DB) *SignupRepository {
	return &SignupRepository{DB: db}
}

func (r *SignupRepository) CreateSignup(signup *Signup) (*Signup, error) {
	result := r.DB.Save(signup)
	if result.Error != nil {
		return nil, result.Error
	}
	return signup, nil
}

func (r *SignupRepository) RemoveSignup(userID int, eventID int) error {
	result := r.DB.Delete(&Signup{}, "user_id = ? and event_id = ?", userID, eventID)
	return result.Error
}
func (r *SignupRepository) GetSignupForUser(userID int, eventID int) (*Signup, error) {
	signup := Signup{}
	result := r.DB.First(&signup, "user_id = ? and event_id = ?", userID, eventID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &signup, nil
}

func (r *SignupRepository) GetSignupsForEvent(eventId int, limit int) ([]*Signup, error) {
	signups := make([]*Signup, 0)
	result := r.DB.Preload("User").Order("timestamp ASC").Limit(limit).Find(&signups, "event_id = ?", eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return signups, nil
}
