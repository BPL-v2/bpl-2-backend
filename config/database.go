package config

import (
	model "bpl/repository"
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbName   = "postgres"
)

var enumQueries = []string{
	`CREATE TYPE bpl2.scoring_method_type AS ENUM ('PRESENCE', 'RANKED', 'RELATIVE_PRESENCE')`,
	`CREATE TYPE bpl2.scoring_method_inheritance AS ENUM ('OVERWRITE', 'INHERIT', 'EXTEND')`,
	`CREATE TYPE bpl2.objective_type AS ENUM ('ITEM')`,
	`CREATE TYPE bpl2.operator AS ENUM ('EQ', 'NEQ', 'GT', 'GTE', 'LT', 'LTE', 'IN', 'NOT_IN', 'MATCHES', 'CONTAINS', 'CONTAINS_ALL', 'CONTAINS_MATCH', 'CONTAINS_ALL_MATCHES')`,
	`CREATE TYPE bpl2.item_field AS ENUM ('BASE_TYPE', 'NAME', 'TYPE_LINE', 'RARITY', 'ILVL', 'FRAME_TYPE', 'TALISMAN_TIER', 'ENCHANT_MODS', 'EXPLICIT_MODS', 'IMPLICIT_MODS', 'CRAFTED_MODS', 'FRACTURED_MODS', 'SIX_LINK')`,
}

func DatabaseConnection() *gorm.DB {
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbName)

	db, err := gorm.Open(postgres.Open(sqlInfo), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func InitDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "bpl2.",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Silent),
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
		&model.ScoringMethod{},
		&model.Event{},
		&model.Team{},
		&model.User{},
	)

	if err != nil {
		return nil, err
	}
	return db, nil
}
