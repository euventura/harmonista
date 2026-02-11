package blog

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
	"gorm.io/gorm"

	"harmonista/analytics"
	"harmonista/models"
)

type BlogModule struct {
	db        *gorm.DB
	analytics *analytics.AnalyticsModule
}

// markdown renderer configured with Goldmark and useful extensions
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,     // tables, strikethrough, task lists, autolinks (GFM set)
		extension.Linkify, // linkify raw URLs
	),
	goldmark.WithRendererOptions(
		htmlrenderer.WithUnsafe(), // allow raw HTML passthrough in Markdown
	),
)

type NavLink struct {
	Text string
	URL  string
}

func NewBlogModule(db *gorm.DB, analyticsModule *analytics.AnalyticsModule) *BlogModule {
	return &BlogModule{
		db:        db,
		analytics: analyticsModule,
	}
}

func parseNavLinks(navString string) []NavLink {
	if navString == "" {
		return nil
	}

	// Regex para capturar links markdown no formato [texto](url)
	re := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	matches := re.FindAllStringSubmatch(navString, -1)

	var navLinks []NavLink
	for _, match := range matches {
		if len(match) == 3 {
			navLinks = append(navLinks, NavLink{
				Text: match[1],
				URL:  match[2],
			})
		}
	}

	return navLinks
}

func buildBlogURL(c *gin.Context, blog *models.Blog, path string) string {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost"
	}

	// Remove trailing slash
	domain = strings.TrimSuffix(domain, "/")

	// Check if this is a subdomain request
	isSubdomain, exists := c.Get("is_subdomain_request")
	if exists && isSubdomain.(bool) {
		// For subdomain requests, use subdomain.domain format
		baseDomain := strings.TrimPrefix(domain, "http://")
		baseDomain = strings.TrimPrefix(baseDomain, "https://")
		protocol := "http://"
		if strings.HasPrefix(domain, "https://") {
			protocol = "https://"
		}
		return protocol + blog.Subdomain + "." + baseDomain + path
	}

	// For regular requests, use /@/subdomain format
	return domain + "/@/" + blog.Subdomain + path
}

func (b *BlogModule) RegisterRoutes(router *gin.Engine) {
	// Original routes with /@/subdomain format
	blogGroup := router.Group("/@/:subdomain")
	{
		blogGroup.GET("/", b.index)
		blogGroup.GET("/rss", b.rss)
		blogGroup.GET("/p/:pageSlug", b.page)
		blogGroup.GET("/t/:tagName", b.tag)
		blogGroup.GET("/:postSlug", b.post)
	}
}

func (b *BlogModule) getBlogBySubdomain(subdomain string) (*models.Blog, error) {
	var blog models.Blog
	err := b.db.Where("subdomain = ?", subdomain).First(&blog).Error
	return &blog, err
}

func (b *BlogModule) index(c *gin.Context) {
	subdomain := c.Param("subdomain")

	blog, err := b.getBlogBySubdomain(subdomain)
	if err != nil {
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}

	var posts []models.Post
	b.db.Where("blog_id = ? AND draft = ?", blog.ID, false).
		Order("created_at DESC").
		Find(&posts)

	navLinks := parseNavLinks(blog.Nav)

	previewCSS := c.Query("css")

	blogURL := buildBlogURL(c, blog, "")

	if b.analytics != nil {
		b.analytics.TrackVisit(blog.Subdomain, "_")
	}

	c.HTML(http.StatusOK, "blog_index.html", gin.H{
		"blog":                blog,
		"posts":               posts,
		"navLinks":            navLinks,
		"blogDescriptionHTML": template.HTML(renderMarkdown(blog.Description)),
		"previewCSS":          previewCSS,
		"blogThemeCSS":        template.CSS(blog.Theme),
		"blogURL":             blogURL,
	})
}

func (b *BlogModule) page(c *gin.Context) {
	subdomain := c.Param("subdomain")
	pageSlug := c.Param("pageSlug")

	fmt.Printf("DEBUG PAGE - Subdomain: %s, PageSlug: %s\n", subdomain, pageSlug)

	blog, err := b.getBlogBySubdomain(subdomain)
	if err != nil {
		fmt.Printf("DEBUG PAGE - Blog não encontrado: %v\n", err)
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}

	fmt.Printf("DEBUG PAGE - Blog encontrado: ID=%d\n", blog.ID)

	var page models.Page
	if err := b.db.Where("blog_id = ? AND slug = ? AND draft = ?", blog.ID, pageSlug, false).
		First(&page).Error; err != nil {
		fmt.Printf("DEBUG PAGE - Página não encontrada. BlogID=%d, Slug=%s, Error=%v\n", blog.ID, pageSlug, err)

		// Verificar todas as páginas deste blog
		var allPages []models.Page
		b.db.Where("blog_id = ?", blog.ID).Find(&allPages)
		fmt.Printf("DEBUG PAGE - Páginas disponíveis para este blog:\n")
		for _, p := range allPages {
			fmt.Printf("  - ID=%d, Title=%s, Slug=%s, Draft=%v\n", p.ID, p.Title, p.Slug, p.Draft)
		}

		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Página não encontrada",
		})
		return
	}

	fmt.Printf("DEBUG PAGE - Página encontrada: ID=%d, Title=%s\n", page.ID, page.Title)

	// Track visit to blog page (não trackeamos pages no analytics por enquanto)
	// Pages são diferentes de Posts, e o requisito era trackear Posts

	contentHTML := template.HTML(renderMarkdown(page.Content))

	navLinks := parseNavLinks(blog.Nav)

	// Suporte para parâmetro ?css=<path>
	previewCSS := c.Query("css")

	// Build URLs based on request type (subdomain or /@/subdomain)
	pageURL := buildBlogURL(c, blog, "/p/"+page.Slug)
	blogURL := buildBlogURL(c, blog, "")

	c.HTML(http.StatusOK, "blog_page.html", gin.H{
		"blog": blog,
		"page": gin.H{
			"ID":        page.ID,
			"Title":     page.Title,
			"Slug":      page.Slug,
			"Content":   contentHTML,
			"CreatedAt": page.CreatedAt,
			"UpdatedAt": page.UpdatedAt,
		},
		"navLinks":     navLinks,
		"previewCSS":   previewCSS,
		"blogThemeCSS": template.CSS(blog.Theme),
		"pageURL":      pageURL,
		"blogURL":      blogURL,
	})
}

