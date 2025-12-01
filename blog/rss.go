package blog

import (
	"encoding/xml"
	"net/http"
	"time"

	"harmonista/models"

	"github.com/gin-gonic/gin"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Guid        string `xml:"guid"`
}

func (b *BlogModule) rss(c *gin.Context) {
	subdomain := c.Param("subdomain")

	blog, err := b.getBlogBySubdomain(subdomain)
	if err != nil {
		c.XML(http.StatusNotFound, gin.H{"error": "Blog não encontrado"})
		return
	}

	var posts []models.Post
	b.db.Where("blog_id = ? AND draft = ?", blog.ID, false).
		Order("created_at DESC").
		Find(&posts)

	blogURL := buildBlogURL(c, blog, "")

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       blog.Title,
			Link:        blogURL,
			Description: blog.Description, // Note: Description might be markdown, ideally should be plain text or HTML escaped
		},
	}

	for _, post := range posts {
		postURL := buildBlogURL(c, blog, "/"+post.Slug)
		rss.Channel.Items = append(rss.Channel.Items, Item{
			Title:       post.Title,
			Link:        postURL,
			Description: renderMarkdown(post.Content), // Rendering markdown to HTML for description
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
			Guid:        postURL,
		})
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.XML(http.StatusOK, rss)
}
