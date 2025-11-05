package cache

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// responseWriter é um wrapper para capturar o HTML gerado
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// Middleware intercepta requisições e serve do cache se disponível
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Apenas processar requisições GET
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Verificar se é uma rota de blog (não admin, não login, etc)
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/admin") ||
			strings.HasPrefix(path, "/login") ||
			strings.HasPrefix(path, "/cadastrar") ||
			strings.HasPrefix(path, "/confirmar") ||
			strings.HasPrefix(path, "/public") {
			c.Next()
			return
		}

		// Extrair subdomínio
		subdomain := c.Param("subdomain")
		if subdomain == "" {
			// Tentar pegar do contexto (caso tenha sido setado pelo SubdomainMiddleware)
			subdomainVal, exists := c.Get("subdomain")
			if exists {
				subdomain = subdomainVal.(string)
			}
		}

		// Se não tem subdomínio, não é uma rota de blog
		if subdomain == "" {
			c.Next()
			return
		}

		// Ignorar parâmetros de preview CSS
		if c.Query("css") != "" {
			c.Next()
			return
		}

		// Construir path para o cache
		cachePath := path
		// Remover prefixo /@/subdomain se existir
		cachePath = strings.TrimPrefix(cachePath, "/@/"+subdomain)
		// Se estiver vazio, é a página inicial
		if cachePath == "" || cachePath == "/" {
			cachePath = "/"
		}

		// Tentar ler do cache
		if Exists(subdomain, cachePath) {
			cachedContent, err := Read(subdomain, cachePath)
			if err == nil {
				log.Printf("Cache HIT: %s%s", subdomain, cachePath)
				c.Data(http.StatusOK, "text/html; charset=utf-8", cachedContent)
				c.Abort()
				return
			}
			log.Printf("Cache read error: %v", err)
		}

		log.Printf("Cache MISS: %s%s", subdomain, cachePath)

		// Criar wrapper para capturar o HTML gerado
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// Armazenar info no contexto para uso posterior
		c.Set("cache_subdomain", subdomain)
		c.Set("cache_path", cachePath)
		c.Set("cache_writer", writer)

		c.Next()

		// Após o controller processar, salvar no cache se for HTML
		if c.Writer.Status() == http.StatusOK {
			contentType := c.Writer.Header().Get("Content-Type")
			if strings.Contains(contentType, "text/html") && writer.body.Len() > 0 {
				if err := Write(subdomain, cachePath, writer.body.Bytes()); err != nil {
					log.Printf("Failed to write cache: %v", err)
				}
			}
		}
	}
}
