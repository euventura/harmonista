package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"harmonista/admin"
	"harmonista/analytics"
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

	db := common.ConnectDb()
	if db == nil {
		log.Fatal("Failed to connect to database")
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Conectar ao banco de analytics (separado)
	analyticsDb := common.ConnectAnalyticsDb()
	analyticsModule := analytics.NewAnalyticsModule(analyticsDb)

	router := gin.Default()

	// Desabilitar trusted proxies já que não usamos proxy reverso
	// Se você usar nginx ou cloudflare no futuro, configure aqui os IPs confiáveis
	router.SetTrustedProxies(nil)

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable not set")
	}

	// Determinar se estamos em modo HTTPS
	certFile := os.Getenv("SSL_CERT_PATH")
	keyFile := os.Getenv("SSL_KEY_PATH")
	useHTTPS := certFile != "" && keyFile != ""

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   useHTTPS, // Secure cookies apenas em HTTPS
	})

	router.Use(sessions.Sessions("harmonista-session", store))

	// Add subdomain middleware
	router.Use(common.SubdomainMiddleware())

	// Add cache middleware
	router.Use(cache.Middleware())

	router.SetFuncMap(map[string]interface{}{
		"now": func() time.Time {
			return time.Now()
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

	siteModule := site.NewSiteModule(db, analyticsModule)
	siteModule.RegisterRoutes(router)

	adminModule := admin.NewAdminModule(db, analyticsModule)
	adminModule.RegisterRoutes(router)

	blogModule := blog.NewBlogModule(db, analyticsModule)
	blogModule.RegisterRoutes(router)

	// Configurar servidores HTTP e HTTPS
	if useHTTPS {
		// Servidor HTTP na porta 80 para redirecionamento
		httpRedirect := &http.Server{
			Addr:         ":80",
			Handler:      createHTTPRedirectHandler(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		// Servidor HTTPS na porta 443
		httpsServer := &http.Server{
			Addr:         ":443",
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		// Iniciar servidor HTTP em goroutine
		go func() {
			log.Println("Starting HTTP redirect server on port 80...")
			if err := httpRedirect.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP redirect server error: %v", err)
			}
		}()

		// Canal para capturar sinais de interrupção
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Iniciar servidor HTTPS em goroutine
		go func() {
			log.Println("Starting HTTPS server on port 443...")
			if err := httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTPS server:", err)
			}
		}()

		// Aguardar sinal de interrupção
		<-quit
		log.Println("Shutting down servers...")

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpRedirect.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		if err := httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server shutdown error: %v", err)
		}

		log.Println("Servers stopped")
	} else {
		// Modo HTTP apenas (desenvolvimento)
		port := os.Getenv("PORT")
		if port == "" {
			port = "80"
		}

		log.Printf("Starting HTTP server on port %s (development mode)...", port)
		if err := router.Run(":" + port); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}
}

// createHTTPRedirectHandler cria um handler que redireciona HTTP para HTTPS
func createHTTPRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirecionar para HTTPS
		target := "https://" + r.Host + r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
