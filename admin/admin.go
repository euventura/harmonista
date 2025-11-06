package admin

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"harmonista/analytics"
	emailpkg "harmonista/email"
	"harmonista/models"
)

type AdminModule struct {
	db        *gorm.DB
	analytics *analytics.AnalyticsModule
}

func NewAdminModule(db *gorm.DB, analyticsModule *analytics.AnalyticsModule) *AdminModule {
	return &AdminModule{
		db:        db,
		analytics: analyticsModule,
	}
}

func (a *AdminModule) RegisterRoutes(router *gin.Engine) {
	router.GET("/login", a.loginPage)
	router.POST("/login", a.loginPost)
	router.GET("/cadastrar", a.cadastroPage)
	router.POST("/cadastro", a.cadastroPost)
	router.GET("/confirmar/:token", a.confirmEmail)
	router.GET("/admin", a.adminRoot)

	adminGroup := router.Group("/admin/:subdomain")
	adminGroup.Use(a.requireAuth, a.loadBlog)
	{
		adminGroup.GET("/", a.index)
		adminGroup.POST("/inicio", a.updateBlogSettings)
		adminGroup.GET("/posts", a.listPosts)
		adminGroup.GET("/post/novo", a.newPost)
		adminGroup.POST("/post/salvar", a.savePost)
		adminGroup.POST("/post/autosave", a.autoSavePost)
		adminGroup.GET("/post/:id", a.editPost)
		adminGroup.POST("/post/:id", a.updatePost)
		adminGroup.POST("/post/:id/autosave", a.autoSaveExistingPost)
		adminGroup.DELETE("/post/:id", a.deletePost)
		adminGroup.GET("/pages", a.listPages)
		adminGroup.GET("/page/novo", a.newPage)
		adminGroup.POST("/page/salvar", a.savePage)
		adminGroup.POST("/page/autosave", a.autoSavePage)
		adminGroup.GET("/page/:id", a.editPage)
		adminGroup.POST("/page/:id", a.updatePage)
		adminGroup.POST("/page/:id/autosave", a.autoSaveExistingPage)
		adminGroup.DELETE("/page/:id", a.deletePage)
		adminGroup.GET("/tema", a.theme)
		adminGroup.POST("/tema", a.saveTheme)
		adminGroup.POST("/tema/aplicar", a.applyTheme)
		adminGroup.GET("/menu", a.menu)
		adminGroup.POST("/menu", a.updateMenu)
		adminGroup.GET("/config", a.config)
		adminGroup.POST("/config", a.updateConfig)
		adminGroup.GET("/visitas", a.analytics_page)
	}

	router.GET("/admin/dashboard", a.requireAuth, a.dashboard)
	router.GET("/admin/logout", a.logout)

}

func (a *AdminModule) requireAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID == nil {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}

	c.Set("user_id", userID)
	c.Next()
}

func (a *AdminModule) loadBlog(c *gin.Context) {
	subdomain := c.Param("subdomain")
	if subdomain == "" {
		c.Next()
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.Next()
		return
	}

	blog, err := a.getBlogBySubdomain(subdomain, userID.(int))
	if err != nil {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{"error": "Blog não encontrado"})
		c.Abort()
		return
	}

	c.Set("blog", blog)
	c.Next()
}

