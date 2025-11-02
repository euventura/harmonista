package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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

	db.AutoMigrate(&models.User{}, &models.Blog{}, &models.Post{}, &models.Tag{}, &models.PostTag{})
	return db
}

func setupTestRouter(adminModule *AdminModule) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("test-session", store))
	adminModule.RegisterRoutes(router)
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

func createTestBlog(db *gorm.DB, userID int) *models.Blog {
	blog := &models.Blog{
		UserID:      userID,
		Title:       "Test Blog",
		Description: "Test Description",
		Subdomain:   "testblog",
	}
	db.Create(blog)
	return blog
}

func createTestPost(db *gorm.DB, blogID int) *models.Post {
	post := &models.Post{
		BlogID:    blogID,
		Title:     "Test Post",
		Slug:      "test-post",
		Content:   "Test content",
		Draft:     true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.Create(post)
	return post
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Testing 123", "testing-123"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special@#Characters!", "specialcharacters"},
		{"---Dashes---", "dashes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequireAuth_Unauthorized(t *testing.T) {
	db := setupTestDB()
	adminModule := NewAdminModule(db)
	router := setupTestRouter(adminModule)

	req, _ := http.NewRequest("GET", "/admin/testblog/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/login")
}

func TestIndex_BlogNotFound(t *testing.T) {
	db := setupTestDB()
	adminModule := NewAdminModule(db)
	router := setupTestRouter(adminModule)

	createTestUser(db)

	req, _ := http.NewRequest("GET", "/admin/nonexistent/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
}

func TestUpdateBlogSettings(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)

	var updatedBlog models.Blog
	db.First(&updatedBlog, blog.ID)

	updatedBlog.Title = "Updated Title"
	updatedBlog.Description = "Updated Description"
	db.Save(&updatedBlog)

	db.First(&updatedBlog, blog.ID)
	assert.Equal(t, "Updated Title", updatedBlog.Title)
	assert.Equal(t, "Updated Description", updatedBlog.Description)
}

func TestListPosts(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	createTestPost(db, blog.ID)
	createTestPost(db, blog.ID)

	var posts []models.Post
	db.Where("blog_id = ?", blog.ID).Find(&posts)

	assert.Equal(t, 2, len(posts))
}

func TestSavePost(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)

	post := models.Post{
		BlogID:    blog.ID,
		Title:     "New Post",
		Slug:      "new-post",
		Content:   "Post content",
		Draft:     true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.Create(&post)

	var savedPost models.Post
	db.Where("blog_id = ?", blog.ID).First(&savedPost)
	assert.Equal(t, "New Post", savedPost.Title)
	assert.Equal(t, "new-post", savedPost.Slug)
	assert.True(t, savedPost.Draft)
}

func TestUpdatePost(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID)

	post.Title = "Updated Post"
	post.Content = "Updated content"
	post.Draft = false
	db.Save(&post)

	var updatedPost models.Post
	db.First(&updatedPost, post.ID)
	assert.Equal(t, "Updated Post", updatedPost.Title)
	assert.False(t, updatedPost.Draft)
}

func TestDeletePost(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID)

	db.Delete(&post)

	var deletedPost models.Post
	result := db.First(&deletedPost, post.ID)
	assert.Error(t, result.Error)
}

func TestCreateOrAssignTag(t *testing.T) {
	db := setupTestDB()
	adminModule := NewAdminModule(db)

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID)

	err := adminModule.createOrAssignTag(blog.ID, int(post.ID), "Technology")
	assert.NoError(t, err)

	var tag models.Tag
	db.Where("title = ?", "Technology").First(&tag)
	assert.Equal(t, "Technology", tag.Title)

	err = adminModule.createOrAssignTag(blog.ID, int(post.ID), "Technology")
	assert.NoError(t, err)

	var tagCount int64
	db.Model(&models.Tag{}).Where("title = ?", "Technology").Count(&tagCount)
	assert.Equal(t, int64(1), tagCount)
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := hashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "testpassword123"
	hash, _ := hashPassword(password)

	assert.True(t, checkPasswordHash(password, hash))
	assert.False(t, checkPasswordHash("wrongpassword", hash))
}

func TestGenerateToken(t *testing.T) {
	token1, err1 := generateToken()
	token2, err2 := generateToken()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2)
}

func TestProcessPostTags(t *testing.T) {
	db := setupTestDB()
	adminModule := NewAdminModule(db)

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID)

	err := adminModule.processPostTags(blog.ID, int(post.ID), "Go, Programming, Web Development")
	assert.NoError(t, err)

	var tags []models.Tag
	db.Find(&tags)
	assert.Equal(t, 3, len(tags))

	var postTags []models.PostTag
	db.Where("post_id = ?", post.ID).Find(&postTags)
	assert.Equal(t, 3, len(postTags))

	err = adminModule.processPostTags(blog.ID, int(post.ID), "Go, Testing")
	assert.NoError(t, err)

	db.Where("post_id = ?", post.ID).Find(&postTags)
	assert.Equal(t, 2, len(postTags))

	db.Find(&tags)
	assert.Equal(t, 4, len(tags))
}
