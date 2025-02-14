package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

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
	`CREATE TYPE bpl2.approval_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED')`,
}

var (
	db   *gorm.DB
	once sync.Once
)

func DatabaseConnection() *gorm.DB {
	return db
}

func InitDB() (*gorm.DB, error) {
	var err error
	once.Do(func() {
		sqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=bpl2",
			os.Getenv("DATABASE_HOST"),
			os.Getenv("DATABASE_PORT"),
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("DATABASE_NAME"))

		db, err = gorm.Open(postgres.Open(sqlInfo), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "bpl2.",
				SingularTable: false,
			},
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return
		}

		x := db.Exec(`CREATE SCHEMA IF NOT EXISTS bpl2`)
		if x.Error != nil {
			err = x.Error
			return
		}
		for _, query := range enumQueries {
			x := db.Exec(query)
			if x.Error != nil {
				if strings.Contains(x.Error.Error(), "already exists") {
					continue
				}
				err = x.Error
				return
			}
		}
	})

	return db, err
}
