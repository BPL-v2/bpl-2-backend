package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=bpl2",
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	version, err := getMigrationVersion(db)
	if err != nil {
		log.Fatal(err)
	}

	for {
		version++
		err = migrateUp(db, version)
		if err != nil {
			break
		}
	}
}

func migrateUp(db *sql.DB, version int) error {
	file, err := os.ReadFile(fmt.Sprintf("migrations/%d.sql", version))
	if err != nil {
		fmt.Println("Cannot migrate further up")
		return err
	}
	_, err = db.Exec(string(file))
	if err != nil {
		fmt.Printf("error executing migration: %v", err)
		return err
	}
	_, err = db.Exec("UPDATE migrations SET version = $1", version+1)
	if err != nil {
		fmt.Printf("error updating migration version: %v", err)
		return err
	}
	fmt.Printf("Migrated to version %d\n", version+1)
	return nil
}

func getMigrationVersion(db *sql.DB) (version int, err error) {
	db.Exec("CREATE SCHEMA IF NOT EXISTS bpl2;")
	err = db.QueryRow("SELECT version FROM migrations").Scan(&version)
	if err != nil {
		err := generateMigrationTable(db)
		if err != nil {
			return 0, err
		}
		return 0, nil
	}
	return version, nil
}

func generateMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INT PRIMARY KEY
		);
		INSERT INTO migrations (version) VALUES (0);
	`)
	if err != nil {
		return err
	}
	return nil
}
