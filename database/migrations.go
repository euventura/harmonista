package database

import (
	"log"

	"harmonista/models"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Blog{},
		&models.Post{},
		&models.Page{},
		&models.Tag{},
		&models.PostTag{},
	)

	if err != nil {
		log.Printf("Error running migrations: %v", err)
		return err
	}

	log.Println("Migrations completed successfully")
	return nil
}