func (a *AdminModule) adminRoot(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func (a *AdminModule) loginPage(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	c.HTML(http.StatusOK, "admin_login.html", gin.H{})
}

func (a *AdminModule) loginPost(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "admin_login.html", gin.H{
			"error": "Email ou senha incorretos",
			"email": email,
		})
		return
	}

	if !checkPasswordHash(password, user.PasswordHash) {
		c.HTML(http.StatusUnauthorized, "admin_login.html", gin.H{
			"error": "Email ou senha incorretos",
			"email": email,
		})
		return
	}

	if !user.EmailVerified {
		c.HTML(http.StatusUnauthorized, "admin_login.html", gin.H{
			"error": "Email não verificado. Por favor, verifique sua caixa de entrada e confirme seu email.",
			"email": email,
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Save()

	c.Redirect(http.StatusFound, "/admin/dashboard")
}

func (a *AdminModule) cadastroPage(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	c.HTML(http.StatusOK, "admin_cadastro.html", gin.H{})
}

func (a *AdminModule) cadastroPost(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	subdomain := c.PostForm("subdomain")
	title := c.PostForm("title")
	description := c.PostForm("description")

	// Dados para reenviar em caso de erro (não incluir senha por segurança)
	formData := gin.H{
		"email":       email,
		"subdomain":   subdomain,
		"title":       title,
		"description": description,
	}

	var existingUser models.User
	if err := a.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		formData["error"] = "Este email já está cadastrado"
		c.HTML(http.StatusBadRequest, "admin_cadastro.html", formData)
		return
	}

	var existingBlog models.Blog
	if err := a.db.Where("subdomain = ?", subdomain).First(&existingBlog).Error; err == nil {
		formData["error"] = "Este subdomínio já está em uso"
		c.HTML(http.StatusBadRequest, "admin_cadastro.html", formData)
		return
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		formData["error"] = "Erro ao criar conta"
		c.HTML(http.StatusInternalServerError, "admin_cadastro.html", formData)
		return
	}

	// Gerar token de verificação
	verificationToken, err := generateToken()
	if err != nil {
		formData["error"] = "Erro ao gerar token de verificação"
		c.HTML(http.StatusInternalServerError, "admin_cadastro.html", formData)
		return
	}

	user := models.User{
		Email:                  email,
		PasswordHash:           passwordHash,
		EmailVerified:          false,
		EmailVerificationToken: verificationToken,
	}

	if err := a.db.Create(&user).Error; err != nil {
		formData["error"] = "Erro ao criar conta"
		c.HTML(http.StatusInternalServerError, "admin_cadastro.html", formData)
		return
	}

	blog := models.Blog{
		UserID:      user.ID,
		Title:       title,
		Description: description,
		Subdomain:   subdomain,
	}

	if err := a.db.Create(&blog).Error; err != nil {
		a.db.Delete(&user)
		formData["error"] = "Erro ao criar blog"
		c.HTML(http.StatusInternalServerError, "admin_cadastro.html", formData)
		return
	}

	// Enviar email de verificação
	emailService := emailpkg.NewEmailService()
	emailErr := emailService.SendVerificationEmail(user.Email, verificationToken)

	// Sempre mostra a página de sucesso, mas informa se houve erro no envio
	if emailErr != nil {
		log.Printf("Erro ao enviar email de verificação para %s: %v", user.Email, emailErr)
		c.HTML(http.StatusOK, "admin_cadastro_success.html", gin.H{
			"email":      user.Email,
			"emailError": "Erro ao enviar email: " + emailErr.Error() + ". Entre em contato com o suporte.",
		})
		return
	}

	c.HTML(http.StatusOK, "admin_cadastro_success.html", gin.H{
		"email": user.Email,
	})
}

func (a *AdminModule) confirmEmail(c *gin.Context) {
	token := c.Param("token")

	var user models.User
	if err := a.db.Where("email_verification_token = ?", token).First(&user).Error; err != nil {
		c.HTML(http.StatusNotFound, "admin_confirm_email.html", gin.H{
			"success": false,
			"message": "Token inválido ou expirado",
		})
		return
	}

	if user.EmailVerified {
		c.HTML(http.StatusOK, "admin_confirm_email.html", gin.H{
			"success": true,
			"message": "Email já confirmado anteriormente",
		})
		return
	}

	user.EmailVerified = true
	user.EmailVerificationToken = ""

	if err := a.db.Save(&user).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_confirm_email.html", gin.H{
			"success": false,
			"message": "Erro ao confirmar email",
		})
		return
	}

	c.HTML(http.StatusOK, "admin_confirm_email.html", gin.H{
		"success": true,
		"message": "Email confirmado com sucesso! Você já pode fazer login.",
	})
}

func (a *AdminModule) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Redirect(http.StatusFound, "/login")
}

func (a *AdminModule) dashboard(c *gin.Context) {
	userID := c.GetInt("user_id")

	var blogs []models.Blog
	if err := a.db.Where("user_id = ?", userID).Find(&blogs).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao carregar blogs",
		})
		return
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"blogs": blogs,
	})
}

