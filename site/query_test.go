package site

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"harmonista/models"
)

func TestListReaderQuery(t *testing.T) {
	// Setup in-memory SQLite
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate schemas
	err = db.AutoMigrate(&models.Blog{}, &models.Post{})
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Seed data
	blog1 := models.Blog{Subdomain: "blog1", IsListReader: true}
	blog2 := models.Blog{Subdomain: "blog2", IsListReader: true}
	db.Create(&blog1)
	db.Create(&blog2)

	// Blog 1 posts
	db.Create(&models.Post{BlogID: blog1.ID, Title: "B1 P1", Draft: false, CreatedAt: time.Now().Add(-2 * time.Hour)})
	db.Create(&models.Post{BlogID: blog1.ID, Title: "B1 P2", Draft: false, CreatedAt: time.Now().Add(-1 * time.Hour)}) // Should be picked

	// Blog 2 posts
	db.Create(&models.Post{BlogID: blog2.ID, Title: "B2 P1", Draft: false, CreatedAt: time.Now().Add(-3 * time.Hour)}) // Should be picked

	// Execute query logic from site.go
	var rawPosts []models.Post
	
	subQuery := db.Table("posts").
		Select("blog_id, MAX(created_at) as max_created_at").
		Where("draft = ?", false).
		Group("blog_id")

	err = db.Table("posts").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Joins("INNER JOIN (?) as latest ON posts.blog_id = latest.blog_id AND posts.created_at = latest.max_created_at", subQuery).
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Order("posts.created_at DESC").
		Find(&rawPosts).Error

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(rawPosts) != 2 {
		t.Errorf("Expected 2 posts, got %d", len(rawPosts))
	}

	for _, p := range rawPosts {
		if p.Title == "B1 P1" {
			t.Error("Got old post B1 P1, expected B1 P2")
		}
	}
}
