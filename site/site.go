package site

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"harmonista/models"
)

type SiteModule struct {
	db *gorm.DB
}

func NewSiteModule(db *gorm.DB) *SiteModule {
	return &SiteModule{db: db}
}

func (s *SiteModule) RegisterRoutes(router *gin.Engine) {
	router.GET("/", s.index)
	router.GET("/leia", s.listReader)
	router.GET("/leia/:tagName", s.listReaderByTag)
}

func (s *SiteModule) index(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}

	c.HTML(http.StatusOK, "site_index.html", gin.H{
		"domain": domain,
	})
}

func (s *SiteModule) listReader(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}

	// Buscar todos os posts de blogs que tem isListReader = true
	var posts []struct {
		models.Post
		BlogSubdomain string
		Tags          []models.Tag
	}

	// Primeiro buscar os posts
	var rawPosts []models.Post
	err := s.db.Table("posts").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Order("posts.created_at DESC").
		Find(&rawPosts).Error

	if err != nil {
		c.HTML(http.StatusInternalServerError, "site_list_reader.html", gin.H{
			"error":  "Erro ao carregar posts",
			"domain": domain,
		})
		return
	}

	// Para cada post, buscar o subdomain do blog e as tags
	for _, post := range rawPosts {
		var blog models.Blog
		s.db.Where("id = ?", post.BlogID).First(&blog)

		var tags []models.Tag
		s.db.Table("tags").
			Joins("INNER JOIN post_tags ON tags.id = post_tags.tag_id").
			Where("post_tags.post_id = ?", post.ID).
			Find(&tags)

		posts = append(posts, struct {
			models.Post
			BlogSubdomain string
			Tags          []models.Tag
		}{
			Post:          post,
			BlogSubdomain: blog.Subdomain,
			Tags:          tags,
		})
	}

	c.HTML(http.StatusOK, "site_list_reader.html", gin.H{
		"posts":  posts,
		"domain": domain,
	})
}

func (s *SiteModule) listReaderByTag(c *gin.Context) {
	tagName := c.Param("tagName")
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}

	// Buscar a tag pelo nome
	var tag models.Tag
	if err := s.db.Where("title = ?", tagName).First(&tag).Error; err != nil {
		c.HTML(http.StatusNotFound, "site_list_reader_by_tag.html", gin.H{
			"error":  "Tag não encontrada",
			"domain": domain,
		})
		return
	}

	// Buscar todos os posts de blogs que tem isListReader = true com essa tag
	var posts []struct {
		models.Post
		BlogSubdomain string
		Tags          []models.Tag
	}

	// Primeiro buscar os posts
	var rawPosts []models.Post
	err := s.db.Table("posts").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Joins("INNER JOIN post_tags ON posts.id = post_tags.post_id").
		Where("blogs.is_list_reader = ? AND posts.draft = ? AND post_tags.tag_id = ?", true, false, tag.ID).
		Order("posts.created_at DESC").
		Find(&rawPosts).Error

	if err != nil {
		c.HTML(http.StatusInternalServerError, "site_list_reader_by_tag.html", gin.H{
			"error":   "Erro ao carregar posts",
			"tag":     tag,
			"domain":  domain,
			"tagName": tagName,
		})
		return
	}

	// Para cada post, buscar o subdomain do blog e as tags
	for _, post := range rawPosts {
		var blog models.Blog
		s.db.Where("id = ?", post.BlogID).First(&blog)

		var tags []models.Tag
		s.db.Table("tags").
			Joins("INNER JOIN post_tags ON tags.id = post_tags.tag_id").
			Where("post_tags.post_id = ?", post.ID).
			Find(&tags)

		posts = append(posts, struct {
			models.Post
			BlogSubdomain string
			Tags          []models.Tag
		}{
			Post:          post,
			BlogSubdomain: blog.Subdomain,
			Tags:          tags,
		})
	}

	c.HTML(http.StatusOK, "site_list_reader_by_tag.html", gin.H{
		"posts":   posts,
		"tag":     tag,
		"domain":  domain,
		"tagName": tagName,
	})
}
