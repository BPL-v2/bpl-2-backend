package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/utils"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Character struct {
	Id               string `gorm:"not null;primaryKey"`
	UserId           int    `gorm:"not null;index"`
	EventId          int    `gorm:"not null;index"`
	Name             string `gorm:"not null"`
	Level            int    `gorm:"not null"`
	MainSkill        string `gorm:"not null"`
	Ascendancy       string `gorm:"not null"`
	AscendancyPoints int    `gorm:"not null"`
	Pantheon         bool   `gorm:"not null"`
	AtlasPoints      int    `gorm:"not null"`
}

type CharacterStat struct {
	Time          time.Time `gorm:"not null;index"`
	EventId       int       `gorm:"not null;index"`
	CharacterId   string    `gorm:"not null;index"`
	DPS           int64     `gorm:"not null"`
	EHP           int32     `gorm:"not null"`
	PhysMaxHit    int32     `gorm:"not null"`
	EleMaxHit     int32     `gorm:"not null"`
	HP            int32     `gorm:"not null"`
	Mana          int32     `gorm:"not null"`
	ES            int32     `gorm:"not null"`
	Armour        int32     `gorm:"not null"`
	Evasion       int32     `gorm:"not null"`
	XP            int64     `gorm:"not null"`
	MovementSpeed int32     `gorm:"not null"`

	Character *Character `gorm:"foreignKey:CharacterId"`
	Event     *Event     `gorm:"foreignKey:EventId"`
}

func (c *CharacterStat) IsEqual(other *CharacterStat) bool {
	return c.DPS == other.DPS &&
		c.EHP == other.EHP &&
		c.PhysMaxHit == other.PhysMaxHit &&
		c.EleMaxHit == other.EleMaxHit &&
		c.HP == other.HP &&
		c.Mana == other.Mana &&
		c.ES == other.ES &&
		c.Armour == other.Armour &&
		c.Evasion == other.Evasion &&
		c.XP == other.XP &&
		c.MovementSpeed == other.MovementSpeed
}

type CharacterPob struct {
	Id          int       `gorm:"not null;primaryKey"`
	CharacterId string    `gorm:"not null;index"`
	Level       int       `gorm:"not null"`
	MainSkill   string    `gorm:"not null"`
	Ascendancy  string    `gorm:"not null"`
	Export      string    `gorm:"not null;type:text"`
	Timestamp   time.Time `gorm:"not null;index"`
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

func (r *CharacterRepository) GetPobByCharacterIdBeforeTimestamp(characterId string, timestamp time.Time) (*CharacterPob, error) {
	characterPob := &CharacterPob{}
	err := r.DB.Where("character_id = ? AND timestamp < ?", characterId, timestamp).
		Order("timestamp DESC").First(characterPob).Error
	if err != nil {
		return nil, err
	}
	return characterPob, nil
}

func (r *CharacterRepository) CreateCharacterStat(characterStat *CharacterStat) error {
	return r.DB.Create(&characterStat).Error
}

func (r *CharacterRepository) SavePoB(characterPoB *CharacterPob) error {
	return r.DB.Save(&characterPoB).Error
}

func (r *CharacterRepository) CreateCharacterCheckpoint(character *Character) error {
	if character.Id == "" || character.Name == "" {
		return fmt.Errorf("character ID and Name must be set")
	}
	fmt.Println("Creating character checkpoint for", character.Name, "with id", character.Id)
	return r.DB.Save(&character).Error
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
		switch i {
		case 0:
			atlas.Tree1 = utils.ConvertIntSlice(v.Hashes)
		case 1:
			atlas.Tree2 = utils.ConvertIntSlice(v.Hashes)
		case 2:
			atlas.Tree3 = utils.ConvertIntSlice(v.Hashes)
		}
		if strings.HasPrefix(v.Name, "x") {
			atlas.Index = i
		}
	}

	return r.DB.Save(&atlas).Error
}

func (r *CharacterRepository) GetCharactersForEvent(eventId int) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetCharactersForEvent"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	err := r.DB.Find(&charData, Character{EventId: eventId}).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}
func (r *CharacterRepository) GetCharactersForUser(userId int) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetCharactersForUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	err := r.DB.Find(&charData, Character{UserId: userId}).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepository) GetCharacterHistory(characterId string) ([]*CharacterStat, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetCharacterHistory"))
	defer timer.ObserveDuration()
	charData := []*CharacterStat{}
	err := r.DB.Where(CharacterStat{CharacterId: characterId}).Find(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepository) GetLatestCharacterStatsForEvent(eventId int) (map[string]*CharacterStat, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLatestCharacterStatsForEvent"))
	defer timer.ObserveDuration()
	charData := []*CharacterStat{}
	query := `SELECT DISTINCT ON (character_id) * FROM character_stats
		WHERE event_id = ?
		ORDER BY character_id, time DESC
	`
	err := r.DB.Raw(query, eventId).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest character stats for event %d: %w", eventId, err)
	}
	result := make(map[string]*CharacterStat, len(charData))
	for _, stat := range charData {
		result[stat.CharacterId] = stat
	}
	return result, nil
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

func (r *CharacterRepository) GetLatestStatsForEvent(eventId int) ([]*CharacterStat, error) {
	charData := []*CharacterStat{}
	// for each unique character_id in the event, get the latest stat
	query := `		SELECT DISTINCT ON (character_id) * FROM character_stats
		WHERE event_id = ?
		ORDER BY character_id, time DESC	
	`
	err := r.DB.Raw(query, eventId).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest stats for event %d: %w", eventId, err)
	}
	return charData, nil
}

func (r *CharacterRepository) GetLatestPoBsForEvent(eventId int) ([]*CharacterPob, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLatestPoBsForEvent"))
	defer timer.ObserveDuration()
	charData := []*CharacterPob{}
	query := `SELECT DISTINCT ON (character_id) pobs.* from character_pobs as pobs
		JOIN characters ON pobs.character_id = characters.id
		WHERE characters.event_id = ?
		ORDER BY character_id, timestamp DESC`
	err := r.DB.Raw(query, eventId).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest PoBs for event %d: %w", eventId, err)
	}
	return charData, nil
}
