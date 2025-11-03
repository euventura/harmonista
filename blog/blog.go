package blog

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
	"gorm.io/gorm"

	"harmonista/models"
)

type BlogModule struct {
	db *gorm.DB
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

func NewBlogModule(db *gorm.DB) *BlogModule {
	return &BlogModule{db: db}
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

func (b *BlogModule) RegisterRoutes(router *gin.Engine) {
	blogGroup := router.Group("/@/:subdomain")
	{
		blogGroup.GET("/", b.index)
		blogGroup.GET("/p/:pageSlug", b.page)
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

	// Debug: verificar se o tema está sendo carregado
	fmt.Printf("DEBUG - Blog ID: %d, Subdomain: %s, Theme length: %d\n", blog.ID, blog.Subdomain, len(blog.Theme))
	if len(blog.Theme) > 0 {
		fmt.Printf("DEBUG - Theme preview: %.100s...\n", blog.Theme)
	}

	var posts []models.Post
	if err := b.db.Where("blog_id = ? AND draft = ?", blog.ID, false).
		Order("created_at DESC").
		Find(&posts).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "blog_error.html", gin.H{
			"error": "Erro ao carregar posts",
		})
		return
	}

	navLinks := parseNavLinks(blog.Nav)

	// Suporte para parâmetro ?css=<path>
	previewCSS := c.Query("css")

	c.HTML(http.StatusOK, "blog_index.html", gin.H{
		"blog":                blog,
		"posts":               posts,
		"navLinks":            navLinks,
		"blogDescriptionHTML": template.HTML(renderMarkdown(blog.Description)),
		"previewCSS":          previewCSS,
		"blogThemeCSS":        template.CSS(blog.Theme),
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

	contentHTML := template.HTML(renderMarkdown(page.Content))

	navLinks := parseNavLinks(blog.Nav)

	// Suporte para parâmetro ?css=<path>
	previewCSS := c.Query("css")

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

	contentHTML := template.HTML(renderMarkdown(post.Content))

	navLinks := parseNavLinks(blog.Nav)

	// Suporte para parâmetro ?css=<path>
	previewCSS := c.Query("css")

	c.HTML(http.StatusOK, "blog_post.html", gin.H{
		"blog": blog,
		"post": gin.H{
			"ID":        post.ID,
			"Title":     post.Title,
			"Slug":      post.Slug,
			"Content":   contentHTML,
			"CreatedAt": post.CreatedAt,
			"UpdatedAt": post.UpdatedAt,
		},
		"navLinks":     navLinks,
		"previewCSS":   previewCSS,
		"blogThemeCSS": template.CSS(blog.Theme),
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

func formatInlineMarkdown(text string) string {
	text = replaceBold(text)
	text = replaceItalic(text)
	text = replaceLinks(text)
	text = replaceCode(text)
	return text
}

func replaceBold(text string) string {
	for strings.Contains(text, "**") {
		first := strings.Index(text, "**")
		if first == -1 {
			break
		}
		second := strings.Index(text[first+2:], "**")
		if second == -1 {
			break
		}
		second += first + 2
		content := text[first+2 : second]
		text = text[:first] + "<strong>" + content + "</strong>" + text[second+2:]
	}
	return text
}

func replaceItalic(text string) string {
	for strings.Contains(text, "*") && !strings.Contains(text, "**") {
		first := strings.Index(text, "*")
		if first == -1 {
			break
		}
		second := strings.Index(text[first+1:], "*")
		if second == -1 {
			break
		}
		second += first + 1
		content := text[first+1 : second]
		text = text[:first] + "<em>" + content + "</em>" + text[second+1:]
	}
	return text
}

func replaceLinks(text string) string {
	for strings.Contains(text, "[") {
		linkStart := strings.Index(text, "[")
		if linkStart == -1 {
			break
		}
		linkEnd := strings.Index(text[linkStart:], "]")
		if linkEnd == -1 {
			break
		}
		linkEnd += linkStart

		if linkEnd+1 >= len(text) || text[linkEnd+1] != '(' {
			break
		}

		urlEnd := strings.Index(text[linkEnd+2:], ")")
		if urlEnd == -1 {
			break
		}
		urlEnd += linkEnd + 2

		linkText := text[linkStart+1 : linkEnd]
		url := text[linkEnd+2 : urlEnd]
		text = text[:linkStart] + "<a href=\"" + url + "\">" + linkText + "</a>" + text[urlEnd+1:]
	}
	return text
}

func replaceCode(text string) string {
	for strings.Contains(text, "`") {
		first := strings.Index(text, "`")
		if first == -1 {
			break
		}
		second := strings.Index(text[first+1:], "`")
		if second == -1 {
			break
		}
		second += first + 1
		content := text[first+1 : second]
		text = text[:first] + "<code>" + content + "</code>" + text[second+1:]
	}
	return text
}
