package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type TimingKey string

const (
	CharacterRefetchDelay          TimingKey = "delay_after_character_is_refetched"
	CharacterRefetchDelayImportant TimingKey = "delay_after_po_relevant_character_is_refetched"
	CharacterRefetchDelayInactive  TimingKey = "delay_after_inactive_character_is_refetched"

	LeagueAccountRefetchDelay          TimingKey = "delay_after_league_account_is_refetched"
	LeagueAccountRefetchDelayImportant TimingKey = "delay_after_po_relevant_league_account_is_refetched"
	LeagueAccountRefetchDelayInactive  TimingKey = "delay_after_inactive_league_account_is_refetched"

	PoBRecalculationDelay     TimingKey = "delay_after_pob_is_recalculated"
	CharacterNameRefetchDelay TimingKey = "delay_after_character_name_is_refetched"

	InactivityDuration TimingKey = "character_inactivity_duration"

	LadderUpdateInterval      TimingKey = "ladder_update_interval"
	GuildstashUpdateInterval  TimingKey = "guildstash_update_interval"
	PublicStashUpdateInterval TimingKey = "public_stash_update_interval"
)

var TimingKeyDescriptions = map[TimingKey]string{
	CharacterRefetchDelay:          "Minimum delay after a character is refetched",
	CharacterRefetchDelayImportant: "Minimum delay after a character that is relevant for Personal Objective points is refetched",
	CharacterRefetchDelayInactive:  "Minimum delay after an inactive character is refetched",

	LeagueAccountRefetchDelay:          "Minimum delay after a league account is refetched",
	LeagueAccountRefetchDelayImportant: "Minimum delay after a league account that is relevant for Personal Objective points is refetched",
	LeagueAccountRefetchDelayInactive:  "Minimum delay after an inactive league account is refetched",

	PoBRecalculationDelay:     "Minimum delay after a PoB will be recalculated",
	CharacterNameRefetchDelay: "Minimum delay after a character name is refetched",
	InactivityDuration:        "Duration after which a character is considered inactive",

	LadderUpdateInterval:      "Interval at which the ladder is updated",
	GuildstashUpdateInterval:  "Interval at which the guild stash is updated",
	PublicStashUpdateInterval: "Interval at which the public stash is updated",
}

var DefaultTimings = map[TimingKey]time.Duration{
	CharacterRefetchDelay:          5 * time.Minute,
	CharacterRefetchDelayImportant: 2 * time.Minute,
	CharacterRefetchDelayInactive:  30 * time.Minute,

	LeagueAccountRefetchDelay:          10 * time.Minute,
	LeagueAccountRefetchDelayImportant: 5 * time.Minute,
	LeagueAccountRefetchDelayInactive:  60 * time.Minute,

	PoBRecalculationDelay:     5 * time.Minute,
	CharacterNameRefetchDelay: 60 * time.Minute,

	InactivityDuration: 30 * time.Minute,

	LadderUpdateInterval:      30 * time.Second,
	GuildstashUpdateInterval:  2 * time.Minute,
	PublicStashUpdateInterval: 0 * time.Minute,
}

type Timing struct {
	Key        TimingKey `gorm:"primaryKey"`
	DurationMs int64     `gorm:"not null"`
}

func (t *Timing) GetDuration() time.Duration {
	return time.Duration(t.DurationMs) * time.Millisecond
}

type TimingRepository struct {
	db *gorm.DB
}

func NewTimingRepository() *TimingRepository {
	db := config.DatabaseConnection()
	return &TimingRepository{db: db}
}

func (r *TimingRepository) GetTimings() (map[TimingKey]time.Duration, error) {
	var timings []*Timing
	err := r.db.Find(&timings).Error
	if err != nil {
		return nil, err
	}
	result := make(map[TimingKey]time.Duration)
	for _, timing := range timings {
		result[timing.Key] = time.Duration(timing.DurationMs) * time.Millisecond
	}
	for key, defaultDuration := range DefaultTimings {
		if _, exists := result[key]; !exists {
			result[key] = defaultDuration
		}
	}
	return result, nil
}

func (r *TimingRepository) SaveTimings(timings []*Timing) error {
	for _, timing := range timings {
		err := r.db.Save(timing).Error
		if err != nil {
			return err
		}
	}
	return nil
}