func (a *AdminModule) theme(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	// Listar arquivos CSS da pasta /public/css/temas
	themesDir := "./public/css/temas"
	themes := []string{}

	files, err := ioutil.ReadDir(themesDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".css" {
				themes = append(themes, file.Name())
			}
		}
	}

	c.HTML(http.StatusOK, "admin_theme.html", gin.H{
		"blog":      blog,
		"subdomain": subdomain,
		"themes":    themes,
	})
}

func (a *AdminModule) saveTheme(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	theme := c.PostForm("theme")
	blog.Theme = theme

	if err := a.db.Save(blog).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao salvar tema",
			"blog":  blog,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/tema")
}

func (a *AdminModule) applyTheme(c *gin.Context) {
	blogData, exists := c.Get("blog")
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog não encontrado"})
		return
	}
	blog := blogData.(*models.Blog)

	themePath := c.PostForm("theme_path")
	if themePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Caminho do tema não fornecido"})
		return
	}

	// Ler o conteúdo do arquivo CSS
	fullPath := filepath.Join("./public/css/temas", themePath)
	cssContent, err := ioutil.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler arquivo CSS"})
		return
	}

	// Salvar o conteúdo em blog.Theme
	blog.Theme = string(cssContent)
	if err := a.db.Save(blog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar tema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tema aplicado com sucesso"})
}

func (a *AdminModule) menu(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	c.HTML(http.StatusOK, "admin_menu_page.html", gin.H{
		"subdomain": subdomain,
		"blog":      blog,
	})
}

func (a *AdminModule) updateMenu(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	navInput := c.PostForm("nav")

	// Filtrar apenas links markdown
	filteredNav := filterMarkdownLinks(navInput)

	blog.Nav = filteredNav

	if err := a.db.Save(blog).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_menu_page.html", gin.H{
			"error":     "Erro ao salvar menu",
			"blog":      blog,
			"subdomain": subdomain,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/menu")
}

func (a *AdminModule) config(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	c.HTML(http.StatusOK, "admin_config.html", gin.H{
		"subdomain": subdomain,
		"blog":      blog,
	})
}

func (a *AdminModule) updateConfig(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)
	userID := c.GetInt("user_id")

	newSubdomain := c.PostForm("subdomain")
	nav := c.PostForm("nav")
	password := c.PostForm("password")
	isAdult := c.PostForm("isAdult") == "1"
	isListReader := c.PostForm("IsListReader") == "1"

	// Validate subdomain change if different
	if newSubdomain != blog.Subdomain {
		var existingBlog models.Blog
		if err := a.db.Where("subdomain = ?", newSubdomain).First(&existingBlog).Error; err == nil {
			c.HTML(http.StatusBadRequest, "admin_config.html", gin.H{
				"error": "Este subdomínio já está em uso",
				"blog":  blog,
			})
			return
		}
		blog.Subdomain = newSubdomain
	}

	blog.Nav = nav
	blog.IsAdult = isAdult
	blog.IsListReader = isListReader

	if err := a.db.Save(blog).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_config.html", gin.H{
			"error": "Erro ao salvar configurações",
			"blog":  blog,
		})
		return
	}

	// Update password if provided
	if password != "" {
		var user models.User
		if err := a.db.First(&user, userID).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "admin_config.html", gin.H{
				"error": "Erro ao atualizar senha",
				"blog":  blog,
			})
			return
		}

		passwordHash, err := hashPassword(password)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "admin_config.html", gin.H{
				"error": "Erro ao atualizar senha",
				"blog":  blog,
			})
			return
		}

		user.PasswordHash = passwordHash
		if err := a.db.Save(&user).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "admin_config.html", gin.H{
				"error": "Erro ao atualizar senha",
				"blog":  blog,
			})
			return
		}
	}

	// Redirect to new subdomain if changed
	if newSubdomain != subdomain {
		c.Redirect(http.StatusFound, "/admin/"+newSubdomain+"/config")
	} else {
		c.Redirect(http.StatusFound, "/admin/"+subdomain+"/config")
	}
}

func (a *AdminModule) getBlogBySubdomain(subdomain string, userID int) (*models.Blog, error) {
	var blog models.Blog
	err := a.db.Where("subdomain = ? AND user_id = ?", subdomain, userID).First(&blog).Error
	return &blog, err
}

