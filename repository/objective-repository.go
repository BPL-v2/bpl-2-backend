package repository

import (
	"bpl/config"
	"bpl/utils"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ObjectiveType string

const (
	ObjectiveTypeItem       ObjectiveType = "ITEM"
	ObjectiveTypePlayer     ObjectiveType = "PLAYER"
	ObjectiveTypeTeam       ObjectiveType = "TEAM"
	ObjectiveTypeSubmission ObjectiveType = "SUBMISSION"
	ObjectiveTypeCategory   ObjectiveType = "CATEGORY"
)

type AggregationType string

const (
	AggregationTypeSumLatest         AggregationType = "SUM_LATEST"
	AggregationTypeEarliest          AggregationType = "EARLIEST"
	AggregationTypeEarliestFreshItem AggregationType = "EARLIEST_FRESH_ITEM"
	AggregationTypeMaximum           AggregationType = "MAXIMUM"
	AggregationTypeMinimum           AggregationType = "MINIMUM"
	AggregationTypeDifferenceBetween AggregationType = "DIFFERENCE_BETWEEN"
	AggregationTypeNone              AggregationType = "NONE"
)

type NumberField string

const (
	NumberFieldStackSize NumberField = "STACK_SIZE"

	NumberFieldPlayerLevel       NumberField = "PLAYER_LEVEL"
	NumberFieldDelveDepth        NumberField = "DELVE_DEPTH"
	NumberFieldDelveDepthPast100 NumberField = "DELVE_DEPTH_PAST_100"
	NumberFieldPantheon          NumberField = "PANTHEON"
	NumberFieldAscendancy        NumberField = "ASCENDANCY"
	NumberFieldPlayerScore       NumberField = "PLAYER_SCORE"

	NumberFieldSubmissionValue NumberField = "SUBMISSION_VALUE"

	NumberFieldFinishedObjectives NumberField = "FINISHED_OBJECTIVES"
)

var ObjectiveTypeToNumberFields = map[ObjectiveType][]NumberField{
	ObjectiveTypeItem:       {NumberFieldStackSize},
	ObjectiveTypePlayer:     {NumberFieldPlayerLevel, NumberFieldDelveDepth, NumberFieldDelveDepthPast100, NumberFieldPantheon, NumberFieldAscendancy, NumberFieldPlayerScore},
	ObjectiveTypeTeam:       {NumberFieldPlayerLevel, NumberFieldDelveDepth, NumberFieldDelveDepthPast100, NumberFieldPantheon, NumberFieldAscendancy, NumberFieldPlayerScore},
	ObjectiveTypeSubmission: {NumberFieldSubmissionValue},
	ObjectiveTypeCategory:   {NumberFieldFinishedObjectives},
}

type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "SYNCED"
	SyncStatusSyncing  SyncStatus = "SYNCING"
	SyncStatusDesynced SyncStatus = "DESYNCED"
)

type Objective struct {
	Id                     int             `gorm:"primaryKey"`
	Name                   string          `gorm:"not null"`
	Extra                  string          `gorm:"null"`
	RequiredAmount         int             `gorm:"not null"`
	Conditions             Conditions      `gorm:"type:jsonb"`
	ParentId               *int            `gorm:"null"`
	EventId                int             `gorm:"not null;references:events(id)"`
	ObjectiveType          ObjectiveType   `gorm:"not null"`
	NumberField            NumberField     `gorm:"not null"`
	Aggregation            AggregationType `gorm:"not null"`
	ValidFrom              *time.Time      `gorm:"null"`
	ValidTo                *time.Time      `gorm:"null"`
	ScoringId              *int            `gorm:"null;references:scoring_presets(id)"`
	HideProgress           bool            `gorm:"not null;default:false"`
	ScoringPreset          *ScoringPreset  `gorm:"foreignKey:ScoringId;references:Id"`
	SyncStatus             SyncStatus      `gorm:"not null;default:DESYNCED"`
	NumberFieldExplanation *string         `gorm:"null"`
	Children               []*Objective    `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
}

func (o *Objective) FlatMap() []*Objective {
	result := []*Objective{o}
	for _, child := range o.Children {
		result = append(result, child.FlatMap()...)
	}
	return result
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

func (r *ObjectiveRepository) DeleteObjectivesByEventId(eventId int) error {
	result := r.DB.Where("event_id = ?", eventId).Delete(&Objective{})
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

func (r *ObjectiveRepository) GetObjectivesByEventId(eventId int, preloads ...string) (*Objective, error) {
	objectives, err := r.GetObjectivesByEventIdFlat(eventId, preloads...)
	if err != nil {
		return nil, fmt.Errorf("failed to get objectives for event %d: %w", eventId, err)
	}
	idMap := make(map[int]*Objective)
	for _, objective := range objectives {
		idMap[objective.Id] = objective
	}
	for _, objective := range objectives {
		if objective.ParentId != nil {
			parent, exists := idMap[*objective.ParentId]
			if exists {
				parent.Children = append(parent.Children, objective)
			}
		}
	}
	rootObjective, found := utils.FindFirst(objectives, func(o *Objective) bool {
		return o.ParentId == nil
	})
	if !found {
		return nil, fmt.Errorf("no root objective found for event %d", eventId)
	}
	return rootObjective, nil
}

func (r *ObjectiveRepository) GetObjectivesByEventIdFlat(eventId int, preloads ...string) ([]*Objective, error) {
	var objectives []*Objective
	query := r.DB.Model(&Objective{}).Where("event_id = ?", eventId)
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&objectives)
	if result.Error != nil {
		return nil, result.Error
	}
	return objectives, nil
}

func (r *ObjectiveRepository) GetAllObjectives(preloads ...string) ([]*Objective, error) {
	var objectives []*Objective
	query := r.DB.Model(&Objective{})
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&objectives)
	if result.Error != nil {
		return nil, result.Error
	}
	return objectives, nil
}
