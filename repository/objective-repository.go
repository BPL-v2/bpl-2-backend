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
	ObjectiveTypeStashTab   ObjectiveType = "STASH_TAB"
	ObjectiveTypePlayer     ObjectiveType = "PLAYER"
	ObjectiveTypeTeam       ObjectiveType = "TEAM"
	ObjectiveTypeSubmission ObjectiveType = "SUBMISSION"
	ObjectiveTypeCategory   ObjectiveType = "CATEGORY"
)

type AggregationType string

const (
	AggregationTypeSumLatest         AggregationType = "SUM_LATEST"
	AggregationTypeLatest            AggregationType = "LATEST"
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

	NumberFieldFossilFuel NumberField = "FOSSIL_FUEL"

	NumberFieldPlayerLevel               NumberField = "PLAYER_LEVEL"
	NumberFieldDelveDepth                NumberField = "DELVE_DEPTH"
	NumberFieldDelveDepthPast100         NumberField = "DELVE_DEPTH_PAST_100"
	NumberFieldProgressiveDelveDepth     NumberField = "PROGRESSIVE_DELVE_DEPTH"
	NumberFieldPantheon                  NumberField = "PANTHEON"
	NumberFieldAscendancy                NumberField = "ASCENDANCY"
	NumberFieldFullyAscended             NumberField = "FULLY_ASCENDED"
	NumberFieldBloodlineAscendancyPoints NumberField = "BLOODLINE_ASCENDANCY_POINTS"
	NumberFieldBloodlineAscendancy       NumberField = "BLOODLINE_ASCENDANCY"
	NumberFieldPlayerScore               NumberField = "PLAYER_SCORE"
	NumberFieldHasRareAscendancyPast90   NumberField = "HAS_RARE_ASCENDANCY_PAST_90"
	NumberFieldVoidStones                NumberField = "VOID_STONES"

	NumberFieldWeaponQuality NumberField = "WEAPON_QUALITY"
	NumberFieldArmourQuality NumberField = "ARMOUR_QUALITY"
	NumberFieldFlaskQuality  NumberField = "FLASK_QUALITY"

	NumberFieldEvasion          NumberField = "EVASION"
	NumberFieldEnergyShield     NumberField = "ENERGY_SHIELD"
	NumberFieldArmour           NumberField = "ARMOUR"
	NumberFieldHP               NumberField = "HP"
	NumberFieldMana             NumberField = "MANA"
	NumberFieldFullDPS          NumberField = "FULL_DPS"
	NumberFieldEHP              NumberField = "EHP"
	NumberFieldIncMovementSpeed NumberField = "INC_MOVEMENT_SPEED"
	NumberFieldPhysMaxHit       NumberField = "PHYS_MAX_HIT"
	NumberFieldEleMaxHit        NumberField = "ELE_MAX_HIT"
	NumberFieldAttackBlock      NumberField = "ATTACK_BLOCK"
	NumberFieldSpellBlock       NumberField = "SPELL_BLOCK"
	NumberFieldHighIlvlFlasks   NumberField = "HIGH_ILVL_FLASKS"
	NumberFieldEleMaxRes        NumberField = "ELE_MAX_RES"

	NumberFieldAtlasPoints NumberField = "ATLAS_POINTS"

	NumberFieldInfluenceEquipped           NumberField = "INFLUENCE_EQUIPPED"
	NumberFieldFoulbornEquipped            NumberField = "FOULBORN_EQUIPPED"
	NumberFieldGemsEquipped                NumberField = "GEMS_EQUIPPED"
	NumberFieldCorruptedItemsEquipped      NumberField = "CORRUPTED_ITEMS_EQUIPPED"
	NumberFieldJewelsWithImplicitsEquipped NumberField = "JEWELS_WITH_IMPLICITS_EQUIPPED"
	NumberFieldEnchantedItemsEquipped      NumberField = "ENCHANTED_ITEMS_EQUIPPED"

	NumberFieldSubmissionValue NumberField = "SUBMISSION_VALUE"

	NumberFieldFinishedObjectives NumberField = "FINISHED_OBJECTIVES"
)