func (a *AdminModule) index(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blog, _ := c.Get("blog")

	c.HTML(http.StatusOK, "admin_index.html", gin.H{
		"subdomain": subdomain,
		"blog":      blog,
	})
}

func (a *AdminModule) updateBlogSettings(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	title := c.PostForm("title")
	description := c.PostForm("description")

	blog.Title = title
	blog.Description = description

	if err := a.db.Save(blog).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao salvar",
			"blog":  blog,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/")
}

func (a *AdminModule) listPosts(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var posts []models.Post
	if err := a.db.Where("blog_id = ?", blog.ID).Order("created_at DESC").Find(&posts).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao carregar posts",
			"blog":  blog,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_list_posts.html", gin.H{
		"subdomain": subdomain,
		"posts":     posts,
		"blog":      blog,
	})
}

func (a *AdminModule) newPost(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blog, _ := c.Get("blog")

	c.HTML(http.StatusOK, "admin_new_post.html", gin.H{
		"subdomain": subdomain,
		"blog":      blog,
	})
}

func (a *AdminModule) savePost(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	title := c.PostForm("title")
	content := c.PostForm("content")
	tags := c.PostForm("tags")
	action := c.PostForm("action")

	slug := generateSlug(title)
	draft := action == "save_draft"

	post := models.Post{
		BlogID:    blog.ID,
		Title:     title,
		Slug:      slug,
		Content:   content,
		Draft:     draft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := a.db.Create(&post).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao criar post",
			"blog":  blog,
		})
		return
	}

	if tags != "" {
		if err := a.processPostTags(blog.ID, int(post.ID), tags); err != nil {
			c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
				"error": "Erro ao processar tags: " + err.Error(),
				"blog":  blog,
			})
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/posts")
}

// autoSavePost salva automaticamente o conteúdo de um novo post (rascunho)
func (a *AdminModule) autoSavePost(c *gin.Context) {

	var request struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Tags    string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Para auto-save de novos posts, vamos usar um sistema simples
	// que salva no localStorage do navegador por enquanto
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto-save simulado (localStorage)",
	})
}

// autoSaveExistingPost salva automaticamente o conteúdo de um post existente (apenas rascunhos)
func (a *AdminModule) autoSaveExistingPost(c *gin.Context) {
	postID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var post models.Post
	if err := a.db.Where("id = ? AND blog_id = ?", postID, blog.ID).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post não encontrado"})
		return
	}

	// Só permite auto-save em rascunhos
	if !post.Draft {
		c.JSON(http.StatusForbidden, gin.H{"error": "Auto-save só é permitido em rascunhos"})
		return
	}

	var request struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Tags    string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Atualiza apenas o conteúdo, mantendo como rascunho
	updates := map[string]interface{}{
		"content":    request.Content,
		"updated_at": time.Now(),
	}

	if request.Title != "" {
		updates["title"] = request.Title
		updates["slug"] = generateSlug(request.Title)
	}

	if err := a.db.Model(&post).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar automaticamente"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Rascunho salvo automaticamente",
		"saved_at": time.Now().Format("15:04:05"),
	})
}

func (a *AdminModule) editPost(c *gin.Context) {
	subdomain := c.Param("subdomain")
	postID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var post models.Post
	if err := a.db.Where("id = ? AND blog_id = ?", postID, blog.ID).First(&post).Error; err != nil {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Post não encontrado",
			"blog":  blog,
		})
		return
	}

	tags := a.getPostTags(int(post.ID))

	// Buscar contagem de visitas do post
	postIDInt, _ := strconv.Atoi(postID)
	visitCount := int64(0)
	if a.analytics != nil {
		visitCount = a.analytics.GetPostVisitCount(postIDInt)
	}

	c.HTML(http.StatusOK, "admin_edit_post.html", gin.H{
		"subdomain":  subdomain,
		"post":       post,
		"blog":       blog,
		"tags":       tags,
		"visitCount": visitCount,
	})
}

