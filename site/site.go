package site

import (
	"net/http"
	"os"
	"strings"
	"time"

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
	router.GET("/sitemap.xml", s.sitemap)
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

func (s *SiteModule) sitemap(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost"
	}
	
	// Remove trailing slash if present
	domain = strings.TrimSuffix(domain, "/")

	// Build sitemap XML
	var sitemap strings.Builder
	sitemap.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sitemap.WriteString("\n")
	sitemap.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sitemap.WriteString("\n")

	// Add main site pages
	sitemap.WriteString("  <url>\n")
	sitemap.WriteString("    <loc>" + domain + "/</loc>\n")
	sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
	sitemap.WriteString("    <priority>1.0</priority>\n")
	sitemap.WriteString("  </url>\n")

	sitemap.WriteString("  <url>\n")
	sitemap.WriteString("    <loc>" + domain + "/leia</loc>\n")
	sitemap.WriteString("    <changefreq>daily</changefreq>\n")
	sitemap.WriteString("    <priority>0.8</priority>\n")
	sitemap.WriteString("  </url>\n")

	// Get all blogs
	var blogs []models.Blog
	s.db.Find(&blogs)

	// Add blog URLs
	for _, blog := range blogs {
		// Blog home page
		sitemap.WriteString("  <url>\n")
		sitemap.WriteString("    <loc>" + domain + "/@/" + blog.Subdomain + "/</loc>\n")
		sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
		sitemap.WriteString("    <priority>0.7</priority>\n")
		sitemap.WriteString("  </url>\n")

		// Get blog posts
		var posts []models.Post
		s.db.Where("blog_id = ?", blog.ID).Find(&posts)

		for _, post := range posts {
			sitemap.WriteString("  <url>\n")
			sitemap.WriteString("    <loc>" + domain + "/@/" + blog.Subdomain + "/" + post.Slug + "</loc>\n")
			sitemap.WriteString("    <lastmod>" + post.UpdatedAt.Format(time.RFC3339) + "</lastmod>\n")
			sitemap.WriteString("    <changefreq>monthly</changefreq>\n")
			sitemap.WriteString("    <priority>0.6</priority>\n")
			sitemap.WriteString("  </url>\n")
		}

		// Get blog pages
		var pages []models.Page
		s.db.Where("blog_id = ?", blog.ID).Find(&pages)

		for _, page := range pages {
			sitemap.WriteString("  <url>\n")
			sitemap.WriteString("    <loc>" + domain + "/@/" + blog.Subdomain + "/p/" + page.Slug + "</loc>\n")
			sitemap.WriteString("    <lastmod>" + page.UpdatedAt.Format(time.RFC3339) + "</lastmod>\n")
			sitemap.WriteString("    <changefreq>monthly</changefreq>\n")
			sitemap.WriteString("    <priority>0.5</priority>\n")
			sitemap.WriteString("  </url>\n")
		}
	}

	// Get all unique tags for tag pages
	var tags []string
	s.db.Model(&models.PostTag{}).
		Joins("JOIN posts ON post_tags.post_id = posts.id").
		Joins("JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ?", true).
		Distinct("post_tags.tag_name").
		Pluck("post_tags.tag_name", &tags)

	for _, tag := range tags {
		sitemap.WriteString("  <url>\n")
		sitemap.WriteString("    <loc>" + domain + "/leia/" + tag + "</loc>\n")
		sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
		sitemap.WriteString("    <priority>0.4</priority>\n")
		sitemap.WriteString("  </url>\n")
	}

	sitemap.WriteString("</urlset>\n")

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, sitemap.String())
}
