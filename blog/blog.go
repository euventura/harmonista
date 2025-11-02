package blog

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"harmonista/models"
)

type BlogModule struct {
	db *gorm.DB
}

func NewBlogModule(db *gorm.DB) *BlogModule {
	return &BlogModule{db: db}
}

func (b *BlogModule) RegisterRoutes(router *gin.Engine) {
	blogGroup := router.Group("/@/:subdomain")
	{
		blogGroup.GET("/", b.index)
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
	if err := b.db.Where("blog_id = ? AND draft = ?", blog.ID, false).
		Order("created_at DESC").
		Find(&posts).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "blog_error.html", gin.H{
			"error": "Erro ao carregar posts",
		})
		return
	}

	c.HTML(http.StatusOK, "blog_index.html", gin.H{
		"blog":  blog,
		"posts": posts,
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
	})
}

func renderMarkdown(content string) string {
	html := content

	lines := strings.Split(html, "\n")
	var result []string
	inCodeBlock := false
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				result = append(result, "<pre><code>")
			} else {
				result = append(result, "</code></pre>")
			}
			continue
		}

		if inCodeBlock {
			result = append(result, line)
			continue
		}

		if strings.HasPrefix(trimmed, "# ") {
			result = append(result, "<h1>"+trimmed[2:]+"</h1>")
		} else if strings.HasPrefix(trimmed, "## ") {
			result = append(result, "<h2>"+trimmed[3:]+"</h2>")
		} else if strings.HasPrefix(trimmed, "### ") {
			result = append(result, "<h3>"+trimmed[4:]+"</h3>")
		} else if strings.HasPrefix(trimmed, "#### ") {
			result = append(result, "<h4>"+trimmed[5:]+"</h4>")
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				result = append(result, "<ul>")
				inList = true
			}
			result = append(result, "<li>"+trimmed[2:]+"</li>")
		} else {
			if inList && trimmed == "" {
				result = append(result, "</ul>")
				inList = false
			}
			if trimmed != "" {
				formatted := formatInlineMarkdown(trimmed)
				result = append(result, "<p>"+formatted+"</p>")
			}
		}
	}

	if inList {
		result = append(result, "</ul>")
	}

	return strings.Join(result, "\n")
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
