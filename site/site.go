package site

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"harmonista/models"
)

// xmlSafeURL escapes a value used inside an XML <loc> element. The path
// segments are percent-encoded so user-controlled tokens (slugs, subdomains,
// tag names) cannot break out of the element.
func xmlSafeURL(s string) string {
	return html.EscapeString(s)
}

// pathEscapeSegment percent-encodes a single path segment.
func pathEscapeSegment(s string) string {
	return url.PathEscape(s)
}

type SiteModule struct {
	db *gorm.DB
}

func NewSiteModule(db *gorm.DB) *SiteModule {
	return &SiteModule{
		db: db,
	}
}

func (s *SiteModule) RegisterRoutes(router *gin.Engine) {
	router.GET("/", s.index)
	router.GET("/leia", s.listReader)
	router.GET("/rss", s.rss)
	router.GET("/sitemap.xml", s.sitemap)
	router.StaticFile("/robots.txt", "./public/robots.txt")
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
		"posts":         posts,
		"canonicalPath": "/",
	})
}

func (s *SiteModule) listReader(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost/"
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	log.Println(page)
	limit := 50
	offset := (page - 1) * limit

	var posts []models.Post

	s.db.Preload("Blog").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Order("posts.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts)

	overflow := 0

	if len(posts) == limit {
		overflow = 1
	}

	c.HTML(http.StatusOK, "site_list_reader.html", gin.H{
		"posts":         posts,
		"domain":        domain,
		"nextpage":      page + 1,
		"prevpage":      page - 1,
		"overflow":      overflow,
		"canonicalPath": "/leia",
	})
}

func (s *SiteModule) sitemap(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost"
	}

	// Remove trailing slash if present
	domain = strings.TrimSuffix(domain, "/")

	type postWithBlog struct {
		ID            uint
		Slug          string
		UpdatedAt     time.Time
		BlogID        uint
		BlogSubdomain string
	}

	var postRows []postWithBlog
	s.db.Table("posts").
		Select("posts.id, posts.slug, posts.updated_at, posts.blog_id, blogs.subdomain as blog_subdomain").
		Joins("JOIN blogs ON posts.blog_id = blogs.id").
		Joins("JOIN users ON blogs.user_id = users.id").
		Where("users.email_verified = ? AND posts.draft = ?", true, false).
		Order("posts.updated_at DESC").
		Limit(1000).
		Scan(&postRows)

	blogPosts := make(map[uint][]postWithBlog)
	blogSubdomains := make(map[uint]string)
	blogIDs := make([]uint, 0)

	for _, row := range postRows {
		blogPosts[row.BlogID] = append(blogPosts[row.BlogID], row)
		if _, exists := blogSubdomains[row.BlogID]; !exists {
			blogSubdomains[row.BlogID] = row.BlogSubdomain
			blogIDs = append(blogIDs, row.BlogID)
		}
	}

	type pageRow struct {
		BlogID    uint
		Slug      string
		UpdatedAt time.Time
	}

	pagesByBlog := make(map[uint][]pageRow)
	if len(blogIDs) > 0 {
		var pageRows []pageRow
		s.db.Table("pages").
			Select("pages.blog_id, pages.slug, pages.updated_at").
			Where("pages.blog_id IN ?", blogIDs).
			Scan(&pageRows)

		for _, row := range pageRows {
			pagesByBlog[row.BlogID] = append(pagesByBlog[row.BlogID], row)
		}
	}

	// Build sitemap XML
	var sitemap strings.Builder
	sitemap.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sitemap.WriteString("\n")
	sitemap.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sitemap.WriteString("\n")

	// Add main site pages
	sitemap.WriteString("  <url>\n")
	sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/") + "</loc>\n")
	sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
	sitemap.WriteString("    <priority>1.0</priority>\n")
	sitemap.WriteString("  </url>\n")

	sitemap.WriteString("  <url>\n")
	sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/leia") + "</loc>\n")
	sitemap.WriteString("    <changefreq>daily</changefreq>\n")
	sitemap.WriteString("    <priority>0.8</priority>\n")
	sitemap.WriteString("  </url>\n")

	for _, blogID := range blogIDs {
		subdomain := pathEscapeSegment(blogSubdomains[blogID])

		sitemap.WriteString("  <url>\n")
		sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/@/"+subdomain+"/") + "</loc>\n")
		sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
		sitemap.WriteString("    <priority>0.7</priority>\n")
		sitemap.WriteString("  </url>\n")

		for _, post := range blogPosts[blogID] {
			slug := pathEscapeSegment(post.Slug)
			sitemap.WriteString("  <url>\n")
			sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/@/"+subdomain+"/"+slug) + "</loc>\n")
			sitemap.WriteString("    <lastmod>" + post.UpdatedAt.Format(time.RFC3339) + "</lastmod>\n")
			sitemap.WriteString("    <changefreq>monthly</changefreq>\n")
			sitemap.WriteString("    <priority>0.6</priority>\n")
			sitemap.WriteString("  </url>\n")
		}

		for _, page := range pagesByBlog[blogID] {
			slug := pathEscapeSegment(page.Slug)
			sitemap.WriteString("  <url>\n")
			sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/@/"+subdomain+"/p/"+slug) + "</loc>\n")
			sitemap.WriteString("    <lastmod>" + page.UpdatedAt.Format(time.RFC3339) + "</lastmod>\n")
			sitemap.WriteString("    <changefreq>monthly</changefreq>\n")
			sitemap.WriteString("    <priority>0.5</priority>\n")
			sitemap.WriteString("  </url>\n")
		}
	}

	var tags []string
	s.db.Model(&models.PostTag{}).
		Joins("JOIN posts ON post_tags.post_id = posts.id").
		Joins("JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ?", true).
		Distinct("post_tags.tag_name").
		Pluck("post_tags.tag_name", &tags)

	for _, tag := range tags {
		safeTag := pathEscapeSegment(tag)
		sitemap.WriteString("  <url>\n")
		sitemap.WriteString("    <loc>" + xmlSafeURL(domain+"/leia/"+safeTag) + "</loc>\n")
		sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
		sitemap.WriteString("    <priority>0.4</priority>\n")
		sitemap.WriteString("  </url>\n")
	}

	sitemap.WriteString("</urlset>\n")

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, sitemap.String())
}
