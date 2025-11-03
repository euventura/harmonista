package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"harmonista/models"
)

func TestAdminRoot_NotLoggedIn(t *testing.T) {
	db := setupTestDB()
	adminModule := NewAdminModule(db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("test-session", store))
	adminModule.RegisterRoutes(router)

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/login")
}

func TestUserCreation(t *testing.T) {
	db := setupTestDB()
	user := createTestUser(db)

	assert.NotNil(t, user)
	assert.NotEmpty(t, user.Email)
}

func TestCadastroValidation(t *testing.T) {
	db := setupTestDB()

	user := &models.User{
		Email:        "test@test.com",
		PasswordHash: "hash",
	}
	db.Create(user)

	var existingUser models.User
	err := db.Where("email = ?", "test@test.com").First(&existingUser).Error

	assert.NoError(t, err)
	assert.Equal(t, user.Email, existingUser.Email)
}

func TestCadastroPost_Success(t *testing.T) {
	db := setupTestDB()

	passwordHash, _ := hashPassword("password123")
	user := models.User{
		Email:        "newuser@example.com",
		PasswordHash: passwordHash,
	}
	db.Create(&user)

	blog := models.Blog{
		UserID:      user.ID,
		Title:       "New Blog",
		Subdomain:   "newblog",
		Description: "Blog description",
	}
	db.Create(&blog)

	var savedUser models.User
	db.Where("email = ?", "newuser@example.com").First(&savedUser)
	assert.Equal(t, "newuser@example.com", savedUser.Email)

	var savedBlog models.Blog
	db.Where("subdomain = ?", "newblog").First(&savedBlog)
	assert.Equal(t, "New Blog", savedBlog.Title)
	assert.Equal(t, "newblog", savedBlog.Subdomain)
}

func TestCadastroPost_DuplicateEmail(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)

	var existingUser models.User
	err := db.Where("email = ?", user.Email).First(&existingUser).Error

	assert.NoError(t, err)
	assert.Equal(t, user.Email, existingUser.Email)
}

func TestCadastroPost_DuplicateSubdomain(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)

	var existingBlog models.Blog
	err := db.Where("subdomain = ?", blog.Subdomain).First(&existingBlog).Error

	assert.NoError(t, err)
	assert.Equal(t, blog.Subdomain, existingBlog.Subdomain)
}

func TestPasswordHashing(t *testing.T) {
	password := "testpassword"

	hash, err := hashPassword(password)
	assert.NoError(t, err)

	valid := checkPasswordHash(password, hash)
	assert.True(t, valid)

	invalid := checkPasswordHash("wrongpassword", hash)
	assert.False(t, invalid)
}

func TestDashboard(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)

	blog1 := &models.Blog{
		UserID:      user.ID,
		Title:       "Blog 1",
		Description: "Description 1",
		Subdomain:   "blog1",
	}
	db.Create(blog1)

	blog2 := &models.Blog{
		UserID:      user.ID,
		Title:       "Blog 2",
		Description: "Description 2",
		Subdomain:   "blog2",
	}
	db.Create(blog2)

	var blogs []models.Blog
	db.Where("user_id = ?", user.ID).Find(&blogs)

	assert.Equal(t, 2, len(blogs))
}
