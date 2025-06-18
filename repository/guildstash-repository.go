package repository

import (
	"bpl/config"
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

	Event  Event          `gorm:"foreignKey:EventId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Team   Team           `gorm:"foreignKey:TeamId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Parent *GuildStashTab `gorm:"foreignKey:ParentId,ParentEventId;references:Id,EventId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Owner  User           `gorm:"foreignKey:OwnerId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
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

func (r *GuildStashRepository) SaveAll(tabs []*GuildStashTab) error {
	if len(tabs) == 0 {
		return nil
	}
	return r.db.Save(tabs).Error
}

func (r *GuildStashRepository) Save(tab *GuildStashTab) error {
	return r.db.Save(tab).Error
}
func (r *GuildStashRepository) GetById(stashId string, eventId int) (tab *GuildStashTab, err error) {
	tab = &GuildStashTab{}
	err = r.db.Where(GuildStashTab{Id: stashId, EventId: eventId}).First(tab).Error
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

func (r *GuildStashRepository) GetByTeam(teamId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{TeamId: teamId}).Find(&tabs).Error
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
