package repository

import (
	"bpl/client"
	"bpl/config"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type CacheKey int

const (
	Score  CacheKey = 1
	Ladder CacheKey = 2
)

type CachedData struct {
	Key       CacheKey  `gorm:"primaryKey"`
	EventId   int       `gorm:"primaryKey"`
	Data      []byte    `gorm:"not null"`
	Timestamp time.Time `gorm:"not null"`
}

type CachedDataRepository interface {
	GetLatestScore(eventId int) ([]byte, error)
	GetLatestLadder(eventId int) ([]byte, error)
	GetLatestLadderUnMarshalled(eventId int) (*client.Ladder, error)
	SaveScore(eventId int, scores []byte) error
	SaveLadder(eventId int, ladder *client.Ladder) error
}

type CachedDataRepositoryImpl struct {
	db *gorm.DB
}

func NewCachedDataRepository() CachedDataRepository {
	return &CachedDataRepositoryImpl{db: config.DatabaseConnection()}
}

func (r *CachedDataRepositoryImpl) GetLatestScore(eventId int) ([]byte, error) {
	var data CachedData
	result := r.db.First(&data, CachedData{Key: Score, EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return data.Data, nil
}

func (r *CachedDataRepositoryImpl) GetLatestLadder(eventId int) ([]byte, error) {
	var data CachedData
	result := r.db.First(&data, CachedData{Key: Ladder, EventId: eventId})
	if result.Error != nil {
		return nil, result.Error
	}
	return data.Data, nil
}

func (r *CachedDataRepositoryImpl) GetLatestLadderUnMarshalled(eventId int) (*client.Ladder, error) {
	var data CachedData
	result := r.db.First(&data, CachedData{Key: Ladder, EventId: eventId})
	if result.Error != nil {
		return nil, result.Error

	}
	var ladder client.Ladder
	err := json.Unmarshal(data.Data, &ladder)
	if err != nil {
		return nil, err
	}
	return &ladder, nil

}

func (r *CachedDataRepositoryImpl) SaveScore(eventId int, scores []byte) error {
	return r.db.Save(&CachedData{
		Key:       Score,
		EventId:   eventId,
		Data:      scores,
		Timestamp: time.Now(),
	}).Error
}

func (r *CachedDataRepositoryImpl) SaveLadder(eventId int, ladder *client.Ladder) error {
	data, err := json.Marshal(ladder)
	if err != nil {
		return err
	}
	return r.db.Save(&CachedData{
		Key:       Ladder,
		EventId:   eventId,
		Data:      data,
		Timestamp: time.Now(),
	}).Error
}
