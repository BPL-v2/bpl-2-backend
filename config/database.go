package config

import (
	model "bpl/repository"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var enumQueries = []string{
	`CREATE TYPE bpl2.scoring_method AS ENUM ('PRESENCE', 'POINTS_FROM_VALUE', 'RANKED_TIME', 'RANKED_VALUE', 'RANKED_REVERSE', 'RANKED_COMPLETION_TIME', 'BONUS_PER_COMPLETION')`,
	`CREATE TYPE bpl2.objective_type AS ENUM ('ITEM', 'PLAYER', 'SUBMISSION')`,
	`CREATE TYPE bpl2.operator AS ENUM ('EQ', 'NEQ', 'GT', 'GTE', 'LT', 'LTE', 'IN', 'NOT_IN', 'MATCHES', 'CONTAINS', 'CONTAINS_ALL', 'CONTAINS_MATCH', 'CONTAINS_ALL_MATCHES')`,
	`CREATE TYPE bpl2.scoring_preset_type AS ENUM ('OBJECTIVE', 'CATEGORY')`,
	`CREATE TYPE bpl2.item_field AS ENUM ('BASE_TYPE', 'NAME', 'TYPE_LINE', 'RARITY', 'ILVL', 'FRAME_TYPE', 'TALISMAN_TIER', 'ENCHANT_MODS', 'EXPLICIT_MODS', 'IMPLICIT_MODS', 'CRAFTED_MODS', 'FRACTURED_MODS', 'SIX_LINK')`,
	`CREATE TYPE bpl2.number_field AS ENUM ('STACK_SIZE', 'PLAYER_LEVEL', 'PLAYER_XP', 'SUBMISSION_VALUE')`,
}

func DatabaseConnection() *gorm.DB {
	sqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := gorm.Open(postgres.Open(sqlInfo), &gorm.Config{})
	db.Exec("SET search_path TO bpl2")
	if err != nil {
		panic(err)
	}
	return db
}

func InitDB() (*gorm.DB, error) {
	sqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_PORT"), os.Getenv("DATABASE_USER"), os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_NAME"))
	db, err := gorm.Open(postgres.Open(sqlInfo), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "bpl2.",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Silent),
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	x := db.Exec(`CREATE SCHEMA IF NOT EXISTS bpl2`)
	if x.Error != nil {
		return nil, x.Error
	}
	for _, query := range enumQueries {
		x := db.Exec(query)
		if x.Error != nil {
			if strings.Contains(x.Error.Error(), "already exists") {
				continue
			}
			return nil, x.Error
		}
	}

	err = db.AutoMigrate(
		&model.ScoringCategory{},
		&model.Objective{},
		&model.Condition{},
		&model.Event{},
		&model.Team{},
		&model.User{},
		&model.StashChange{},
		&model.ObjectiveMatch{},
	)

	if err != nil {
		return nil, err
	}
	db.Exec("SET search_path TO bpl2")
	return db, nil
}
