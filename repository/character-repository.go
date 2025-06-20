package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/utils"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Character struct {
	UserID           int       `gorm:"not null;index"`
	EventID          int       `gorm:"not null;index"`
	Name             string    `gorm:"not null"`
	Level            int       `gorm:"not null"`
	MainSkill        string    `gorm:"not null"`
	Ascendancy       string    `gorm:"not null"`
	AscendancyPoints int       `gorm:"not null"`
	Pantheon         bool      `gorm:"not null"`
	Timestamp        time.Time `gorm:"not null;index"`
	AtlasNodeCount   int       `gorm:"not null"`
	// Pob        string    `gorm:"not null"`
	User  *User  `gorm:"foreignKey:UserID"`
	Event *Event `gorm:"foreignKey:EventID"`
}

type Atlas struct {
	UserID  int           `gorm:"not null;primaryKey"`
	EventID int           `gorm:"not null;index;primaryKey"`
	Index   int           `gorm:"not null"`
	Tree1   pq.Int32Array `gorm:"not null;type:integer[]"`
	Tree2   pq.Int32Array `gorm:"not null;type:integer[]"`
	Tree3   pq.Int32Array `gorm:"not null;type:integer[]"`

	User  *User  `gorm:"foreignKey:UserID"`
	Event *Event `gorm:"foreignKey:EventID"`
}

type CharacterRepository struct {
	DB *gorm.DB
}

func NewCharacterRepository() *CharacterRepository {
	return &CharacterRepository{DB: config.DatabaseConnection()}
}

func (r *CharacterRepository) CreateCharacterCheckpoint(character *Character) error {
	return r.DB.Create(&character).Error
}

func (r *CharacterRepository) SaveAtlasTrees(userId int, eventId int, atlasPassiveTrees []client.AtlasPassiveTree) error {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("SaveAtlasTrees"))
	defer timer.ObserveDuration()
	atlas := Atlas{}
	r.DB.Where(Atlas{UserID: userId, EventID: eventId}).First(&atlas)
	if atlas.UserID == 0 {
		atlas.UserID = userId
		atlas.EventID = eventId
		atlas.Tree1 = pq.Int32Array{}
		atlas.Tree2 = pq.Int32Array{}
		atlas.Tree3 = pq.Int32Array{}
	}
	atlas.Index = -1
	for i, v := range atlasPassiveTrees {
		if i == 0 {
			atlas.Tree1 = utils.ConvertIntSlice(v.Hashes)
		} else if i == 1 {
			atlas.Tree2 = utils.ConvertIntSlice(v.Hashes)
		} else if i == 2 {
			atlas.Tree3 = utils.ConvertIntSlice(v.Hashes)
		}
		if strings.HasPrefix(v.Name, "x") {
			atlas.Index = i
		}
	}

	return r.DB.Save(&atlas).Error
}

func (r *CharacterRepository) GetLatestCharactersForEvent(eventId int) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLatestCharactersForEvent"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	query := `
		SELECT c.* FROM characters c
		INNER JOIN (
			SELECT user_id, MAX(timestamp) AS timestamp
			FROM characters
			WHERE event_id = ?
			GROUP BY user_id
		) latest
		ON c.user_id = latest.user_id AND c.timestamp = latest.timestamp
		`

	err := r.DB.Raw(query, eventId).Scan(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}
func (r *CharacterRepository) GetLatestEventCharactersForUser(userId int) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLatestEventCharactersForUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	query := `
		SELECT c.* FROM characters c
		INNER JOIN (
			SELECT event_id, MAX(timestamp) AS timestamp
			FROM characters
			WHERE user_id = ?
			GROUP BY event_id
		) latest
		ON c.event_id = latest.event_id AND c.timestamp = latest.timestamp
	`
	err := r.DB.Raw(query, userId).Scan(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepository) GetEventCharacterHistoryForUser(userId int, eventId int) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetEventCharacterHistoryForUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	err := r.DB.Where(Character{UserID: userId, EventID: eventId}).Find(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepository) GetTeamAtlasesForEvent(eventId int, teamId int) (atlas []*Atlas, err error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetTeamAtlasesForEvent"))
	defer timer.ObserveDuration()
	query := `
		SELECT a.* FROM atlas a
		JOIN team_users tu ON a.user_id = tu.user_id
		WHERE tu.team_id = ? AND a.event_id = ?
	`
	err = r.DB.Raw(query, teamId, eventId).Scan(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}
