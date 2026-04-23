package config

import (
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// var enumQueries = []string{
// 	`CREATE TYPE bpl2.scoring_rule AS ENUM ('FIXED_POINTS_ON_COMPLETION', 'POINTS_BY_VALUE', 'RANK_BY_COMPLETION_TIME', 'RANK_BY_HIGHEST_VALUE', 'RANK_BY_LOWEST_VALUE', 'RANK_BY_CHILD_COMPLETION_TIME', 'BONUS_PER_CHILD_COMPLETION', 'BINGO_BOARD_RANKING', 'RANK_BY_CHILD_VALUE_SUM')`,
// 	`CREATE TYPE bpl2.objective_type AS ENUM ('ITEM', 'PLAYER', 'SUBMISSION')`,
// 	`CREATE TYPE bpl2.operator AS ENUM ('EQ', 'NEQ', 'GT', 'GTE', 'LT', 'LTE', 'IN', 'NOT_IN', 'MATCHES', 'CONTAINS', 'CONTAINS_ALL', 'CONTAINS_MATCH', 'CONTAINS_ALL_MATCHES')`,
// 	`CREATE TYPE bpl2.scoring_rule_type AS ENUM ('OBJECTIVE', 'CATEGORY')`,
// 	`CREATE TYPE bpl2.item_field AS ENUM ('BASE_TYPE', 'NAME', 'TYPE_LINE', 'RARITY', 'ILVL', 'FRAME_TYPE', 'TALISMAN_TIER', 'ENCHANT_MODS', 'EXPLICIT_MODS', 'IMPLICIT_MODS', 'CRAFTED_MODS', 'FRACTURED_MODS', 'SIX_LINK')`,
// 	`CREATE TYPE bpl2.tracked_value AS ENUM ('STACK_SIZE', 'CHARACTER_LEVEL', 'SUBMITTED_VALUE')`,
// 	`CREATE TYPE bpl2.approval_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED')`,
// }

var (
	db   *gorm.DB
	once sync.Once
)

func DatabaseConnection() *gorm.DB {
	return db
}

func InitDB(host string, port string, user string, password string, dbname string) (*gorm.DB, error) {
	var err error
	once.Do(func() {
		sqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=bpl2",
			host, port, user, password, dbname)
		db, err = gorm.Open(postgres.Open(sqlInfo), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "bpl2.",
				SingularTable: false,
			},
			Logger: logger.Default.LogMode(logger.Silent),
			// Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return
		}

	})

	return db, err
}
