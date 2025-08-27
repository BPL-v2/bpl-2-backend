package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if (len(os.Args) < 3) || (os.Args[1] != "up" && os.Args[1] != "down") {
		fmt.Println("Usage: migrate [up|down] [head|n]")
		return
	}

	direction := os.Args[1]
	value := os.Args[2]
	valint, _ := strconv.Atoi(value)
	if value != "head" && valint == 0 {
		fmt.Println("Invalid value")
		return
	}

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

	i := 0
	for {
		if direction == "down" {
			err = migrateDown(db, version)
			if err != nil {
				break
			}
			version--
		} else if direction == "up" {
			err = migrateUp(db, version)
			if err != nil {
				break
			}
			version++
		} else {
			fmt.Println("Invalid direction")
			break
		}
		i++
		if value != "head" {
			if i >= valint {
				break
			}
		}
	}
}

func migrateUp(db *sql.DB, version int) error {
	file, err := os.ReadFile(fmt.Sprintf("migrations/%d-%d.sql", version, version+1))
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

func migrateDown(db *sql.DB, version int) error {
	file, err := os.ReadFile(fmt.Sprintf("migrations/%d-%d.sql", version, version-1))
	if err != nil {
		fmt.Println("Cannot migrate further down")
		return err
	}
	_, err = db.Exec(string(file))
	if err != nil {
		fmt.Printf("error executing migration: %v", err)
		return err
	}
	_, err = db.Exec("UPDATE migrations SET version = $1", version-1)
	if err != nil {
		fmt.Printf("error updating migration version: %v", err)
		return err
	}
	fmt.Printf("Migrated to version %d\n", version-1)
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
