package backoffice

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"harmonista/cache"
	"harmonista/models"
)

type BackofficeModule struct {
	db *gorm.DB
}

func NewBackofficeModule(db *gorm.DB) *BackofficeModule {
	return &BackofficeModule{db: db}
}

func (b *BackofficeModule) RegisterRoutes(router *gin.Engine) {
	backofficeGroup := router.Group("/$")
	{
		backofficeGroup.GET("/login", b.loginPage)
		backofficeGroup.POST("/login", b.loginPost)
		backofficeGroup.GET("/index", b.requireBackofficeAuth, b.index)
		backofficeGroup.POST("/toggle-list-reader/:blogID", b.requireBackofficeAuth, b.toggleListReader)
		backofficeGroup.POST("/toggle-adult/:blogID", b.requireBackofficeAuth, b.toggleAdult)
		backofficeGroup.POST("/validate-user/:userID", b.requireBackofficeAuth, b.validateUser)
		backofficeGroup.POST("/clear-cache/:blogID", b.requireBackofficeAuth, b.clearBlogCache)
		backofficeGroup.POST("/create-blog/:userID", b.requireBackofficeAuth, b.createBlog)
		backofficeGroup.GET("/logout", b.logout)
	}
}

// requireBackofficeAuth middleware que verifica se o usuário está logado e tem email autorizado
func (b *BackofficeModule) requireBackofficeAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("backoffice_user_id")

	if userID == nil {
		c.Redirect(http.StatusFound, "/$/login")
		c.Abort()
		return
	}

	// Buscar usuário e verificar email
	var user models.User
	if err := b.db.First(&user, userID).Error; err != nil {
		c.Redirect(http.StatusFound, "/$/login")
		c.Abort()
		return
	}

	// Verificar se o email está na lista de backoffice
	if !b.isBackofficeEmail(user.Email) {
		session.Clear()
		session.Save()
		c.HTML(http.StatusForbidden, "backoffice_error.html", gin.H{
			"error": "Acesso não autorizado",
		})
		c.Abort()
		return
	}

	c.Set("backoffice_user", user)
	c.Next()
}

// isBackofficeEmail verifica se o email está na lista de emails autorizados
func (b *BackofficeModule) isBackofficeEmail(email string) bool {
	backofficeEmails := os.Getenv("BACKOFFICE_EMAILS")
	if backofficeEmails == "" {
		return false
	}

	emails := strings.Split(backofficeEmails, ",")
	for _, e := range emails {
		if strings.TrimSpace(e) == email {
			return true
		}
	}
	return false
}

func (b *BackofficeModule) loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "backoffice_login.html", gin.H{})
}

func (b *BackofficeModule) loginPost(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Buscar usuário
	var user models.User
	if err := b.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "backoffice_login.html", gin.H{
			"error": "Email ou senha incorretos",
			"email": email,
		})
		return
	}

	// Verificar senha (reutilizando a lógica do admin)
	if !checkPasswordHash(password, user.PasswordHash) {
		c.HTML(http.StatusUnauthorized, "backoffice_login.html", gin.H{
			"error": "Email ou senha incorretos",
			"email": email,
		})
		return
	}

	// Verificar se o email está na lista de backoffice
	if !b.isBackofficeEmail(user.Email) {
		c.HTML(http.StatusForbidden, "backoffice_login.html", gin.H{
			"error": "Você não tem permissão para acessar o backoffice",
			"email": email,
		})
		return
	}

	// Criar sessão
	session := sessions.Default(c)
	session.Set("backoffice_user_id", user.ID)
	session.Save()

	c.Redirect(http.StatusFound, "/$/index")
}

func (b *BackofficeModule) index(c *gin.Context) {
	// Buscar todos os blogs com os usuários (usando Preload)
	var blogs []models.Blog
	if err := b.db.Preload("User").Find(&blogs).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "backoffice_error.html", gin.H{
			"error": "Erro ao carregar blogs",
		})
		return
	}

	// Para cada blog, contar posts
	type BlogWithStats struct {
		Blog      models.Blog
		PostCount int64
	}

	blogsWithStats := make([]BlogWithStats, len(blogs))
	for i, blog := range blogs {
		var postCount int64
		b.db.Model(&models.Post{}).Where("blog_id = ?", blog.ID).Count(&postCount)

		blogsWithStats[i] = BlogWithStats{
			Blog:      blog,
			PostCount: postCount,
		}
	}

	c.HTML(http.StatusOK, "backoffice_index.html", gin.H{
		"blogs": blogsWithStats,
	})
}

func (b *BackofficeModule) toggleListReader(c *gin.Context) {
	blogID := c.Param("blogID")

	var blog models.Blog
	if err := b.db.First(&blog, blogID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog não encontrado"})
		return
	}

	blog.IsListReader = !blog.IsListReader
	if err := b.db.Save(&blog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"isListReader": blog.IsListReader,
	})
}

func (b *BackofficeModule) toggleAdult(c *gin.Context) {
	blogID := c.Param("blogID")

	var blog models.Blog
	if err := b.db.First(&blog, blogID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog não encontrado"})
		return
	}

	blog.IsAdult = !blog.IsAdult
	if err := b.db.Save(&blog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"isAdult": blog.IsAdult,
	})
}

func (b *BackofficeModule) validateUser(c *gin.Context) {
	userID := c.Param("userID")

	var user models.User
	if err := b.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		return
	}

	user.EmailVerified = true
	user.EmailVerificationToken = ""

	if err := b.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao validar usuário"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"emailVerified": user.EmailVerified,
	})
}

func (b *BackofficeModule) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Redirect(http.StatusFound, "/$/login")
}

// clearBlogCache limpa todo o cache de um blog
func (b *BackofficeModule) clearBlogCache(c *gin.Context) {
	blogID := c.Param("blogID")

	var blog models.Blog
	if err := b.db.First(&blog, blogID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog não encontrado"})
		return
	}

	// Limpar todo o cache do blog
	if err := cache.ClearAllBlogCache(blog.Subdomain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao limpar cache: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache limpo com sucesso",
	})
}

// createBlog cria um novo blog para o usuário especificado com título e subdomain baseados em timestamp
func (b *BackofficeModule) createBlog(c *gin.Context) {
	userID := c.Param("userID")

	var user models.User
	if err := b.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		return
	}

	// Gerar timestamp como base para título e subdomain
	timestamp := time.Now().Unix()
	subdomain := fmt.Sprintf("blog%d", timestamp)
	title := fmt.Sprintf("Blog %d", timestamp)

	// Verificar se subdomain já existe (extremamente improvável, mas por segurança)
	var existingBlog models.Blog
	if err := b.db.Where("subdomain = ?", subdomain).First(&existingBlog).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Subdomain já existe"})
		return
	}

	// Criar o novo blog
	blog := models.Blog{
		UserID:      user.ID,
		Title:       title,
		Description: "",
		Subdomain:   subdomain,
	}

	if err := b.db.Create(&blog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar blog: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Blog criado com sucesso",
		"subdomain": subdomain,
		"title":     title,
		"blogID":    blog.ID,
	})
}

// checkPasswordHash verifica se a senha corresponde ao hash
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
