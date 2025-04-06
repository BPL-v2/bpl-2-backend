package repository

import (
	"bpl/config"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	Id               int              `gorm:"primaryKey"`
	EventId          int              `gorm:"not null;references:event(id)"`
	UserId           int              `gorm:"not null;references:event(id)"`
	Timestamp        time.Time        `gorm:"not null"`
	User             *User            `gorm:"foreignKey:UserId;references:Id"`
	ExpectedPlayTime ExpectedPlayTime `gorm:"not null"`
}

type SignupRepository struct {
	DB *gorm.DB
}

func NewSignupRepository() *SignupRepository {
	return &SignupRepository{DB: config.DatabaseConnection()}
}

func (r *SignupRepository) CreateSignup(signup *Signup) (*Signup, error) {
	result := r.DB.Save(signup)
	if result.Error != nil {
		return nil, result.Error
	}
	return signup, nil
}

func (r *SignupRepository) RemoveSignup(userId int, eventId int) error {
	result := r.DB.Delete(&Signup{}, "user_id = ? and event_id = ?", userId, eventId)
	return result.Error
}
func (r *SignupRepository) GetSignupForUser(userId int, eventId int) (*Signup, error) {
	signup := Signup{}
	result := r.DB.First(&signup, "user_id = ? and event_id = ?", userId, eventId)
	if result.Error != nil {
		return nil, result.Error
	}

	return &signup, nil
}

func (r *SignupRepository) GetSignupsForEvent(eventId int, limit int) ([]*Signup, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetSignupsForEvent"))
	defer timer.ObserveDuration()
	signups := make([]*Signup, 0)
	result := r.DB.Preload("User").Preload("User.OauthAccounts").Order("timestamp ASC").Limit(limit).Find(&signups, "event_id = ?", eventId)
	if result.Error != nil {
		return nil, result.Error
	}
	return signups, nil
}
