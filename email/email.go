package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
		domain = "http://localhost/"
	}

	verificationLink := fmt.Sprintf("%s/confirmar/%s", domain, token)

	subject := "⌐◯ᵔ◯ Confirme sua conta - Harmonista"
	body := fmt.Sprintf(`Olá!

Obrigado por se cadastrar no ⌐◯ᵔ◯ Harmonista.

Para confirmar seu email e ativar sua conta, clique no link abaixo:

%s

Se você não se cadastrou na Harmonista, ignore este email.

---
Harmonista - o mínimo necessário`, verificationLink)

	// Tentar enviar via API HTTP do Mailtrap primeiro
	if err := e.sendMailViaAPI(to, subject, body); err != nil {
		log.Printf("Falha ao enviar via API, tentando SMTP: %v", err)
		// Fallback para SMTP
		message := fmt.Sprintf("From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s\r\n", e.from, to, subject, body)
		return e.sendMailWithTLS(to, message)
	}

	return nil
}

// sendMailViaAPI envia email usando a API HTTP do Mailtrap
func (e *EmailService) sendMailViaAPI(to, subject, textBody string) error {
	apiURL := "https://send.api.mailtrap.io/api/send"

	// Estrutura do payload JSON
	type EmailAddress struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}

	type EmailPayload struct {
		From     EmailAddress   `json:"from"`
		To       []EmailAddress `json:"to"`
		Subject  string         `json:"subject"`
		Text     string         `json:"text"`
		Category string         `json:"category"`
	}

	payload := EmailPayload{
		From: EmailAddress{
			Email: e.from,
			Name:  "Harmonista",
		},
		To: []EmailAddress{
			{Email: to},
		},
		Subject:  subject,
		Text:     textBody,
		Category: "Email Verification",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao criar JSON: %v", err)
	}

	log.Printf("Enviando email via API Mailtrap para %s", to)
	log.Printf("Payload JSON: %s", string(jsonData))

	// Verificar se temos token
	if e.password == "" {
		return fmt.Errorf("SMTP_PASSWORD (API token) não está configurado")
	}

	log.Printf("Token API: %s (tamanho: %d chars)", e.password, len(e.password))

	// Criar requisição HTTP
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Headers
	req.Header.Set("Authorization", "Bearer "+e.password)
	req.Header.Set("Content-Type", "application/json")
	log.Printf("Headers configurados: Content-Type=application/json, Authorization=Bearer [token]")

	// Enviar requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar requisição: %v", err)
	}
	defer resp.Body.Close()

	// Ler resposta
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler resposta: %v", err)
	}

	// Verificar status
	if resp.StatusCode != http.StatusOK {
		log.Printf("API Mailtrap retornou status %d: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	log.Printf("Email enviado com sucesso via API Mailtrap para %s", to)
	log.Printf("Resposta da API: %s", string(bodyBytes))
	return nil
}

// sendMailWithTLS envia email usando STARTTLS (porta 587) ou TLS direto (porta 465)
func (e *EmailService) sendMailWithTLS(to, message string) error {
	addr := fmt.Sprintf("%s:%s", e.host, e.port)

	log.Printf("Attempting to send email to %s via %s", to, addr)

	// Criar cliente SMTP
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("erro ao conectar ao servidor SMTP: %v", err)
	}
	defer client.Close()

	// Iniciar STARTTLS se disponível
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: e.host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("erro ao iniciar STARTTLS: %v", err)
		}
		log.Println("STARTTLS iniciado com sucesso")
	}

	// Autenticar
	log.Printf("Tentando autenticar como usuário: %s no host: %s", e.user, e.host)
	auth := smtp.PlainAuth("", e.user, e.password, e.host)
	if err = client.Auth(auth); err != nil {
		log.Printf("Falha na autenticação SMTP: %v", err)
		return fmt.Errorf("erro na autenticação SMTP: %v", err)
	}
	log.Println("Autenticação SMTP bem-sucedida")

	// Definir remetente
	log.Printf("Definindo remetente: %s", e.from)
	if err = client.Mail(e.from); err != nil {
		log.Printf("Erro ao definir remetente: %v", err)
		return fmt.Errorf("erro ao definir remetente: %v", err)
	}

	// Definir destinatário
	log.Printf("Definindo destinatário: %s", to)
	if err = client.Rcpt(to); err != nil {
		log.Printf("Erro ao definir destinatário: %v", err)
		return fmt.Errorf("erro ao definir destinatário: %v", err)
	}

	// Enviar mensagem
	log.Println("Abrindo canal de dados para envio da mensagem")
	w, err := client.Data()
	if err != nil {
		log.Printf("Erro ao abrir canal de dados: %v", err)
		return fmt.Errorf("erro ao abrir canal de dados: %v", err)
	}

	log.Println("Escrevendo mensagem")
	_, err = w.Write([]byte(message))
	if err != nil {
		log.Printf("Erro ao escrever mensagem: %v", err)
		return fmt.Errorf("erro ao escrever mensagem: %v", err)
	}

	log.Println("Fechando canal de dados")
	err = w.Close()
	if err != nil {
		log.Printf("Erro ao fechar canal de dados: %v", err)
		return fmt.Errorf("erro ao fechar canal de dados: %v", err)
	}

	// Finalizar
	log.Println("Finalizando conexão SMTP")
	if err = client.Quit(); err != nil {
		log.Printf("Erro ao finalizar conexão: %v", err)
		return fmt.Errorf("erro ao finalizar conexão: %v", err)
	}

	log.Printf("Email enviado com sucesso para %s", to)
	return nil
}
