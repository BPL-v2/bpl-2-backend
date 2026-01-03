package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type AchievementName string

const (
	AchievementParticipated                  AchievementName = "Participated in an event"
	AchievementWinner                        AchievementName = "Won an event"
	AchievementTeamlead                      AchievementName = "Teamlead"
	AchievementMVP                           AchievementName = "MVP"
	AchievementPlayed5Leagues                AchievementName = "Played 5 leagues"
	AchievementPlayed10Leagues               AchievementName = "Played 10 leagues"
	AchievementReachedLvl90                  AchievementName = "Reached level 90"
	AchievementReachedLvl95                  AchievementName = "Reached level 95"
	AchievementReachedLvl100                 AchievementName = "Reached level 100"
	AchievementSubmittedABounty              AchievementName = "Submitted a bounty"
	AchievementSubmittedAPointUnique         AchievementName = "Submitted a point unique"
	AchievementPlayed5DifferentAscendancies  AchievementName = "Played 5 different ascendancies"
	AchievementPlayed10DifferentAscendancies AchievementName = "Played 10 different ascendancies"
)

var Achievements = map[AchievementName]bool{
	AchievementParticipated:                  true,
	AchievementWinner:                        true,
	AchievementTeamlead:                      true,
	AchievementMVP:                           true,
	AchievementPlayed5Leagues:                true,
	AchievementPlayed10Leagues:               true,
	AchievementReachedLvl90:                  true,
	AchievementReachedLvl95:                  true,
	AchievementReachedLvl100:                 true,
	AchievementSubmittedABounty:              true,
	AchievementSubmittedAPointUnique:         true,
	AchievementPlayed5DifferentAscendancies:  true,
	AchievementPlayed10DifferentAscendancies: true,
}

type Achievement struct {
	UserId int             `gorm:"primaryKey"`
	Name   AchievementName `gorm:"primaryKey"`

	User *User `gorm:"foreignKey:UserId"`
}

type AchievementRepository struct {
	DB *gorm.DB
}

func NewAchievementRepository() *AchievementRepository {
	return &AchievementRepository{DB: config.DatabaseConnection()}
}

func (r *AchievementRepository) SaveAchievement(achievement *Achievement) error {
	return r.DB.Save(&achievement).Error
}

func (r *AchievementRepository) SaveAchievements(achievements []*Achievement) error {
	return r.DB.Save(&achievements).Error
}

func (r *AchievementRepository) GetAllAchievementsForUser() ([]*Achievement, error) {
	achievements := []*Achievement{}
	err := r.DB.Order("completed_at ASC").Find(&achievements).Error
	if err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *AchievementRepository) GetAchievementsForUser(userId int) ([]*Achievement, error) {
	achievements := []*Achievement{}
	err := r.DB.Where("user_id = ?", userId).Order("completed_at ASC").Find(&achievements).Error
	if err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *AchievementRepository) GetAllAchievements() ([]*Achievement, error) {
	achievements := []*Achievement{}
	err := r.DB.Find(&achievements).Error
	if err != nil {
		return nil, err
	}
	return achievements, nil
}
