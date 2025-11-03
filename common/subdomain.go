package common

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SubdomainMiddleware handles subdomain routing
// Converts subdomain.harmonista.com requests to /@/subdomain format internally
func SubdomainMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		
		// Remove port if present (for local development)
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
		
		// Check if this is a subdomain request
		if strings.Contains(host, ".") {
			parts := strings.Split(host, ".")
			
			// Check if it's a subdomain of harmonista.com or localhost
			if len(parts) >= 2 {
				possibleSubdomain := parts[0]
				domain := strings.Join(parts[1:], ".")
				
				// Only handle subdomains for harmonista.com or localhost
				if domain == "harmonista.com" || domain == "localhost" {
					// Skip www, admin, api, mail, etc. - only handle blog subdomains
					if possibleSubdomain != "www" && possibleSubdomain != "admin" && 
					   possibleSubdomain != "api" && possibleSubdomain != "mail" &&
					   possibleSubdomain != "ftp" && possibleSubdomain != "smtp" {
						
						// Rewrite the URL internally to /@/subdomain format
						originalPath := c.Request.URL.Path
						newPath := "/@/" + possibleSubdomain + originalPath
						
						// Set the new path
						c.Request.URL.Path = newPath
						
						// Set subdomain in context for blog routes to use
						c.Set("subdomain", possibleSubdomain)
						c.Set("is_subdomain_request", true)
						c.Set("original_path", originalPath)
					}
				}
			}
		}
		
		c.Next()
	}
}