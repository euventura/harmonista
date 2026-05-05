package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"harmonista/admin"
	"harmonista/analytics"
	"harmonista/backoffice"
	"harmonista/blog"
	"harmonista/cache"
	"harmonista/common"
	"harmonista/database"
	"harmonista/site"
)

func main() {
	// Carregar variáveis de ambiente do arquivo .env (se existir)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Conectar ao banco principal (PostgreSQL)
	db := common.ConnectPgDb()
	if db == nil {
		log.Fatal("Failed to connect to PostgreSQL database")
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Conectar ao Valkey para analytics
	valkeyClient := common.ConnectValkey()
	analyticsModule := analytics.NewAnalyticsModule(valkeyClient)

	router := gin.Default()

	// Confiar no proxy reverso local (nginx)
	router.SetTrustedProxies([]string{"127.0.0.1"})

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable not set")
	}

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   strings.HasPrefix(os.Getenv("DOMAIN"), "https"),
		SameSite: http.SameSiteStrictMode,
	})

	router.Use(sessions.Sessions("harmonista-session", store))

	// CSRF: bloquear POST/PUT/DELETE/PATCH cross-origin checando Origin/Referer
	router.Use(common.CSRFOriginCheck())

	// Redirecionar www para non-www (deve vir primeiro)
	router.Use(common.WWWRedirectMiddleware())

	// Add subdomain middleware
	router.Use(common.SubdomainMiddleware())

	// Add cache middleware for blog posts (24 hour cache)
	router.Use(cache.CacheMiddleware(24*time.Hour, analyticsModule))

	router.SetFuncMap(map[string]interface{}{
		"now": func() time.Time {
			return time.Now()
		},
		"version": func() string {
			return time.Now().Format("150405") // HHMMSS format for lightweight cache busting
		},
		"domain": func() string {
			d := os.Getenv("DOMAIN")
			if d == "" {
				return "http://localhost/"
			}
			return d
		},
	})

	router.LoadHTMLGlob("*/views/*.html")

	router.Static("/public", "./public")

	siteModule := site.NewSiteModule(db)
	siteModule.RegisterRoutes(router)

	adminModule := admin.NewAdminModule(db, analyticsModule, valkeyClient)
	adminModule.RegisterRoutes(router)

	backofficeModule := backoffice.NewBackofficeModule(db, valkeyClient)
	backofficeModule.RegisterRoutes(router)

	blogModule := blog.NewBlogModule(db, analyticsModule)
	blogModule.RegisterRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting HTTP server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
