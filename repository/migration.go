package repository

import (
	"bpl/config"
	"fmt"

	"gorm.io/gorm"
)

func Migration() error {
	DB := config.DatabaseConnection()
	version := 0
	if err := DB.Raw("SELECT version FROM migrations").Scan(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Println("No migrations found, starting from version 0")
			version = 0
		} else {
			fmt.Println("Error fetching migration version:", err)
			return err
		}
	}
	if version > 15 {
		fmt.Println("Migration already applied, skipping")
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		fmt.Println("Starting migration to add category objectives")
		categories := make([]*ScoringCategory, 0)
		if err := tx.Find(&categories).Error; err != nil {
			fmt.Println("Error fetching categories:", err)
			return err
		}

		catIDMapping := make(map[int]int)
		eventIdMapping := make(map[int]int)
		for _, category := range categories {
			catObjective := Objective{
				ParentId:       category.ParentId,
				Name:           category.Name,
				EventId:        category.EventId,
				ScoringId:      category.ScoringId,
				RequiredAmount: 1,
				ObjectiveType:  ObjectiveTypeCategory,
				Aggregation:    AggregationTypeNone,
				NumberField:    NumberFieldFinishedObjectives,
				SyncStatus:     SyncStatusDesynced,
			}
			if err := tx.Create(&catObjective).Error; err != nil {
				fmt.Println("Error creating category objective:", err)
				return err
			}
			catIDMapping[category.Id] = catObjective.Id
			eventIdMapping[category.Id] = category.EventId
		}

		objectives := make([]*Objective, 0)
		if err := tx.Find(&objectives).Error; err != nil {
			fmt.Println("Error fetching objectives:", err)
			return err
		}
		for _, objective := range objectives {
			if objective.ParentId != nil {
				if newParentId, exists := catIDMapping[*objective.ParentId]; exists {
					objective.EventId = eventIdMapping[*objective.ParentId]
					objective.ParentId = &newParentId
				}
			}
		}
		if err := tx.Save(objectives).Error; err != nil {
			fmt.Println("Error updating objective with new parent ID:", err)
			return err
		}
		suggestions := make([]*TeamSuggestion, 0)
		if err := tx.Find(&suggestions).Error; err != nil {
			fmt.Println("Error fetching team suggestions:", err)
		}
		for _, suggestion := range suggestions {
			if !suggestion.IsObjective {
				if newParentId, exists := catIDMapping[suggestion.Id]; exists {
					suggestion.Id = newParentId
				}
			}
			if err := tx.Save(suggestion).Error; err != nil {
				fmt.Println("Error updating team suggestion with new parent ID:", err)
				return err
			}
		}

		tx.Exec("UPDATE migrations SET version = 16")
		fmt.Println("Migration completed successfully")
		return nil
	})
}
