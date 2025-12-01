package site

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"os"
	"strings"
	"time"

	"harmonista/models"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
)

// markdown renderer configured with Goldmark and useful extensions
// Duplicated from blog package to keep modules decoupled
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,     // tables, strikethrough, task lists, autolinks (GFM set)
		extension.Linkify, // linkify raw URLs
	),
	goldmark.WithRendererOptions(
		htmlrenderer.WithUnsafe(), // allow raw HTML passthrough in Markdown
	),
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
	Author      string `xml:"author,omitempty"`
}

func renderMarkdown(content string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return content
	}
	return buf.String()
}

func (s *SiteModule) rss(c *gin.Context) {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost"
	}
	domain = strings.TrimSuffix(domain, "/")

	var posts []struct {
		models.Post
		BlogSubdomain string `json:"blog_subdomain"`
		BlogTitle     string `json:"blog_title"`
	}

	// Fetch latest 50 posts from blogs in the list reader
	s.db.Table("posts").
		Select("posts.*, blogs.subdomain as blog_subdomain, blogs.title as blog_title").
		Joins("INNER JOIN blogs ON posts.blog_id = blogs.id").
		Where("blogs.is_list_reader = ? AND posts.draft = ?", true, false).
		Order("posts.created_at DESC").
		Limit(50).
		Find(&posts)

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "Harmonista Reader",
			Link:        domain + "/leia",
			Description: "Latest posts from all blogs on Harmonista",
		},
	}

	for _, post := range posts {
		// Construct URL: domain/@/subdomain/slug
		// We use the main domain format here since this is the aggregator RSS
		postURL := domain + "/@/" + post.BlogSubdomain + "/" + post.Slug

		rss.Channel.Items = append(rss.Channel.Items, Item{
			Title:       post.Title,
			Link:        postURL,
			Description: renderMarkdown(post.Content),
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
			Guid:        postURL,
			Author:      post.BlogTitle, // Using Blog Title as author
		})
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.XML(http.StatusOK, rss)
}