func (a *AdminModule) updatePost(c *gin.Context) {
	subdomain := c.Param("subdomain")
	postID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var post models.Post
	if err := a.db.Where("id = ? AND blog_id = ?", postID, blog.ID).First(&post).Error; err != nil {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Post não encontrado",
			"blog":  blog,
		})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")
	tags := c.PostForm("tags")
	action := c.PostForm("action")

	post.Title = title
	post.Content = content
	post.UpdatedAt = time.Now()

	switch action {
	case "publish":
		post.Draft = false
	case "unpublish":
		post.Draft = true
	case "save", "update":
	}

	if err := a.db.Save(&post).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao atualizar post",
			"blog":  blog,
		})
		return
	}

	if tags != "" {
		if err := a.processPostTags(blog.ID, int(post.ID), tags); err != nil {
			c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
				"error": "Erro ao processar tags: " + err.Error(),
				"blog":  blog,
			})
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/posts")
}

func (a *AdminModule) deletePost(c *gin.Context) {
	postID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	postIDInt, err := strconv.Atoi(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	result := a.db.Where("id = ? AND blog_id = ?", postIDInt, blog.ID).Delete(&models.Post{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar post"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post não encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post deletado com sucesso"})
}

func (a *AdminModule) listPages(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var pages []models.Page
	if err := a.db.Where("blog_id = ?", blog.ID).Order("created_at DESC").Find(&pages).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao carregar páginas",
			"blog":  blog,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_list_pages.html", gin.H{
		"subdomain": subdomain,
		"pages":     pages,
		"blog":      blog,
	})
}

func (a *AdminModule) newPage(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blog, _ := c.Get("blog")

	c.HTML(http.StatusOK, "admin_new_page.html", gin.H{
		"subdomain": subdomain,
		"blog":      blog,
	})
}

func (a *AdminModule) savePage(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	title := c.PostForm("title")
	content := c.PostForm("content")
	action := c.PostForm("action")

	slug := generateSlug(title)
	draft := action == "save_draft"

	page := models.Page{
		BlogID:    blog.ID,
		Title:     title,
		Slug:      slug,
		Content:   content,
		Draft:     draft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := a.db.Create(&page).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao criar página",
			"blog":  blog,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/pages")
}

// autoSavePage salva automaticamente o conteúdo de uma nova página (rascunho)
func (a *AdminModule) autoSavePage(c *gin.Context) {
	var request struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Para auto-save de novas páginas, vamos usar um sistema simples
	// que salva no localStorage do navegador por enquanto
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto-save simulado (localStorage)",
	})
}

// autoSaveExistingPage salva automaticamente o conteúdo de uma página existente (apenas rascunhos)
func (a *AdminModule) autoSaveExistingPage(c *gin.Context) {
	pageID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var page models.Page
	if err := a.db.Where("id = ? AND blog_id = ?", pageID, blog.ID).First(&page).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Página não encontrada"})
		return
	}

	// Só permite auto-save em rascunhos
	if !page.Draft {
		c.JSON(http.StatusForbidden, gin.H{"error": "Auto-save só é permitido em rascunhos"})
		return
	}

	var request struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Atualiza apenas o conteúdo, mantendo como rascunho
	updates := map[string]interface{}{
		"content":    request.Content,
		"updated_at": time.Now(),
	}

	if request.Title != "" {
		updates["title"] = request.Title
		updates["slug"] = generateSlug(request.Title)
	}

	if err := a.db.Model(&page).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar automaticamente"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Rascunho salvo automaticamente",
		"saved_at": time.Now().Format("15:04:05"),
	})
}

func (a *AdminModule) editPage(c *gin.Context) {
	subdomain := c.Param("subdomain")
	pageID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var page models.Page
	if err := a.db.Where("id = ? AND blog_id = ?", pageID, blog.ID).First(&page).Error; err != nil {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Página não encontrada",
			"blog":  blog,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_edit_page.html", gin.H{
		"subdomain": subdomain,
		"page":      page,
		"blog":      blog,
	})
}

func (a *AdminModule) updatePage(c *gin.Context) {
	subdomain := c.Param("subdomain")
	pageID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	var page models.Page
	if err := a.db.Where("id = ? AND blog_id = ?", pageID, blog.ID).First(&page).Error; err != nil {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Página não encontrada",
			"blog":  blog,
		})
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")
	action := c.PostForm("action")

	page.Title = title
	page.Content = content
	page.UpdatedAt = time.Now()

	switch action {
	case "publish":
		page.Draft = false
	case "unpublish":
		page.Draft = true
	case "save", "update":
	}

	if err := a.db.Save(&page).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "admin_error.html", gin.H{
			"error": "Erro ao atualizar página",
			"blog":  blog,
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/"+subdomain+"/pages")
}

