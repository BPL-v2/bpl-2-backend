package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type UniqueItemSource string

const (
	UniqueItemSourcePublicStash UniqueItemSource = "public_stash"
	UniqueItemSourceGuildStash  UniqueItemSource = "guild_stash"
	UniqueItemSourceCharacter   UniqueItemSource = "character"
)

type UniqueItemTracking struct {
	ItemId    string           `gorm:"not null"`
	ItemRefId int              `gorm:"not null;references:items(id)"`
	TeamId    int              `gorm:"not null;references:teams(id)"`
	PlayerId  *int             `gorm:"references:users(id)"`
	EventId   int              `gorm:"not null;references:events(id)"`
	Source    UniqueItemSource `gorm:"not null"`
	Timestamp time.Time        `gorm:"not null"`
}

func (UniqueItemTracking) TableName() string {
	return "unique_item_tracking"
}

type UniqueItemTrackingRepository interface {
	SaveBatch(entries []*UniqueItemTracking) error
	GetByEventId(eventId int) ([]*UniqueItemTracking, error)
	GetByTeamId(teamId int) ([]*UniqueItemTracking, error)
}

type UniqueItemTrackingRepositoryImpl struct {
	DB *gorm.DB
}

func NewUniqueItemTrackingRepository() UniqueItemTrackingRepository {
	return &UniqueItemTrackingRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *UniqueItemTrackingRepositoryImpl) SaveBatch(entries []*UniqueItemTracking) error {
	if len(entries) == 0 {
		return nil
	}
	return r.DB.Create(&entries).Error
}

func (r *UniqueItemTrackingRepositoryImpl) GetByEventId(eventId int) ([]*UniqueItemTracking, error) {
	var entries []*UniqueItemTracking
	err := r.DB.Where("event_id = ?", eventId).Order("timestamp DESC").Find(&entries).Error
	return entries, err
}

func (r *UniqueItemTrackingRepositoryImpl) GetByTeamId(teamId int) ([]*UniqueItemTracking, error) {
	var entries []*UniqueItemTracking
	err := r.DB.Where("team_id = ?", teamId).Order("timestamp DESC").Find(&entries).Error
	return entries, err
}