var ObjectiveTypeToNumberFields = map[ObjectiveType][]NumberField{
	ObjectiveTypeItem:     {NumberFieldStackSize},
	ObjectiveTypeStashTab: {NumberFieldFossilFuel},
	ObjectiveTypePlayer: {
		NumberFieldPlayerLevel,
		NumberFieldDelveDepth,
		NumberFieldDelveDepthPast100,
		NumberFieldProgressiveDelveDepth,
		NumberFieldPantheon,
		NumberFieldAscendancy,
		NumberFieldBloodlineAscendancy,
		NumberFieldBloodlineAscendancyPoints,
		NumberFieldFullyAscended,
		NumberFieldPlayerScore,
		NumberFieldWeaponQuality,
		NumberFieldArmourQuality,
		NumberFieldFlaskQuality,
		NumberFieldEvasion,
		NumberFieldEnergyShield,
		NumberFieldArmour,
		NumberFieldHP,
		NumberFieldMana,
		NumberFieldFullDPS,
		NumberFieldEHP,
		NumberFieldIncMovementSpeed,
		NumberFieldPhysMaxHit,
		NumberFieldEleMaxHit,
		NumberFieldAtlasPoints,
		NumberFieldInfluenceEquipped,
		NumberFieldFoulbornEquipped,
		NumberFieldGemsEquipped,
		NumberFieldCorruptedItemsEquipped,
		NumberFieldJewelsWithImplicitsEquipped,
		NumberFieldHasRareAscendancyPast90,
		NumberFieldEnchantedItemsEquipped,
	},
	ObjectiveTypeTeam: {
		NumberFieldPlayerLevel,
		NumberFieldDelveDepth,
		NumberFieldDelveDepthPast100,
		NumberFieldProgressiveDelveDepth,
		NumberFieldPantheon,
		NumberFieldAscendancy,
		NumberFieldBloodlineAscendancy,
		NumberFieldBloodlineAscendancyPoints,
		NumberFieldFullyAscended,
		NumberFieldPlayerScore,
		NumberFieldWeaponQuality,
		NumberFieldArmourQuality,
		NumberFieldFlaskQuality,
		NumberFieldEvasion,
		NumberFieldEnergyShield,
		NumberFieldArmour,
		NumberFieldHP,
		NumberFieldMana,
		NumberFieldFullDPS,
		NumberFieldEHP,
		NumberFieldIncMovementSpeed,
		NumberFieldPhysMaxHit,
		NumberFieldEleMaxHit,
		NumberFieldAtlasPoints,
		NumberFieldInfluenceEquipped,
		NumberFieldFoulbornEquipped,
		NumberFieldGemsEquipped,
		NumberFieldCorruptedItemsEquipped,
		NumberFieldJewelsWithImplicitsEquipped,
		NumberFieldHasRareAscendancyPast90,
		NumberFieldEnchantedItemsEquipped,
	},
	ObjectiveTypeSubmission: {NumberFieldSubmissionValue},
	ObjectiveTypeCategory:   {NumberFieldFinishedObjectives},
}

type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "SYNCED"
	SyncStatusSyncing  SyncStatus = "SYNCING"
	SyncStatusDesynced SyncStatus = "DESYNCED"
)

type ObjectiveScoringPreset struct {
	ObjectiveId     int `gorm:"primaryKey"`
	ScoringPresetId int `gorm:"primaryKey"`
}