func (a *AdminModule) deletePage(c *gin.Context) {
	pageID := c.Param("id")
	blogData, _ := c.Get("blog")
	blog := blogData.(*models.Blog)

	pageIDInt, err := strconv.Atoi(pageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	result := a.db.Where("id = ? AND blog_id = ?", pageIDInt, blog.ID).Delete(&models.Page{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar página"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Página não encontrada"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Página deletada com sucesso"})
}

func (a *AdminModule) getPostTags(postID int) string {
	var postTags []models.PostTag
	if err := a.db.Where("post_id = ?", postID).Find(&postTags).Error; err != nil {
		return ""
	}

	if len(postTags) == 0 {
		return ""
	}

	var tagIDs []int
	for _, pt := range postTags {
		tagIDs = append(tagIDs, pt.TagID)
	}

	var tags []models.Tag
	if err := a.db.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return ""
	}

	var tagTitles []string
	for _, tag := range tags {
		tagTitles = append(tagTitles, tag.Title)
	}

	return strings.Join(tagTitles, ", ")
}

func (a *AdminModule) processPostTags(blogID int, postID int, tagsString string) error {
	result := a.db.Where("post_id = ?", postID).Delete(&models.PostTag{})
	if result.Error != nil {
		return result.Error
	}

	if tagsString == "" {
		return nil
	}

	tagNames := strings.Split(tagsString, ",")
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		strings.ToLower(tagName)
		var tag models.Tag
		err := a.db.Where("title = ?", tagName).First(&tag).Error

		if err == gorm.ErrRecordNotFound {
			tag = models.Tag{
				Title: tagName,
			}
			if err := a.db.Create(&tag).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		var existingPostTag models.PostTag
		err = a.db.Where("post_id = ? AND tag_id = ?", postID, tag.ID).First(&existingPostTag).Error

		if err == gorm.ErrRecordNotFound {
			postTag := models.PostTag{
				PostID: postID,
				TagID:  int(tag.ID),
			}
			if err := a.db.Create(&postTag).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}

func (a *AdminModule) createOrAssignTag(blogID int, postID int, tagTitle string) error {
	var tag models.Tag
	err := a.db.Where("title = ?", tagTitle).First(&tag).Error

	if err == gorm.ErrRecordNotFound {
		tag = models.Tag{
			Title: tagTitle,
		}
		if err := a.db.Create(&tag).Error; err != nil {
			return err
		}
	}

	var existingPostTag models.PostTag
	err = a.db.Where("post_id = ? AND tag_id = ?", postID, tag.ID).First(&existingPostTag).Error

	if err == gorm.ErrRecordNotFound {
		postTag := models.PostTag{
			TagID:  int(tag.ID),
			PostID: postID,
		}
		if err := a.db.Create(&postTag).Error; err != nil {
			return err
		}
	}

	return nil
}

func generateSlug(title string) string {
	// Mapa de caracteres acentuados para suas versões sem acento
	accentMap := map[rune]rune{
		'á': 'a', 'à': 'a', 'ã': 'a', 'â': 'a', 'ä': 'a', 'å': 'a', 'ā': 'a',
		'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e', 'ē': 'e',
		'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ī': 'i',
		'ó': 'o', 'ò': 'o', 'õ': 'o', 'ô': 'o', 'ö': 'o', 'ø': 'o', 'ō': 'o',
		'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ū': 'u',
		'ç': 'c', 'ć': 'c', 'č': 'c',
		'ñ': 'n', 'ń': 'n',
		'ý': 'y', 'ÿ': 'y',
		'ß': 's',
		// Versões maiúsculas também
		'Á': 'a', 'À': 'a', 'Ã': 'a', 'Â': 'a', 'Ä': 'a', 'Å': 'a', 'Ā': 'a',
		'É': 'e', 'È': 'e', 'Ê': 'e', 'Ë': 'e', 'Ē': 'e',
		'Í': 'i', 'Ì': 'i', 'Î': 'i', 'Ï': 'i', 'Ī': 'i',
		'Ó': 'o', 'Ò': 'o', 'Õ': 'o', 'Ô': 'o', 'Ö': 'o', 'Ø': 'o', 'Ō': 'o',
		'Ú': 'u', 'Ù': 'u', 'Û': 'u', 'Ü': 'u', 'Ū': 'u',
		'Ç': 'c', 'Ć': 'c', 'Č': 'c',
		'Ñ': 'n', 'Ń': 'n',
		'Ý': 'y', 'Ÿ': 'y',
	}

	// Primeiro, converter para minúsculas e remover acentos
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		// Se encontrar um caractere acentuado no mapa, substitui
		if replacement, exists := accentMap[r]; exists {
			return replacement
		}
		return r
	}, slug)

	// Depois, manter apenas caracteres válidos para slug
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == ' ' {
			return '-'
		}
		return -1 // Remove caractere
	}, slug)

	// Limpar hífens duplos e nas bordas
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	return slug
}

