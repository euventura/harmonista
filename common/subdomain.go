package common

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WWWRedirectMiddleware redireciona www para non-www com 301 (permanente)
func WWWRedirectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host

		// Remover porta se houver (para desenvolvimento local)
		hostWithoutPort := host
		if strings.Contains(host, ":") {
			hostWithoutPort = strings.Split(host, ":")[0]
		}

		// Se o host começa com www., redirecionar para versão sem www
		if strings.HasPrefix(strings.ToLower(hostWithoutPort), "www.") {
			newHost := strings.TrimPrefix(strings.ToLower(hostWithoutPort), "www.")

			// Se tinha porta, adicionar de volta
			if strings.Contains(host, ":") {
				port := strings.Split(host, ":")[1]
				newHost = newHost + ":" + port
			}

			// Determinar o scheme (http ou https)
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			// Verificar se está atrás de um proxy (X-Forwarded-Proto)
			if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
				scheme = "https"
			}

			// Construir a URL completa
			newURL := scheme + "://" + newHost + c.Request.RequestURI

			// Redirecionar com 301 (Moved Permanently)
			c.Redirect(http.StatusMovedPermanently, newURL)
			c.Abort()
			return
		}

		c.Next()
	}
}

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