type Objective struct {
	Id                     int              `gorm:"primaryKey"`
	Name                   string           `gorm:"not null"`
	Extra                  string           `gorm:"null"`
	RequiredAmount         int              `gorm:"not null"`
	Conditions             Conditions       `gorm:"type:jsonb"`
	ParentId               *int             `gorm:"null"`
	EventId                int              `gorm:"not null;references:events(id)"`
	ObjectiveType          ObjectiveType    `gorm:"not null"`
	NumberField            NumberField      `gorm:"not null"`
	Aggregation            AggregationType  `gorm:"not null"`
	ValidFrom              *time.Time       `gorm:"null"`
	ValidTo                *time.Time       `gorm:"null"`
	ScoringPresets         []*ScoringPreset `gorm:"many2many:objective_scoring_presets;joinForeignKey:objective_id;joinReferences:scoring_preset_id"`
	HideProgress           bool             `gorm:"not null;default:false"`
	SyncStatus             SyncStatus       `gorm:"not null;default:DESYNCED"`
	NumberFieldExplanation *string          `gorm:"null"`
	Children               []*Objective     `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
}

func (o *Objective) FlatMap() []*Objective {
	result := []*Objective{o}
	for _, child := range o.Children {
		result = append(result, child.FlatMap()...)
	}
	return result
}

type ObjectiveRepository interface {
	SaveObjective(objective *Objective) (*Objective, error)
	SaveObjectives(objectives []*Objective) ([]*Objective, error)
	GetObjectiveById(objectiveId int, preloads ...string) (*Objective, error)
	DeleteObjective(objectiveId int) error
	DeleteObjectivesByEventId(eventId int) error
	RemoveScoringPreset(scoringId int) error
	AssociateScoringPresets(objectiveId int, presetIds []int) error
	StartSync(objectiveIds []int) error
	FinishSync(objectiveIds []int) error
	GetObjectivesByEventId(eventId int, preloads ...string) (*Objective, error)
	GetObjectivesByEventIdFlat(eventId int, preloads ...string) ([]*Objective, error)
	GetAllObjectives(preloads ...string) ([]*Objective, error)
}

type ObjectiveRepositoryImpl struct {
	DB *gorm.DB
}

func NewObjectiveRepository() ObjectiveRepository {
	return &ObjectiveRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ObjectiveRepositoryImpl) SaveObjective(objective *Objective) (*Objective, error) {
	objective.SyncStatus = SyncStatusDesynced
	result := r.DB.Save(objective)
	if result.Error != nil {
		return nil, result.Error
	}
	return objective, nil
}

func (r *ObjectiveRepositoryImpl) SaveObjectives(objectives []*Objective) ([]*Objective, error) {
	result := r.DB.Save(objectives)
	return objectives, result.Error
}

func (r *ObjectiveRepositoryImpl) GetObjectiveById(objectiveId int, preloads ...string) (*Objective, error) {
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

func (r *ObjectiveRepositoryImpl) DeleteObjective(objectiveId int) error {
	result := r.DB.Delete(&Objective{Id: objectiveId})
	return result.Error
}

func (r *ObjectiveRepositoryImpl) DeleteObjectivesByEventId(eventId int) error {
	result := r.DB.Where("event_id = ?", eventId).Delete(&Objective{})
	return result.Error
}

func (r *ObjectiveRepositoryImpl) RemoveScoringPreset(scoringId int) error {
	// Remove all associations between objectives and this scoring preset
	return r.DB.Where("scoring_preset_id = ?", scoringId).Delete(&ObjectiveScoringPreset{}).Error
}

func (r *ObjectiveRepositoryImpl) AssociateScoringPresets(objectiveId int, presetIds []int) error {
	if len(presetIds) == 0 {
		return nil
	}
	err := r.DB.Where("objective_id = ?", objectiveId).Delete(&ObjectiveScoringPreset{}).Error
	if err != nil {
		return err
	}
	if len(presetIds) == 0 {
		return nil
	}
	associations := make([]ObjectiveScoringPreset, len(presetIds))
	for i, presetId := range presetIds {
		associations[i] = ObjectiveScoringPreset{
			ObjectiveId:     objectiveId,
			ScoringPresetId: presetId,
		}
	}

	return r.DB.Create(&associations).Error
}

func (r *ObjectiveRepositoryImpl) StartSync(objectiveIds []int) error {
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusDesynced).
		Update("sync_status", SyncStatusSyncing)
	return result.Error
}

func (r *ObjectiveRepositoryImpl) FinishSync(objectiveIds []int) error {
	if len(objectiveIds) == 0 {
		return nil
	}
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusSyncing).
		Update("sync_status", SyncStatusSynced)
	return result.Error
}

func (r *ObjectiveRepositoryImpl) GetObjectivesByEventId(eventId int, preloads ...string) (*Objective, error) {
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

func (r *ObjectiveRepositoryImpl) GetObjectivesByEventIdFlat(eventId int, preloads ...string) ([]*Objective, error) {
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

func (r *ObjectiveRepositoryImpl) GetAllObjectives(preloads ...string) ([]*Objective, error) {
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
