package repository

import (
	"bpl/config"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Activity struct {
	Time    time.Time `gorm:"not null;index"`
	UserId  int       `gorm:"not null;index"`
	EventId int       `gorm:"not null;index"`

	User  *User  `gorm:"foreignKey:UserId"`
	Event *Event `gorm:"foreignKey:EventId"`
}

func (Activity) TableName() string {
	return "activity"
}

type ActivityRepository struct {
	DB *gorm.DB
}

func NewActivityRepository() *ActivityRepository {
	return &ActivityRepository{DB: config.DatabaseConnection()}
}

func (r *ActivityRepository) SaveActivity(activity *Activity) error {
	return r.DB.Create(&activity).Error
}

func (r *ActivityRepository) GetActivity(userId int, eventId int) ([]*Activity, error) {
	activities := []*Activity{}
	err := r.DB.Where("user_id = ? AND event_id = ?", userId, eventId).Order("time ASC").Find(&activities).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching activities for user %d and event %d: %w", userId, eventId, err)
	}
	return activities, nil
}
