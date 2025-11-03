package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"harmonista/admin"
	"harmonista/blog"
	"harmonista/common"
	"harmonista/database"
)

func main() {
	db := common.ConnectDb()
	if db == nil {
		log.Fatal("Failed to connect to database")
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	router := gin.Default()

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable not set")
	}

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   false,
	})

	router.Use(sessions.Sessions("harmonista-session", store))

	router.SetFuncMap(map[string]interface{}{
		"now": func() time.Time {
			return time.Now()
		},
		"domain": func() string {
			d := os.Getenv("DOMAIN")
			if d == "" {
				return "http://localhost:8080"
			}
			return d
		},
	})

	router.LoadHTMLGlob("*/views/*.html")

	router.Static("/public", "./public")

	adminModule := admin.NewAdminModule(db)
	adminModule.RegisterRoutes(router)

	blogModule := blog.NewBlogModule(db)
	blogModule.RegisterRoutes(router)

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "home.html", gin.H{
			"title": "Harmonista - Plataforma de Blogs",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
