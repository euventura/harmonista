package database

import (
	"fmt"
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

// MigrateDataFromSQLite copia dados do SQLite para PostgreSQL se o PG estiver vazio.
// TODO: remover após migração completa
func MigrateDataFromSQLite(pgDb, sqliteDb *gorm.DB) error {
	// Verificar se o PostgreSQL já tem dados
	var userCount int64
	pgDb.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		log.Println("PostgreSQL already has data, skipping migration from SQLite")
		return nil
	}

	log.Println("PostgreSQL is empty, starting data migration from SQLite...")

	// Copiar na ordem correta (respeitando foreign keys)

	// 1. Users
	var users []models.User
	if err := sqliteDb.Find(&users).Error; err != nil {
		return fmt.Errorf("error reading users from SQLite: %w", err)
	}
	if len(users) > 0 {
		if err := pgDb.Create(&users).Error; err != nil {
			return fmt.Errorf("error inserting users into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d users", len(users))
	}

	// 2. Blogs
	var blogs []models.Blog
	if err := sqliteDb.Find(&blogs).Error; err != nil {
		return fmt.Errorf("error reading blogs from SQLite: %w", err)
	}
	if len(blogs) > 0 {
		if err := pgDb.Create(&blogs).Error; err != nil {
			return fmt.Errorf("error inserting blogs into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d blogs", len(blogs))
	}

	// 3. Posts (incluindo soft-deleted)
	var posts []models.Post
	if err := sqliteDb.Unscoped().Find(&posts).Error; err != nil {
		return fmt.Errorf("error reading posts from SQLite: %w", err)
	}
	if len(posts) > 0 {
		if err := pgDb.Create(&posts).Error; err != nil {
			return fmt.Errorf("error inserting posts into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d posts", len(posts))
	}

	// 4. Pages (incluindo soft-deleted)
	var pages []models.Page
	if err := sqliteDb.Unscoped().Find(&pages).Error; err != nil {
		return fmt.Errorf("error reading pages from SQLite: %w", err)
	}
	if len(pages) > 0 {
		if err := pgDb.Create(&pages).Error; err != nil {
			return fmt.Errorf("error inserting pages into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d pages", len(pages))
	}

	// 5. Tags
	var tags []models.Tag
	if err := sqliteDb.Find(&tags).Error; err != nil {
		return fmt.Errorf("error reading tags from SQLite: %w", err)
	}
	if len(tags) > 0 {
		if err := pgDb.Create(&tags).Error; err != nil {
			return fmt.Errorf("error inserting tags into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d tags", len(tags))
	}

	// 6. PostTags
	var postTags []models.PostTag
	if err := sqliteDb.Find(&postTags).Error; err != nil {
		return fmt.Errorf("error reading post_tags from SQLite: %w", err)
	}
	if len(postTags) > 0 {
		if err := pgDb.Create(&postTags).Error; err != nil {
			return fmt.Errorf("error inserting post_tags into PostgreSQL: %w", err)
		}
		log.Printf("Migrated %d post_tags", len(postTags))
	}

	// Resetar sequences do PostgreSQL para continuar do ID correto
	sequences := []struct {
		table  string
		column string
	}{
		{"users", "id"},
		{"blogs", "id"},
		{"posts", "id"},
		{"pages", "id"},
		{"tags", "id"},
		{"post_tags", "id"},
	}

	for _, seq := range sequences {
		query := fmt.Sprintf(
			"SELECT setval(pg_get_serial_sequence('%s', '%s'), COALESCE((SELECT MAX(%s) FROM %s), 0))",
			seq.table, seq.column, seq.column, seq.table,
		)
		if err := pgDb.Exec(query).Error; err != nil {
			log.Printf("Warning: failed to reset sequence for %s.%s: %v", seq.table, seq.column, err)
		}
	}

	log.Println("Migration from SQLite completed successfully")
	return nil
}
