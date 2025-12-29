package site

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"harmonista/models"
)

func TestSiteSitemap_FiltersAndIncludes(t *testing.T) {
	t.Setenv("DOMAIN", "http://example.com")

	db := setupTestDB()
	siteModule := NewSiteModule(db, nil)
	router := setupTestRouter(siteModule)

	// Verified user and visible content
	user := createTestUser(db)
	user.EmailVerified = true
	db.Save(user)

	blog := createTestBlog(db, user.ID, true)
	post := createTestPost(db, blog.ID, false)
	post.Slug = "visible-post"
	db.Save(post)

	page := &models.Page{
		BlogID:    blog.ID,
		Title:     "Visible Page",
		Slug:      "visible-page",
		Content:   "content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.Create(page)

	// Draft post should be excluded
	draftPost := createTestPost(db, blog.ID, true)
	draftPost.Slug = "draft-post"
	db.Save(draftPost)

	// Unverified user content should be excluded
	otherUser := &models.User{Email: "other@example.com", PasswordHash: "hash", EmailVerified: false}
	db.Create(otherUser)

	otherBlog := createTestBlog(db, otherUser.ID, true)
	otherBlog.Subdomain = "unverifiedblog"
	db.Save(otherBlog)

	otherPost := createTestPost(db, otherBlog.ID, false)
	otherPost.Slug = "hidden-post"
	db.Save(otherPost)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sitemap.xml", nil)
	router.ServeHTTP(w, req)

	body := w.Body.String()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "<loc>http://example.com/</loc>")
	assert.Contains(t, body, "<loc>http://example.com/@/testblog/</loc>")
	assert.Contains(t, body, "<loc>http://example.com/@/testblog/visible-post</loc>")
	assert.Contains(t, body, "<loc>http://example.com/@/testblog/p/visible-page</loc>")
	assert.NotContains(t, body, "draft-post")
	assert.NotContains(t, body, "unverifiedblog")
	assert.NotContains(t, body, "hidden-post")
}
