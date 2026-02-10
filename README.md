# harmonista

Uma rede de publicação de texto minimalista para quem gosta de ler e escrever. Sem analytics, sem trackers, sem cookies. Apenas texto, conteúdo e silêncio.

[harmonista.org](https://harmonista.org)

## Filosofia
- Sem distrações, sem coleta de dados
- Respeito à privacidade

## Como rodar
Pré‑requisitos: Go 1.22+ (para compilar) ou binário já construído, PostgreSQL, Nginx.

### Configuração Inicial

1. Copie o arquivo de exemplo de configuração:
   ```bash
   cp .env.example .env
   ```

2. Edite o arquivo `.env` com suas configurações:
   ```bash
   nano .env  # ou use seu editor preferido
   ```

### Desenvolvimento

Configure o `.env` para desenvolvimento:
```env
PG_HOST=localhost
PG_PORT=5432
PG_USER=harmonista
PG_PASSWORD=
PG_DBNAME=harmonista
SESSION_SECRET=LONG_LONG_SECRET
DOMAIN=http://localhost:8080
PORT=8080
```

Execute o servidor:
```bash
# Compilar e rodar
go build -o harmonista .
./harmonista

# Ou rodar diretamente sem compilar
go run main.go
```

A aplicação roda na porta 8080. Em produção, o Nginx faz proxy reverso na porta 80 e o Certbot configura o HTTPS.

### Produção (Nginx + Certbot)

1. Copie o `nginx.conf` para o Nginx:
   ```bash
   sudo cp nginx.conf /etc/nginx/sites-available/harmonista
   sudo ln -s /etc/nginx/sites-available/harmonista /etc/nginx/sites-enabled/
   sudo nginx -t && sudo systemctl reload nginx
   ```

2. Configure o HTTPS com Certbot:
   ```bash
   sudo certbot --nginx -d seudominio.com -d *.seudominio.com
   ```

## Estrutura (resumo)
- `admin/` — painel administrativo (views e handlers)
- `blog/` — frontend público (views e handlers)
- `database/` — migrações e SQL
- `models/` — definições de tabelas e modelos
- `public/` — assets estáticos (CSS)
- `main.go` — ponto de entrada da aplicação

## Licença
Este projeto é licenciado sob a WTFPL (Do What the F— You Want To Public License).

Mais detalhes: http://www.wtfpl.net/
