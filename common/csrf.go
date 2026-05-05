package common

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSRFOriginCheck rejects state-changing requests (POST/PUT/PATCH/DELETE)
// whose Origin or Referer header does not match the configured DOMAIN host
// (or any of its subdomains, since each blog runs at <sub>.<domain>).
//
// This is a defense-in-depth on top of SameSite=Strict cookies. Modern
// browsers always send Origin on cross-site form POSTs, so a missing or
// mismatching value indicates an off-site request.
func CSRFOriginCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		expectedHost := configuredHost()
		if expectedHost == "" {
			// No DOMAIN configured (local dev or misconfig) - skip check.
			c.Next()
			return
		}

		if !originMatches(c.GetHeader("Origin"), expectedHost) &&
			!originMatches(c.GetHeader("Referer"), expectedHost) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// configuredHost returns the host portion of the DOMAIN env var, or empty.
func configuredHost() string {
	d := os.Getenv("DOMAIN")
	if d == "" {
		return ""
	}
	if !strings.Contains(d, "://") {
		d = "http://" + d
	}
	u, err := url.Parse(d)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// originMatches returns true if header (an Origin or Referer URL) refers to
// the expected host or any of its subdomains.
func originMatches(header, expectedHost string) bool {
	if header == "" {
		return false
	}
	u, err := url.Parse(header)
	if err != nil || u.Host == "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	if h == expectedHost {
		return true
	}
	return strings.HasSuffix(h, "."+expectedHost)
}
