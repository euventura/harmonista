package common

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ConnectPgDb conecta ao banco de dados PostgreSQL (banco principal)
func ConnectPgDb() *gorm.DB {
	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USER")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DBNAME")
	sslmode := os.Getenv("PG_SSLMODE")

	if host == "" || user == "" || dbname == "" {
		log.Println("PostgreSQL config incomplete (PG_HOST, PG_USER, PG_DBNAME required)")
		return nil
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Error connecting to PostgreSQL: " + err.Error())
		return nil
	}
	log.Printf("Connected to PostgreSQL (%s@%s:%s/%s)", user, host, port, dbname)
	return db
}

// ConnectDb conecta ao banco SQLite legado (usado para migração de dados)
func ConnectDb() *gorm.DB {
	var envFile map[string]string

	get := func(key string) string {
		if v, ok := envFile[key]; ok && v != "" {
			return v
		}
		return os.Getenv(key)
	}

	dbFile := get("sqlite_db")
	log.Println("attemptConnectDb: sqlite_db from env (raw):", dbFile)
	if dbFile == "" {
		log.Println("sqlite_db not set")
		return nil
	}

	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Println("Error opening sqlite db: " + err.Error())
		return nil
	}
	log.Println("opened sqlite db at:", dbFile)
	return db

}

// ConnectAnalyticsDb conecta ao banco de dados de analytics separado
func ConnectAnalyticsDb() *gorm.DB {
	analyticsDbFile := os.Getenv("analytics_db")
	log.Println("attemptConnectAnalyticsDb: analytics_db from env (raw):", analyticsDbFile)

	if analyticsDbFile == "" {
		log.Println("analytics_db not set - analytics will be disabled")
		return nil
	}

	db, err := gorm.Open(sqlite.Open(analyticsDbFile), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Println("Error opening analytics sqlite db: " + err.Error())
		return nil
	}

	log.Println("opened analytics sqlite db at:", analyticsDbFile)
	return db
}
