package site

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"harmonista/analytics"
	"harmonista/models"
)

type SiteModule struct {
	db        *gorm.DB
	analytics *analytics.AnalyticsModule
}

func NewSiteModule(db *gorm.DB, analyticsModule *analytics.AnalyticsModule) *SiteModule {
	return &SiteModule{
		db:        db,
		analytics: analyticsModule,
	}
}

func (s *SiteModule) RegisterRoutes(router *gin.Engine) {
	router.GET("/", s.index)
	router.GET("/leia", s.listReader)
	router.GET("/rss", s.rss)
	router.GET("/sitemap.xml", s.sitemap)
}

func (s *SiteModule) index(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}
	fmt.Println(domain)
	var posts []models.Post

	s.db.Preload("Blog").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Order("posts.created_at DESC").
		Limit(10).
		Find(&posts)

	c.HTML(http.StatusOK, "site_index.html", gin.H{
		"posts": posts,
	})
}

func (s *SiteModule) listReader(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	var posts []struct {
		models.Post
		BlogSubdomain string `json:"blog_subdomain"`
		Tags          string `json:"tags"`
	}

	s.db.Table("posts").
		Select("posts.*, blogs.subdomain as blog_subdomain, GROUP_CONCAT(tags.title) as tags").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Joins("LEFT JOIN post_tags ON post_tags.post_id = posts.id").
		Joins("LEFT JOIN tags ON tags.id = post_tags.tag_id").
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Group("posts.id").
		Order("posts.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts)

	c.HTML(http.StatusOK, "site_list_reader.html", gin.H{
		"posts":  posts,
		"domain": domain,
		"page":   page,
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
