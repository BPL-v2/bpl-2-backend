package repository

import (
	"bpl/config"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Character struct {
	Id               string  `gorm:"not null;primaryKey"`
	UserId           *int    `gorm:"null;index"`
	EventId          int     `gorm:"not null;index"`
	Name             string  `gorm:"not null"`
	Level            int     `gorm:"not null"`
	MainSkill        string  `gorm:"not null"`
	Ascendancy       string  `gorm:"not null"`
	AscendancyPoints int     `gorm:"not null"`
	Pantheon         bool    `gorm:"not null"`
	AtlasPoints      int     `gorm:"not null"`
	OldAccountName   *string `gorm:"null;index"`
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
	if other == nil {
		return false
	}
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

type PoBExport []byte

func (p *PoBExport) FromString(s string) error {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %w", err)
	}

	*p = PoBExport(decoded)
	return nil
}

func (p PoBExport) ToString() string {
	encoded := base64.StdEncoding.EncodeToString([]byte(p))
	// Convert to URL-safe format
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	return encoded
}

func (p *PoBExport) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan PoBExport: expected []byte, got %T", value)
	}

	*p = PoBExport(bytes)
	return nil
}

func (p PoBExport) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return []byte(p), nil
}

type CharacterPob struct {
	Id          int       `gorm:"not null;primaryKey"`
	CharacterId string    `gorm:"not null;index"`
	Level       int       `gorm:"not null"`
	MainSkill   string    `gorm:"not null"`
	Ascendancy  string    `gorm:"not null"`
	Export      PoBExport `gorm:"not null;type:bytea"`
	CreatedAt   time.Time `gorm:"not null;index"`
	UpdatedAt   time.Time `gorm:"not null"`
}

type CharacterRepository struct {
	DB *gorm.DB
}

func NewCharacterRepository() *CharacterRepository {
	return &CharacterRepository{DB: config.DatabaseConnection()}
}

func (r *CharacterRepository) GetPobByCharacterIdBeforeTimestamp(characterId string, timestamp time.Time) (*CharacterPob, error) {
	characterPob := &CharacterPob{}
	err := r.DB.Where("character_id = ? AND created_at < ?", characterId, timestamp).
		Order("created_at DESC").First(characterPob).Error
	if err != nil {
		return nil, err
	}
	return characterPob, nil
}

func (r *CharacterRepository) GetPobs(characterId string) ([]*CharacterPob, error) {
	pobs := []*CharacterPob{}
	err := r.DB.Where("character_id = ?", characterId).Order("created_at ASC").Find(&pobs).Error
	if err != nil {
		return nil, fmt.Errorf("error getting PoBs for character %s: %w", characterId, err)
	}
	return pobs, nil
}

func (r *CharacterRepository) SaveCharacters(characters []*Character) error {
	if len(characters) == 0 {
		return nil
	}
	return r.DB.CreateInBatches(characters, 500).Error
}

func (r *CharacterRepository) SaveCharacterStats(characterStats []*CharacterStat) error {
	if len(characterStats) == 0 {
		return nil
	}
	return r.DB.CreateInBatches(characterStats, 500).Error
}

func (r *CharacterRepository) CreateCharacterStat(characterStat *CharacterStat) error {
	return r.DB.Create(&characterStat).Error
}

func (r *CharacterRepository) SavePoB(characterPoB *CharacterPob) error {
	return r.DB.Save(&characterPoB).Error
}

func (r *CharacterRepository) Save(character *Character) error {
	if character.Id == "" || character.Name == "" {
		return fmt.Errorf("character ID and Name must be set")
	}
	return r.DB.Save(&character).Error
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
func (r *CharacterRepository) GetCharactersForUser(user *User) ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetCharactersForUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	accountName := ""
	if user.GetAccountName(ProviderPoE) != nil {
		accountName = strings.Split(*user.GetAccountName(ProviderPoE), "#")[0]
	}
	err := r.DB.Where("user_id = ? or old_account_name = ?", user.Id, accountName).Find(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}
func (r *CharacterRepository) GetCharacterById(characterId string) (*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetCharacterById"))
	defer timer.ObserveDuration()
	character := &Character{}
	err := r.DB.Where("id = ?", characterId).First(character).Error
	if err != nil {
		return nil, err
	}
	return character, nil
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

func (r *CharacterRepository) GetLatestCharacterStats(characterId string) (*CharacterStat, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetLatestCharacterStats"))
	defer timer.ObserveDuration()
	charData := &CharacterStat{}
	err := r.DB.Where("character_id = ?", characterId).Order("time DESC").First(charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest character stats for character %s: %w", characterId, err)
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

func (r *CharacterRepository) GetAllHighestLevelCharactersForEachEventAndUser() ([]*Character, error) {
	timer := prometheus.NewTimer(queryDuration.WithLabelValues("GetAllHighestLevelCharactersForEachEventAndUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	query := `SELECT c.* FROM characters c
		JOIN (
			SELECT event_id, user_id, MAX(level) as max_level
			FROM characters
			WHERE user_id IS NOT NULL
			GROUP BY event_id, user_id
		) as max_chars
		ON c.event_id = max_chars.event_id AND c.user_id = max_chars.user_id AND c.level = max_chars.max_level
		ORDER BY c.event_id, c.user_id`
	err := r.DB.Raw(query).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting all highest level characters for each event and user: %w", err)
	}
	return charData, nil
}
