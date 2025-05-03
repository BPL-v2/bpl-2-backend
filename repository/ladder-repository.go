package repository

import (
	"bpl/client"
	"bpl/config"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type LadderEntry struct {
	UserId     int    `gorm:"index;not null"`
	Account    string `gorm:"not null"`
	Character  string `gorm:"not null"`
	Class      string `gorm:"not null"`
	Level      int    `gorm:"not null"`
	Delve      int    `gorm:"not null"`
	Experience int    `gorm:"not null"`
	Rank       int    `gorm:"not null"`
	EventId    int    `gorm:"foreignKey:EventId;constraint:OnDelete:CASCADE;index;not null"`
}

type LadderRepository struct {
	DB *gorm.DB
}

func NewLadderRepository() *LadderRepository {
	return &LadderRepository{DB: config.DatabaseConnection()}
}

func (r *LadderRepository) UpsertLadder(ladder []*client.LadderEntry, eventId int, playerMap map[string]int) error {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("UpsertLadder"))
	defer timer.ObserveDuration()

	err := r.DB.Delete(&LadderEntry{}, &LadderEntry{EventId: eventId}).Error
	if err != nil {
		return err
	}
	dbEntries := make([]*LadderEntry, 0, len(ladder))
	for _, entry := range ladder {
		dbEntry := &LadderEntry{
			Character: entry.Character.Name,
			Level:     entry.Character.Level,
			Class:     entry.Character.Class,
			Account:   entry.Account.Name,
			EventId:   eventId,
			UserId:    playerMap[entry.Character.Name],
			Rank:      entry.Rank,
		}
		if entry.Character.Depth != nil && entry.Character.Depth.Default != nil {
			dbEntry.Delve = *entry.Character.Depth.Default
		}
		if entry.Character.Experience != nil {
			dbEntry.Experience = *entry.Character.Experience
		}
		dbEntries = append(dbEntries, dbEntry)
	}
	return r.DB.CreateInBatches(dbEntries, 500).Error
}

func (r *LadderRepository) GetLadderForEvent(eventId int) ([]*LadderEntry, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLadderForEvent"))
	defer timer.ObserveDuration()
	var ladder []*LadderEntry
	result := r.DB.Find(&ladder, &LadderEntry{EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return ladder, nil
}
