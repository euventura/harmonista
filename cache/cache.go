package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"gorm.io/gorm"
	"harmonista/models"
)

// GetCachePath returns the cache file path for a blog post
func GetCachePath(subdomain, slug string) string {
	hash := generateHash(subdomain + slug)
	shortHash := hash[:16]
	cacheDir := filepath.Join("cache", subdomain)
	return filepath.Join(cacheDir, fmt.Sprintf("%s_%s.html", slug, shortHash))
}

// generateHash generates an xxHash hash for the given string
func generateHash(s string) string {
	hash := xxhash.Sum64String(s)
	// Convert uint64 to hex string
	return fmt.Sprintf("%016x", hash)
}

// EnsureCacheDir ensures the cache directory exists
func EnsureCacheDir(subdomain string) error {
	cacheDir := filepath.Join("cache", subdomain)
	return os.MkdirAll(cacheDir, 0755)
}

// WriteCache writes HTML content to cache file
func WriteCache(subdomain, slug, html string) error {
	if err := EnsureCacheDir(subdomain); err != nil {
		return err
	}

	cachePath := GetCachePath(subdomain, slug)
	return ioutil.WriteFile(cachePath, []byte(html), 0644)
}

// ReadCache reads HTML content from cache file if it exists and is not expired
func ReadCache(subdomain, slug string, maxAge time.Duration) (string, bool) {
	cachePath := GetCachePath(subdomain, slug)

	info, err := os.Stat(cachePath)
	if err != nil {
		return "", false
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > maxAge {
		return "", false
	}

	content, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return "", false
	}

	return string(content), true
}

// ClearCache removes a specific cache file
func ClearCache(subdomain, slug string) error {
	cachePath := GetCachePath(subdomain, slug)
	err := os.Remove(cachePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ClearCacheByPostID removes cache for a post by its ID
// This looks up the blog subdomain and post slug from the database
func ClearCacheByPostID(db *gorm.DB, postID int) error {
	var post models.Post
	if err := db.Preload("Blog").First(&post, postID).Error; err != nil {
		// If post not found, it's ok - nothing to clear
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return ClearCache(post.Blog.Subdomain, post.Slug)
}

// ClearCacheBySlugs removes cache files matching a glob pattern in subdomain directory
func ClearCacheBySlugs(subdomain string, slugs ...string) error {
	cacheDir := filepath.Join("cache", subdomain)

	for _, slug := range slugs {
		// Remove exact match
		if err := ClearCache(subdomain, slug); err != nil {
			return err
		}

		// Also try to remove any files starting with this slug
		// This handles cases where slug might have changed
		pattern := filepath.Join(cacheDir, slug+"_*.html")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			os.Remove(match)
		}
	}

	return nil
}

// ClearAllBlogCache removes all cache files for a blog
func ClearAllBlogCache(subdomain string) error {
	cacheDir := filepath.Join("cache", subdomain)
	return os.RemoveAll(cacheDir)
}

// ClearOldCache removes cache files older than the specified duration
func ClearOldCache(maxAge time.Duration) error {
	cacheRoot := "cache"

	return filepath.Walk(cacheRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip non-HTML files
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Remove if older than maxAge
		if time.Since(info.ModTime()) > maxAge {
			os.Remove(path)
		}

		return nil
	})
}
