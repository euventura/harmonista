package email

import (
	"fmt"
	"net/smtp"
	"os"
)

type EmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewEmailService() *EmailService {
	return &EmailService{
		host:     os.Getenv("SMTP_HOST"),
		port:     os.Getenv("SMTP_PORT"),
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

func (e *EmailService) SendVerificationEmail(to, token string) error {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "http://localhost:8080"
	}

	verificationLink := fmt.Sprintf("%s/confirmar/%s", domain, token)

	subject := "Confirme seu email - Harmonista"
	body := fmt.Sprintf(`
Olá!

Obrigado por se cadastrar na Harmonista.

Para confirmar seu email e ativar sua conta, clique no link abaixo:

%s

Se você não se cadastrou na Harmonista, ignore este email.

---
Harmonista - Plataforma de Blogs
`, verificationLink)

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", e.from, to, subject, body)

	auth := smtp.PlainAuth("", e.user, e.password, e.host)
	addr := fmt.Sprintf("%s:%s", e.host, e.port)

	err := smtp.SendMail(addr, auth, e.from, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("erro ao enviar email: %v", err)
	}

	return nil
}
