# harmonista

Uma rede de publicação de texto minimalista para quem gosta de ler e escrever. Sem analytics, sem trackers, sem cookies. Apenas texto, conteúdo e silêncio.

[harmonista.org](https://harmonista.org)

## Filosofia
- Sem distrações, sem coleta de dados
- Respeito à privacidade

## Como rodar
Pré‑requisitos: Go 1.22+ (para compilar) ou binário já construído.

### Configuração Inicial

1. Copie o arquivo de exemplo de configuração:
   ```bash
   cp .env.example .env
   ```

2. Edite o arquivo `.env` com suas configurações:
   ```bash
   nano .env  # ou use seu editor preferido
   ```

### Modo Desenvolvimento (HTTP apenas)

Configure o `.env` para desenvolvimento:
```env
database=sqlite
sqlite_db=database.db
SESSION_SECRET=LONG_LONG_SECRET
DOMAIN=http://localhost
# Deixe SSL_CERT_PATH e SSL_KEY_PATH vazios ou comentados
```

Execute o servidor:
```bash
# Compilar e rodar
go build -o harmonista .
./harmonista

# Ou rodar diretamente sem compilar
go run main.go
```

Por padrão, o servidor inicia na porta 80 em modo HTTP (desenvolvimento).

### Modo Produção (HTTPS com Certbot)

O sistema suporta HTTPS automático com certificados do Let's Encrypt via Certbot.

#### 1. Instalar o Certbot

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install certbot

# Fedora/RHEL
sudo dnf install certbot

# Arch Linux
sudo pacman -S certbot
```

#### 2. Gerar certificados SSL

```bash
# Certifique-se de que nada está rodando nas portas 80 e 443
sudo certbot certonly --standalone -d seudominio.com -d www.seudominio.com
```

Os certificados serão gerados em:
- Certificado: `/etc/letsencrypt/live/seudominio.com/fullchain.pem`
- Chave privada: `/etc/letsencrypt/live/seudominio.com/privkey.pem`

#### 3. Configurar arquivo .env

Edite o arquivo `.env` com as configurações de produção:

```env
database=sqlite
sqlite_db=database.db
SESSION_SECRET=LONG_LONG_SECRET
DOMAIN=https://seudominio.com
SSL_CERT_PATH=/etc/letsencrypt/live/seudominio.com/fullchain.pem
SSL_KEY_PATH=/etc/letsencrypt/live/seudominio.com/privkey.pem
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
