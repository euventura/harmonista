package site

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"harmonista/models"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&models.User{}, &models.Blog{}, &models.Post{}, &models.Page{}, &models.Tag{}, &models.PostTag{})
	return db
}

func setupTestRouter(siteModule *SiteModule) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	siteModule.RegisterRoutes(router)
	return router
}

func createTestUser(db *gorm.DB) *models.User {
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}
	db.Create(user)
	return user
}

func createTestBlog(db *gorm.DB, userID int, isListReader bool) *models.Blog {
	blog := &models.Blog{
		UserID:       userID,
		Title:        "Test Blog",
		Description:  "Test Description",
		Subdomain:    "testblog",
		IsListReader: isListReader,
	}
	db.Create(blog)
	return blog
}

func createTestPost(db *gorm.DB, blogID int, draft bool) *models.Post {
	post := &models.Post{
		BlogID:    blogID,
		Title:     "Test Post",
		Slug:      "test-post",
		Content:   "# Test Content\n\nThis is a **test** post.",
		Draft:     draft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.Create(post)
	return post
}

func TestSiteRSS_Success(t *testing.T) {
	db := setupTestDB()
	siteModule := NewSiteModule(db, nil)
	router := setupTestRouter(siteModule)

	user := createTestUser(db)

	// Blog included in list reader
	blog1 := createTestBlog(db, user.ID, true)
	createTestPost(db, blog1.ID, false)

	// Blog NOT included in list reader
	blog2 := createTestBlog(db, user.ID, false)
	blog2.Subdomain = "otherblog"
	db.Save(blog2)
	createTestPost(db, blog2.ID, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rss", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<rss")
	assert.Contains(t, w.Body.String(), "<channel>")
	assert.Contains(t, w.Body.String(), "<title>Harmonista Reader</title>")

	// Should contain post from blog1
	assert.Contains(t, w.Body.String(), "<item>")
	assert.Contains(t, w.Body.String(), "<title>Test Post</title>")

	// Should NOT contain post from blog2 (logic check - assuming titles are same, but we can check count or specific content if we made them different)
	// Since titles are same, let's just verify we have 1 item if possible, or check logic.
	// Actually, let's make titles different to be sure.
}

func TestSiteRSS_Filtering(t *testing.T) {
	db := setupTestDB()
	siteModule := NewSiteModule(db, nil)
	router := setupTestRouter(siteModule)

	user := createTestUser(db)

	// Blog included in list reader
	blog1 := createTestBlog(db, user.ID, true)
	post1 := createTestPost(db, blog1.ID, false)
	post1.Title = "Included Post"
	db.Save(post1)

	// Blog NOT included in list reader
	blog2 := createTestBlog(db, user.ID, false)
	blog2.Subdomain = "excluded"
	db.Save(blog2)
	post2 := createTestPost(db, blog2.ID, false)
	post2.Title = "Excluded Post"
	db.Save(post2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rss", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Included Post")
	assert.NotContains(t, w.Body.String(), "Excluded Post")
}
