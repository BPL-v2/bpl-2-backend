package repository

import (
	"bpl/config"
	"time"

	"gorm.io/gorm"
)

type ObjectiveType string

const (
	ITEM       ObjectiveType = "ITEM"
	PLAYER     ObjectiveType = "PLAYER"
	TEAM       ObjectiveType = "TEAM"
	SUBMISSION ObjectiveType = "SUBMISSION"
)

type AggregationType string

const (
	SUM_LATEST          AggregationType = "SUM_LATEST"
	EARLIEST            AggregationType = "EARLIEST"
	EARLIEST_FRESH_ITEM AggregationType = "EARLIEST_FRESH_ITEM"
	MAXIMUM             AggregationType = "MAXIMUM"
	MINIMUM             AggregationType = "MINIMUM"
	DIFFERENCE_BETWEEN  AggregationType = "DIFFERENCE_BETWEEN"
)

type NumberField string

const (
	STACK_SIZE NumberField = "STACK_SIZE"

	PLAYER_LEVEL         NumberField = "PLAYER_LEVEL"
	DELVE_DEPTH          NumberField = "DELVE_DEPTH"
	DELVE_DEPTH_PAST_100 NumberField = "DELVE_DEPTH_PAST_100"
	PANTHEON             NumberField = "PANTHEON"
	ASCENDANCY           NumberField = "ASCENDANCY"
	PLAYER_SCORE         NumberField = "PLAYER_SCORE"

	SUBMISSION_VALUE NumberField = "SUBMISSION_VALUE"
)

var ObjectiveTypeToNumberFields = map[ObjectiveType][]NumberField{
	ITEM:       {STACK_SIZE},
	PLAYER:     {PLAYER_LEVEL, DELVE_DEPTH, DELVE_DEPTH_PAST_100, PANTHEON, ASCENDANCY, PLAYER_SCORE},
	TEAM:       {PLAYER_LEVEL, DELVE_DEPTH, DELVE_DEPTH_PAST_100, PANTHEON, ASCENDANCY, PLAYER_SCORE},
	SUBMISSION: {SUBMISSION_VALUE},
}

type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "SYNCED"
	SyncStatusSyncing  SyncStatus = "SYNCING"
	SyncStatusDesynced SyncStatus = "DESYNCED"
)

type Objective struct {
	Id             int             `gorm:"primaryKey"`
	Name           string          `gorm:"not null"`
	Extra          string          `gorm:"null"`
	RequiredAmount int             `gorm:"not null"`
	Conditions     []*Condition    `gorm:"foreignKey:ObjectiveId;constraint:OnDelete:CASCADE"`
	CategoryId     int             `gorm:"not null"`
	ObjectiveType  ObjectiveType   `gorm:"not null"`
	NumberField    NumberField     `gorm:"not null"`
	Aggregation    AggregationType `gorm:"not null"`
	ValidFrom      *time.Time      `gorm:"null"`
	ValidTo        *time.Time      `gorm:"null"`
	ScoringId      *int            `gorm:"null;references:scoring_presets(id)"`
	ScoringPreset  *ScoringPreset  `gorm:"foreignKey:ScoringId;references:Id"`
	SyncStatus     SyncStatus
}

type ObjectiveRepository struct {
	DB *gorm.DB
}

func NewObjectiveRepository() *ObjectiveRepository {
	return &ObjectiveRepository{DB: config.DatabaseConnection()}
}

func (r *ObjectiveRepository) SaveObjective(objective *Objective) (*Objective, error) {
	objective.SyncStatus = SyncStatusDesynced
	result := r.DB.Save(objective)
	if result.Error != nil {
		return nil, result.Error
	}
	return objective, nil
}

func (r *ObjectiveRepository) GetObjectiveById(objectiveId int, preloads ...string) (*Objective, error) {
	var objective Objective
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.First(&objective, "id = ?", objectiveId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &objective, nil
}

func (r *ObjectiveRepository) DeleteObjective(objectiveId int) error {
	result := r.DB.Delete(&Objective{Id: objectiveId})
	return result.Error
}

func (r *ObjectiveRepository) RemoveScoringId(scoringId int) error {
	result := r.DB.Model(&Objective{}).Where(Objective{ScoringId: &scoringId}).Update("scoring_id", nil)
	return result.Error
}

func (r *ObjectiveRepository) StartSync(objectiveIds []int) error {
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusDesynced).
		Update("sync_status", SyncStatusSyncing)
	return result.Error
}

func (r *ObjectiveRepository) FinishSync(objectiveIds []int) error {
	if len(objectiveIds) == 0 {
		return nil
	}
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusSyncing).
		Update("sync_status", SyncStatusSynced)
	return result.Error
}
