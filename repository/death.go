package repository

import (
	"bpl/config"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Death struct {
	Time        time.Time `gorm:"not null"`
	CharacterId string    `gorm:"not null;index:idx_deaths_character_time,priority:1"`

	Character *Character `gorm:"foreignKey:CharacterId"`
}

func (Death) TableName() string {
	return "deaths"
}

type DeathRepository interface {
	SaveDeath(death *Death) error
	GetDeathsForUser(userId int, eventId int) ([]*Death, error)
	GetAllDeathsForEvent(eventId int) ([]*Death, error)
}

type DeathRepositoryImpl struct {
	DB *gorm.DB
}

func NewDeathRepository() DeathRepository {
	return &DeathRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *DeathRepositoryImpl) SaveDeath(death *Death) error {
	return r.DB.Create(death).Error
}

func (r *DeathRepositoryImpl) GetDeathsForUser(userId int, eventId int) ([]*Death, error) {
	deaths := []*Death{}
	err := r.DB.Joins("JOIN characters ON characters.id = deaths.character_id").
		Where("characters.user_id = ? AND characters.event_id = ?", userId, eventId).Order("deaths.time ASC").Find(&deaths).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching deaths for user %d and event %d: %w", userId, eventId, err)
	}
	return deaths, nil
}

func (r *DeathRepositoryImpl) GetAllDeathsForEvent(eventId int) ([]*Death, error) {
	deaths := []*Death{}
	err := r.DB.Joins("JOIN characters ON characters.id = deaths.character_id").
		Where("characters.event_id = ?", eventId).Order("deaths.time ASC").Find(&deaths).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching deaths for event %d: %w", eventId, err)
	}
	return deaths, nil
}
