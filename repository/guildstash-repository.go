package repository

import (
	"bpl/config"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type GuildStashTab struct {
	Id            string        `gorm:"primaryKey;not null"`
	EventId       int           `gorm:"primaryKey;not null;index:guild_stash_tab_event_idx"`
	TeamId        int           `gorm:"not null;index:guild_stash_tab_team_idx"`
	OwnerId       int           `gorm:"not null"`
	Name          string        `gorm:"not null"`
	Type          string        `gorm:"not null"`
	Index         *int          `gorm:"null"`
	Color         *string       `gorm:"null"`
	ParentId      *string       `gorm:"null;references:Id"`
	ParentEventId *int          `gorm:"null"`
	Raw           string        `gorm:"type:text;not null;default:''"`
	FetchEnabled  bool          `gorm:"not null"`
	UserIds       pq.Int32Array `gorm:"not null;type:integer[]"`
	LastFetch     time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Event    Event            `gorm:"foreignKey:EventId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Team     Team             `gorm:"foreignKey:TeamId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Parent   *GuildStashTab   `gorm:"foreignKey:ParentId,ParentEventId;references:Id,EventId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Owner    User             `gorm:"foreignKey:OwnerId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Children []*GuildStashTab `gorm:"foreignKey:ParentId,ParentEventId;references:Id,EventId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

type GuildStashChangelog struct {
	Id          int       `gorm:"primaryKey"`
	Timestamp   time.Time `gorm:"not null"`
	GuildId     int       `gorm:"not null"`
	EventId     int       `gorm:"not null"`
	StashName   *string   `gorm:"null"`
	AccountName string    `gorm:"not null"`
	Action      Action    `gorm:"not null"`
	Number      int       `gorm:"not null"`
	ItemName    string    `gorm:"not null"`
	X           int       `gorm:"not null"`
	Y           int       `gorm:"not null"`
}

type Guild struct {
	Id     int `gorm:"primaryKey"`
	TeamId int `gorm:"primaryKey"`
	Name   string
	Tag    string
}

type Action int

const (
	ActionAdded    Action = 1
	ActionModified Action = 0
	ActionRemoved  Action = -1
)

func ActionFromString(action string) Action {
	switch action {
	case "added":
		return ActionAdded
	case "modified":
		return ActionModified
	case "removed":
		return ActionRemoved
	default:
		return ActionModified
	}
}

type GuildStashRepository struct {
	db *gorm.DB
}

func NewGuildStashRepository() *GuildStashRepository {
	return &GuildStashRepository{
		db: config.DatabaseConnection(),
	}
}

func (r *GuildStashRepository) DeleteAll(tabs []*GuildStashTab) error {
	if len(tabs) == 0 {
		return nil
	}
	return r.db.Delete(tabs).Error
}

func (r *GuildStashRepository) SaveAll(tabs []*GuildStashTab) (err error) {
	if len(tabs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, tab := range tabs {
			err = r.db.Save(tab).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GuildStashRepository) Save(tab *GuildStashTab) error {
	return r.db.Save(tab).Error
}
func (r *GuildStashRepository) GetById(stashId string, eventId int, preloads ...string) (tab *GuildStashTab, err error) {
	tab = &GuildStashTab{}
	query := r.db
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	err = query.Where(GuildStashTab{Id: stashId, EventId: eventId}).First(tab).Error
	return tab, err
}

func (r *GuildStashRepository) GetByEvent(eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{EventId: eventId}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepository) GetActiveByEvent(eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{EventId: eventId, FetchEnabled: true}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepository) GetByTeam(teamId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{TeamId: teamId}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepository) GetByUserAndEvent(userId int, eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where("event_id = ? AND ? = ANY(user_ids)", eventId, userId).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepository) SwitchStashFetch(stashId string, eventId int) (*GuildStashTab, error) {
	var tab GuildStashTab
	err := r.db.Where(GuildStashTab{Id: stashId, EventId: eventId}).First(&tab).Error
	if err != nil {
		return nil, err
	}

	tab.FetchEnabled = !tab.FetchEnabled
	if !tab.FetchEnabled {
		var children []*GuildStashTab
		err = r.db.Where(GuildStashTab{ParentId: &tab.Id, ParentEventId: &tab.EventId}).Find(&children).Error
		if err == nil && len(children) > 0 {
			for _, child := range children {
				child.FetchEnabled = false
			}
			if err := r.db.Save(&children).Error; err != nil {
				return nil, err
			}
		}
	}
	if err := r.db.Save(&tab).Error; err != nil {
		return nil, err
	}
	return &tab, nil
}

func (r *GuildStashRepository) SaveGuildstashLogs(logs []*GuildStashChangelog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.Save(logs).Error
}

func (r *GuildStashRepository) GetLatestLogEntryTimestampForGuild(event *Event, guildId int) (*int64, *int64) {
	var result struct {
		EarliestTimestamp *time.Time
		LatestTimestamp   *time.Time
	}
	err := r.db.Model(&GuildStashChangelog{}).
		Select("MIN(timestamp) as earliest_timestamp, MAX(timestamp) as latest_timestamp").
		Where("event_id = ? AND guild_id = ?", event.Id, guildId).
		Scan(&result).Error

	if err != nil || result.EarliestTimestamp == nil || result.LatestTimestamp == nil {
		return nil, nil
	}
	earliestTimestamp := result.EarliestTimestamp.Unix() - 1
	latestTimestamp := result.LatestTimestamp.Unix() + 1
	return &earliestTimestamp, &latestTimestamp
}

func (r *GuildStashRepository) GetLogs(eventId, guildId int, limit, offset *int, userName, stashName, itemName *string) ([]*GuildStashChangelog, error) {
	var logs []*GuildStashChangelog
	query := r.db.Model(&GuildStashChangelog{})
	query = query.Where("event_id = ? AND guild_id = ?", eventId, guildId)
	if userName != nil {
		query = query.Where("account_name = ?", strings.ReplaceAll(*userName, "-", "#"))
	}
	if stashName != nil {
		query = query.Where("stash_name = ?", *stashName)
	}
	if itemName != nil {
		query = query.Where("item_name ILIKE ?", "%"+*itemName+"%")
	}
	if limit != nil {
		query = query.Limit(*limit)
	}
	if offset != nil {
		query = query.Offset(*offset)
	}
	err := query.Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *GuildStashRepository) SaveGuild(guild *Guild) error {
	return r.db.Save(guild).Error
}

func (r *GuildStashRepository) GetGuildsForTeams(teamIds []int) ([]*Guild, error) {
	var guilds []*Guild
	err := r.db.Where("team_id IN ?", teamIds).Find(&guilds).Error
	if err != nil {
		return nil, err
	}
	return guilds, nil
}

type PlayerCompletion struct {
	Timestamp time.Time
	UserId    int    `gorm:"column:user_id"`
	ItemName  string `gorm:"column:item_name"`
	TeamId    int    `gorm:"column:team_id"`
}

func (r *GuildStashRepository) GetEarliestDeposits(event *Event) ([]*PlayerCompletion, error) {
	var results []*PlayerCompletion
	query := `
	SELECT timestamp, user_id, item_name, team_id 
		FROM (
			SELECT 
				gsc.timestamp, 
				o.user_id, 
				gsc.item_name, 
				g.team_id,
				ROW_NUMBER() OVER (
					PARTITION BY gsc.item_name, g.team_id 
					ORDER BY gsc.timestamp ASC
				) as rn
			FROM guild_stash_changelogs gsc 
			JOIN guilds g ON g.id = gsc.guild_id 
			JOIN oauths o ON o."name" = gsc.account_name
			WHERE gsc.action = 1 AND gsc.number = 1 and g.team_id in ?
		) ranked
		WHERE rn = 1
		ORDER BY timestamp;
		`
	err := r.db.Raw(query, event.TeamIds()).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *GuildStashRepository) GetGuildById(guildId int) (*Guild, error) {
	var guild Guild
	err := r.db.Where("id = ?", guildId).First(&guild).Error
	if err != nil {
		return nil, err
	}
	return &guild, nil
}
