package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BlogEvent representa um evento de visita no blog
type BlogEvent struct {
	ID        uint      `gorm:"primary_key;autoIncrement"`
	BlogID    int       `gorm:"not null;index"`
	PostID    *int      `gorm:"index"` // nullable - para quando for visita a um post específico
	CookieID  string    `gorm:"not null;index"`
	Event     string    `gorm:"not null;default:'visit'"` // default "visit"
	IP        string    `gorm:"not null"`
	Pais      *string   // nullable
	Lingua    *string   // nullable
	Navegador *string   // nullable
	CreatedAt time.Time `gorm:"index"`
}

// AnalyticsModule gerencia o tracking de analytics
type AnalyticsModule struct {
	db *gorm.DB
}

// NewAnalyticsModule cria uma nova instância do módulo de analytics
func NewAnalyticsModule(db *gorm.DB) *AnalyticsModule {
	if db == nil {
		log.Println("Analytics DB is nil, analytics will be disabled")
		return nil
	}

	// Executar migration da tabela blog_events
	if err := db.AutoMigrate(&BlogEvent{}); err != nil {
		log.Printf("Error migrating blog_events table: %v", err)
		return nil
	}

	log.Println("Analytics module initialized successfully")
	return &AnalyticsModule{db: db}
}

// TrackVisit registra uma visita no banco de dados de analytics
func (a *AnalyticsModule) TrackVisit(c *gin.Context, blogID int, postID *int) {
	if a == nil || a.db == nil {
		return // Analytics desabilitado
	}

	// Obter ou criar cookie ID para identificar visitante único
	cookieID := a.getOrCreateCookieID(c)

	// Capturar IP
	ip := a.getClientIP(c)

	// Capturar User-Agent e extrair informações
	userAgent := c.Request.UserAgent()
	navegador := a.extractBrowser(userAgent)

	// Capturar Accept-Language para detectar idioma
	lingua := a.extractLanguage(c)

	// Por enquanto, país fica como nil (pode ser implementado com GeoIP no futuro)
	var pais *string = nil

	event := BlogEvent{
		BlogID:    blogID,
		PostID:    postID,
		CookieID:  cookieID,
		Event:     "visit",
		IP:        ip,
		Pais:      pais,
		Lingua:    lingua,
		Navegador: navegador,
		CreatedAt: time.Now(),
	}

	// Salvar no banco de forma assíncrona para não impactar performance
	go func() {
		if err := a.db.Create(&event).Error; err != nil {
			log.Printf("Error saving analytics event: %v", err)
		}
	}()
}

// getOrCreateCookieID obtém ou cria um cookie ID único para o visitante
func (a *AnalyticsModule) getOrCreateCookieID(c *gin.Context) string {
	cookieName := "harmonista_visitor_id"

	// Tentar obter cookie existente
	if cookie, err := c.Cookie(cookieName); err == nil && cookie != "" {
		return cookie
	}

	// Criar novo ID baseado em timestamp + IP + User-Agent
	data := time.Now().String() + c.ClientIP() + c.Request.UserAgent()
	hash := sha256.Sum256([]byte(data))
	cookieID := hex.EncodeToString(hash[:])

	// Definir cookie com duração de 2 anos
	c.SetCookie(
		cookieName,
		cookieID,
		60*60*24*365*2, // 2 anos
		"/",
		"",
		false, // secure - seria true em HTTPS
		true,  // httpOnly
	)

	return cookieID
}

// getClientIP obtém o IP real do cliente, considerando proxies
func (a *AnalyticsModule) getClientIP(c *gin.Context) string {
	// Tentar obter de headers comuns de proxy
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For pode ter múltiplos IPs, pegar o primeiro
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}

	// Fallback para IP direto
	return c.ClientIP()
}

// extractBrowser extrai o nome do navegador do User-Agent
func (a *AnalyticsModule) extractBrowser(userAgent string) *string {
	if userAgent == "" {
		return nil
	}

	ua := strings.ToLower(userAgent)
	var browser string

	// Ordem importa - verificar navegadores mais específicos primeiro
	switch {
	case strings.Contains(ua, "edg"):
		browser = "Edge"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg"):
		browser = "Chrome"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr"):
		browser = "Opera"
	case strings.Contains(ua, "msie") || strings.Contains(ua, "trident"):
		browser = "Internet Explorer"
	default:
		browser = "Other"
	}

	return &browser
}

// extractLanguage extrai o idioma preferido do Accept-Language header
func (a *AnalyticsModule) extractLanguage(c *gin.Context) *string {
	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang == "" {
		return nil
	}

	// Accept-Language format: "en-US,en;q=0.9,pt-BR;q=0.8"
	// Pegar apenas o primeiro idioma (mais preferido)
	parts := strings.Split(acceptLang, ",")
	if len(parts) > 0 {
		lang := strings.TrimSpace(parts[0])
		// Remover qualquer parâmetro de qualidade
		lang = strings.Split(lang, ";")[0]
		return &lang
	}

	return nil
}
