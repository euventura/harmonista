package common

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
