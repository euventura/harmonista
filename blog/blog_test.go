package blog

import (
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

	db.AutoMigrate(&models.User{}, &models.Blog{}, &models.Post{}, &models.Tag{}, &models.PostTag{})
	return db
}

func setupTestRouter(blogModule *BlogModule) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	blogModule.RegisterRoutes(router)
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

func TestGetBlogBySubdomain(t *testing.T) {
	db := setupTestDB()
	blogModule := NewBlogModule(db)

	user := createTestUser(db)
	expectedBlog := createTestBlog(db, user.ID)

	blog, err := blogModule.getBlogBySubdomain("testblog")

	assert.NoError(t, err)
	assert.Equal(t, expectedBlog.ID, blog.ID)
	assert.Equal(t, expectedBlog.Subdomain, blog.Subdomain)
}

func TestGetBlogBySubdomain_NotFound(t *testing.T) {
	db := setupTestDB()
	blogModule := NewBlogModule(db)

	blog, err := blogModule.getBlogBySubdomain("nonexistent")

	assert.Error(t, err)
	assert.Equal(t, 0, blog.ID)
}

func TestIndex_Success(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	createTestPost(db, blog.ID, false)
	createTestPost(db, blog.ID, false)

	var posts []models.Post
	db.Where("blog_id = ? AND draft = ?", blog.ID, false).Order("created_at DESC").Find(&posts)

	assert.Equal(t, 2, len(posts))
}

func TestIndex_OnlyPublishedPosts(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	createTestPost(db, blog.ID, false)
	createTestPost(db, blog.ID, true)

	var posts []models.Post
	db.Where("blog_id = ? AND draft = ?", blog.ID, false).Order("created_at DESC").Find(&posts)

	assert.Equal(t, 1, len(posts))
}

func TestPost_Success(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID, false)

	var retrievedPost models.Post
	err := db.Where("blog_id = ? AND slug = ? AND draft = ?", blog.ID, post.Slug, false).First(&retrievedPost).Error

	assert.NoError(t, err)
	assert.Equal(t, post.ID, retrievedPost.ID)
	assert.Equal(t, post.Title, retrievedPost.Title)
}

func TestPost_DraftNotVisible(t *testing.T) {
	db := setupTestDB()

	user := createTestUser(db)
	blog := createTestBlog(db, user.ID)
	post := createTestPost(db, blog.ID, true)

	var retrievedPost models.Post
	err := db.Where("blog_id = ? AND slug = ? AND draft = ?", blog.ID, post.Slug, false).First(&retrievedPost).Error

	assert.Error(t, err)
}

func TestRenderMarkdown_Headers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"# Header 1", "<h1>Header 1</h1>"},
		{"## Header 2", "<h2>Header 2</h2>"},
		{"### Header 3", "<h3>Header 3</h3>"},
		{"#### Header 4", "<h4>Header 4</h4>"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := renderMarkdown(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestRenderMarkdown_Lists(t *testing.T) {
	input := "- Item 1\n- Item 2\n- Item 3"
	result := renderMarkdown(input)

	assert.Contains(t, result, "<ul>")
	assert.Contains(t, result, "<li>Item 1</li>")
	assert.Contains(t, result, "<li>Item 2</li>")
	assert.Contains(t, result, "<li>Item 3</li>")
	assert.Contains(t, result, "</ul>")
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	input := "```\ncode here\n```"
	result := renderMarkdown(input)

	assert.Contains(t, result, "<pre><code>")
	assert.Contains(t, result, "code here")
	assert.Contains(t, result, "</code></pre>")
}

func TestReplaceBold(t *testing.T) {
	input := "This is **bold** text"
	expected := "This is <strong>bold</strong> text"
	result := replaceBold(input)

	assert.Equal(t, expected, result)
}

func TestReplaceItalic(t *testing.T) {
	input := "This is *italic* text"
	expected := "This is <em>italic</em> text"
	result := replaceItalic(input)

	assert.Equal(t, expected, result)
}

func TestReplaceLinks(t *testing.T) {
	input := "Check [this link](https://example.com)"
	expected := "Check <a href=\"https://example.com\">this link</a>"
	result := replaceLinks(input)

	assert.Equal(t, expected, result)
}

func TestReplaceCode(t *testing.T) {
	input := "This is `code` inline"
	expected := "This is <code>code</code> inline"
	result := replaceCode(input)

	assert.Equal(t, expected, result)
}

func TestFormatInlineMarkdown(t *testing.T) {
	input := "This is **bold** and *italic* and `code` and [link](https://example.com)"
	result := formatInlineMarkdown(input)

	assert.Contains(t, result, "<strong>bold</strong>")
	assert.Contains(t, result, "<em>italic</em>")
	assert.Contains(t, result, "<code>code</code>")
	assert.Contains(t, result, "<a href=\"https://example.com\">link</a>")
}

func TestRenderMarkdown_ComplexDocument(t *testing.T) {
	input := `# Main Title

This is a paragraph with **bold** and *italic* text.

## Subtitle

- List item 1
- List item 2

Check [this link](https://example.com) for more info.

` + "```" + `
code block here
` + "```"

	result := renderMarkdown(input)

	assert.Contains(t, result, "<h1>Main Title</h1>")
	assert.Contains(t, result, "<strong>bold</strong>")
	assert.Contains(t, result, "<em>italic</em>")
	assert.Contains(t, result, "<h2>Subtitle</h2>")
	assert.Contains(t, result, "<ul>")
	assert.Contains(t, result, "<li>List item 1</li>")
	assert.Contains(t, result, "<a href=\"https://example.com\">this link</a>")
	assert.Contains(t, result, "<pre><code>")
	assert.Contains(t, result, "code block here")
}
