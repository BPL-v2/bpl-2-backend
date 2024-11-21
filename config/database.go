package config

import (
	model "bpl/model/dbmodel"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbName   = "postgres"
)

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
		// Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "bpl2.", // schema name
			SingularTable: false,
		}})
	if err != nil {
		return nil, err
	}
	x := db.Exec(`CREATE SCHEMA IF NOT EXISTS bpl2`)
	if x.Error != nil {
		return nil, x.Error
	}
	err = db.AutoMigrate(&model.Objective{}, &model.Condition{}, &model.ScoringCategory{}, &model.ScoringMethod{}, &model.Event{})

	if err != nil {
		return nil, err
	}
	return db, nil
}