func filterMarkdownLinks(input string) string {
	// Regex para capturar links markdown no formato [texto](url)
	re := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	matches := re.FindAllString(input, -1)

	if len(matches) == 0 {
		return ""
	}

	// Junta todos os links com espaço
	return strings.Join(matches, " ")
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Structs para dados do analytics com porcentagens calculadas
type DayVisitChart struct {
	Date       string
	Count      int64
	Percentage float64
}

type PostVisitChart struct {
	PostID     int
	PostTitle  string
	Count      int64
	Percentage float64
}

func (a *AdminModule) analytics_page(c *gin.Context) {
	subdomain := c.Param("subdomain")
	blogData, exists := c.Get("blog")
	if !exists {
		c.HTML(http.StatusNotFound, "admin_error.html", gin.H{
			"error": "Blog não encontrado",
		})
		return
	}
	blog := blogData.(*models.Blog)

	// Se analytics não está configurado, mostrar mensagem
	if a.analytics == nil {
		c.HTML(http.StatusOK, "admin_analytics.html", gin.H{
			"subdomain":        subdomain,
			"blog":             blog,
			"analyticsEnabled": false,
		})
		return
	}

	// Buscar visitas por dia dos últimos 30 dias
	visitsByDay := a.analytics.GetVisitsByDay(blog.ID, 15)

	// Buscar top 10 posts dos últimos 30 dias
	topPosts := a.analytics.GetTopPosts(blog.ID, 30, 10)

	// Buscar títulos dos posts
	for i := range topPosts {
		var post models.Post
		if err := a.db.First(&post, topPosts[i].PostID).Error; err == nil {
			topPosts[i].PostTitle = post.Title
		} else {
			topPosts[i].PostTitle = "Post não encontrado"
		}
	}

	// Calcular valor máximo para normalização dos gráficos
	maxVisitsPerDay := int64(1)
	for _, day := range visitsByDay {
		if day.Count > maxVisitsPerDay {
			maxVisitsPerDay = day.Count
		}
	}

	maxVisitsPerPost := int64(1)
	for _, post := range topPosts {
		if post.Count > maxVisitsPerPost {
			maxVisitsPerPost = post.Count
		}
	}

	// Converter para structs com porcentagens calculadas
	dayCharts := make([]DayVisitChart, len(visitsByDay))
	for i, day := range visitsByDay {
		percentage := 0.0
		if maxVisitsPerDay > 0 {
			percentage = (float64(day.Count) / float64(maxVisitsPerDay)) * 100
		}
		dayCharts[i] = DayVisitChart{
			Date:       day.Date,
			Count:      day.Count,
			Percentage: percentage,
		}
	}

	postCharts := make([]PostVisitChart, len(topPosts))
	for i, post := range topPosts {
		percentage := 0.0
		if maxVisitsPerPost > 0 {
			percentage = (float64(post.Count) / float64(maxVisitsPerPost)) * 100
		}
		postCharts[i] = PostVisitChart{
			PostID:     post.PostID,
			PostTitle:  post.PostTitle,
			Count:      post.Count,
			Percentage: percentage,
		}
	}

	c.HTML(http.StatusOK, "admin_analytics.html", gin.H{
		"subdomain":        subdomain,
		"blog":             blog,
		"analyticsEnabled": true,
		"visitsByDay":      dayCharts,
		"topPosts":         postCharts,
	})
}