func (b *BlogModule) tag(c *gin.Context) {
	subdomain := c.Param("subdomain")
	tagName := c.Param("tagName")

	blog, err := b.getBlogBySubdomain(subdomain)
	if err != nil {
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}

	// Buscar a tag pelo nome
	var tag models.Tag
	if err := b.db.Where("title = ?", tagName).First(&tag).Error; err != nil {
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Tag não encontrada",
		})
		return
	}

	// Buscar posts com essa tag
	var posts []models.Post
	b.db.Table("posts").
		Joins("INNER JOIN post_tags ON posts.id = post_tags.post_id").
		Where("post_tags.tag_id = ? AND posts.blog_id = ? AND posts.draft = ?", tag.ID, blog.ID, false).
		Order("posts.created_at DESC").
		Find(&posts)

	navLinks := parseNavLinks(blog.Nav)

	// Suporte para parâmetro ?css=<path>
	previewCSS := c.Query("css")

	c.HTML(http.StatusOK, "blog_tag.html", gin.H{
		"blog":                blog,
		"tag":                 tag,
		"posts":               posts,
		"navLinks":            navLinks,
		"blogDescriptionHTML": template.HTML(renderMarkdown(blog.Description)),
		"previewCSS":          previewCSS,
		"blogThemeCSS":        template.CSS(blog.Theme),
	})
}

func (b *BlogModule) post(c *gin.Context) {
	subdomain := c.Param("subdomain")
	postSlug := c.Param("postSlug")

	blog, err := b.getBlogBySubdomain(subdomain)
	if err != nil {
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}

	var post models.Post
	if err := b.db.Where("blog_id = ? AND slug = ? AND draft = ?", blog.ID, postSlug, false).
		First(&post).Error; err != nil {
		c.HTML(http.StatusNotFound, "blog_error.html", gin.H{
			"error": "Post não encontrado",
		})
		return
	}

	var tags []models.Tag
	b.db.Table("tags").
		Joins("INNER JOIN post_tags ON tags.id = post_tags.tag_id").
		Where("post_tags.post_id = ?", post.ID).
		Find(&tags)

	var replyToPost *models.Post
	if post.ReplyPostID != nil {
		var parentPost models.Post
		if err := b.db.Preload("Blog").
			Where("id = ? AND draft = ?", *post.ReplyPostID, false).
			First(&parentPost).Error; err == nil {
			replyToPost = &parentPost
		}
	}

	var replies []models.Post
	b.db.Preload("Blog").
		Where("reply_post_id = ? AND draft = ?", post.ID, false).
		Order("created_at ASC").
		Find(&replies)

	contentHTML := template.HTML(renderMarkdown(post.Content))

	navLinks := parseNavLinks(blog.Nav)
	previewCSS := c.Query("css")
	postURL := buildBlogURL(c, blog, "/"+post.Slug)
	blogURL := buildBlogURL(c, blog, "")

	postData := gin.H{
		"ID":        post.ID,
		"Title":     post.Title,
		"Slug":      post.Slug,
		"Content":   contentHTML,
		"CreatedAt": post.CreatedAt,
		"UpdatedAt": post.UpdatedAt,
	}

	var replyToData gin.H
	if replyToPost != nil {
		replyToURL := buildBlogURL(c, &replyToPost.Blog, "/"+replyToPost.Slug)
		replyToData = gin.H{
			"Title": replyToPost.Title,
			"Slug":  replyToPost.Slug,
			"URL":   replyToURL,
		}
	}

	// Preparar dados das respostas
	var repliesData []gin.H
	for _, reply := range replies {
		replyURL := buildBlogURL(c, &reply.Blog, "/"+reply.Slug)
		repliesData = append(repliesData, gin.H{
			"ID":        reply.ID,
			"Title":     reply.Title,
			"Slug":      reply.Slug,
			"URL":       replyURL,
			"CreatedAt": reply.CreatedAt,
			"BlogTitle": reply.Blog.Title,
			"Subdomain": reply.Blog.Subdomain,
		})
	}

	c.HTML(http.StatusOK, "blog_post.html", gin.H{
		"blog":         blog,
		"post":         postData,
		"tags":         tags,
		"replyTo":      replyToData,
		"replies":      repliesData,
		"navLinks":     navLinks,
		"previewCSS":   previewCSS,
		"blogThemeCSS": template.CSS(blog.Theme),
		"postURL":      postURL,
		"blogURL":      blogURL,
	})
}

func renderMarkdown(content string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		// Em caso de erro, retorna o conteúdo original para não quebrar a página
		return content
	}
	return buf.String()
}
