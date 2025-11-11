package cache

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// CacheMiddleware is a middleware that caches blog post pages
func CacheMiddleware(maxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// Only cache blog post pages (/@/subdomain/slug)
		path := c.Request.URL.Path
		if !isBlogPostPath(path) {
			c.Next()
			return
		}

		// Extract subdomain and slug from path
		subdomain, slug := extractFromPath(path)
		if subdomain == "" || slug == "" {
			c.Next()
			return
		}

		// Try to read from cache
		if cached, found := ReadCache(subdomain, slug, maxAge); found {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(cached))
			c.Abort()
			return
		}

		// Cache miss - capture response
		c.Header("X-Cache", "MISS")

		// Create custom writer to capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = writer

		c.Next()

		// Only cache successful HTML responses
		if c.Writer.Status() == http.StatusOK &&
			c.Writer.Header().Get("Content-Type") == "text/html; charset=utf-8" {
			WriteCache(subdomain, slug, writer.body.String())
		}
	}
}

// isBlogPostPath checks if the path is a blog post path (/@/subdomain/slug)
func isBlogPostPath(path string) bool {
	// /@/subdomain/postslug or subdomain.domain/postslug
	// We'll detect based on the number of slashes
	// /@/subdomain/slug has 3 parts when split by /
	// Skip if it's an index, page, or tag route
	if len(path) < 4 || path[len(path)-1] == '/' {
		return false
	}

	// Skip /p/ (pages) and /t/ (tags)
	if bytes.Contains([]byte(path), []byte("/p/")) ||
		bytes.Contains([]byte(path), []byte("/t/")) {
		return false
	}

	return true
}

// extractFromPath extracts subdomain and slug from path
// For /@/subdomain/slug format
func extractFromPath(path string) (subdomain, slug string) {
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Split by /
	parts := splitPath(path)

	// /@/subdomain/slug format
	if len(parts) >= 3 && parts[0] == "@" {
		return parts[1], parts[2]
	}

	return "", ""
}

func splitPath(path string) []string {
	var parts []string
	var current string

	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
