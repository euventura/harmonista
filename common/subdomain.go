package common

import (
	"net/http"
	"os"
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
		hostWithoutPort := host
		if strings.Contains(host, ":") {
			hostWithoutPort = strings.Split(host, ":")[0]
		}

		// Check if this is a subdomain request
		if strings.Contains(hostWithoutPort, ".") {
			parts := strings.Split(hostWithoutPort, ".")

			if len(parts) >= 2 {
				possibleSubdomain := parts[0]
				domain := strings.Join(parts[1:], ".")

				envDomain := os.Getenv("DOMAIN")
				if envDomain == "" {
					envDomain = "http://localhost"
				}

				baseDomain := strings.TrimPrefix(envDomain, "https://")
				baseDomain = strings.TrimPrefix(baseDomain, "http://")
				if strings.Contains(baseDomain, ":") {
					baseDomain = strings.Split(baseDomain, ":")[0]
				}

				// Check if the request domain matches the configured base domain
				if domain == baseDomain {
					// Skip www, admin, api, mail, etc. - only handle blog subdomains
					if possibleSubdomain != "www" && possibleSubdomain != "admin" &&
						possibleSubdomain != "api" && possibleSubdomain != "mail" &&
						possibleSubdomain != "ftp" && possibleSubdomain != "smtp" {

						// Rewrite the URL internally to /@/subdomain format
						originalPath := c.Request.URL.Path
						newPath := envDomain + "/@/" + possibleSubdomain + originalPath

						c.Redirect(http.StatusMovedPermanently, newPath)

					}
				}
			}
		}

		c.Next()
	}
}
