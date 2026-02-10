# 🚀 Guia de Deploy - Harmonista

Este guia explica como configurar o deploy automático da aplicação Harmonista usando GitHub Actions.

## 📋 Pré-requisitos

### No Servidor

1. **Sistema Operacional**: Linux (Ubuntu/Debian recomendado)
2. **Usuário**: Acesso SSH configurado
3. **Permissões**: O usuário SSH deve ter permissões sudo
4. **Diretório de instalação**: `/var/www/harmonista`

### No GitHub

1. Repositório configurado
2. Secrets configurados (veja abaixo)

## 🔐 Configurar Secrets no GitHub

1. Acesse seu repositório no GitHub
2. Vá em **Settings** → **Secrets and variables** → **Actions**
3. Clique em **New repository secret**
4. Adicione os seguintes secrets:

| Secret Name | Descrição | Exemplo |
|------------|-----------|---------|
| `SSH_HOST` | Endereço do servidor | `seu-servidor.com` ou `192.168.1.100` |
| `SSH_USER` | Usuário SSH | `ubuntu` ou `seu-usuario` |
| `SSH_KEY` | Chave privada SSH | Conteúdo do arquivo `~/.ssh/id_ed25519` |

### Configurar chave SSH:

1. Gere um par de chaves (se ainda não tiver):
   ```bash
   ssh-keygen -t ed25519 -C "deploy@harmonista"
   ```

2. Copie a chave pública para o servidor:
   ```bash
   ssh-copy-id -i ~/.ssh/id_ed25519.pub seu-usuario@seu-servidor.com
   ```

3. No GitHub, adicione o conteúdo da chave **privada** como secret `SSH_KEY`:
   ```bash
   cat ~/.ssh/id_ed25519
   # Copie todo o conteúdo (incluindo BEGIN e END)
   ```

## 🎯 Como Funciona

### Deploy Automático

O deploy é executado automaticamente quando você faz push na branch `master`:

```bash
git add .
git commit -m "Suas alterações"
git push origin master
```

### Deploy Manual

Você também pode executar o deploy manualmente:

1. Acesse **Actions** no GitHub
2. Selecione o workflow **Deploy to Production**
3. Clique em **Run workflow**
4. Selecione a branch `master`
5. Clique em **Run workflow**

## 📝 Preparação Inicial do Servidor

Na primeira vez, você precisa preparar o servidor manualmente:

### 1. Conectar ao servidor

```bash
ssh seu-usuario@seu-servidor.com
```

### 2. Criar diretório da aplicação

```bash
sudo mkdir -p /var/www/harmonista
sudo chown -R seu-usuario:seu-usuario /var/www/harmonista
```

### 3. Criar arquivo .env no servidor

```bash
sudo nano /var/www/harmonista/.env
```

Cole o conteúdo do seu `.env` (use o `.env.example` como base):

```env
# Configuração do Banco de Dados (PostgreSQL)
PG_HOST=localhost
PG_PORT=5432
PG_USER=harmonista
PG_PASSWORD=sua-senha
PG_DBNAME=harmonista

# Segurança e Sessões
SESSION_KEY=sua-chave-longa-e-segura
SESSION_SECRET=seu-secret-longo-e-seguro

# Domínio
DOMAIN=https://harmonista.org

# Porta da aplicação (nginx faz proxy reverso)
PORT=8080

# Configuração de Email (SMTP)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASSWORD=your-password
SMTP_FROM=noreply@harmonista.org

# Porta
PORT=80
```

Salve com `Ctrl+O`, Enter, `Ctrl+X`

### 4. Ajustar permissões

```bash
sudo chown www-data:www-data /var/www/harmonista/.env
sudo chmod 600 /var/www/harmonista/.env
```

### 5. Criar usuário www-data (se não existir)

```bash
sudo useradd -r -s /bin/false www-data 2>/dev/null || true
```

## 🔧 Gerenciar o Serviço

Após o primeiro deploy, o serviço estará instalado. Use estes comandos:

### Ver status
```bash
sudo systemctl status harmonista
```

### Iniciar
```bash
sudo systemctl start harmonista
```

### Parar
```bash
sudo systemctl stop harmonista
```

### Reiniciar
```bash
sudo systemctl restart harmonista
```

### Ver logs em tempo real
```bash
sudo journalctl -u harmonista -f
```

### Ver logs das últimas 100 linhas
```bash
sudo journalctl -u harmonista -n 100
```

### Ativar no boot (já é feito automaticamente)
```bash
sudo systemctl enable harmonista
```

## 🔍 Verificar Deploy

Após o deploy, verifique se tudo está funcionando:

```bash
# 1. Verificar se o serviço está rodando
sudo systemctl status harmonista

# 2. Verificar logs
sudo journalctl -u harmonista -n 50

# 3. Testar se a aplicação responde
curl -I http://localhost
# ou
curl -I https://seu-dominio.com
```

## 🐛 Solução de Problemas

### Serviço não inicia (exit code 1)

O erro mais comum é falta de arquivos ou diretórios necessários:

```bash
# Ver logs detalhados
sudo journalctl -u harmonista -xe

# Verificar estrutura de diretórios
cd /var/www/harmonista
ls -la
find . -type d

# Verificar se todos os diretórios existem
ls -la admin/views/
ls -la site/views/
ls -la blog/views/
ls -la public/

# Verificar permissões
ls -la /var/www/harmonista/

# Testar binário manualmente para ver o erro
cd /var/www/harmonista
sudo -u www-data ./harmonista
```

**Estrutura mínima necessária:**
- `/var/www/harmonista/harmonista` (binário)
- `/var/www/harmonista/.env` (configuração)
- `/var/www/harmonista/admin/views/` (templates admin)
- `/var/www/harmonista/site/views/` (templates site)
- `/var/www/harmonista/blog/views/` (templates blog)
- `/var/www/harmonista/public/` (arquivos estáticos)

### Erro de permissão no .env

```bash
sudo chown www-data:www-data /var/www/harmonista/.env
sudo chmod 600 /var/www/harmonista/.env
```

### Porta já em uso

```bash
# Ver o que está usando a porta 80/443
sudo lsof -i :80
sudo lsof -i :443

# Parar outro serviço (ex: Apache)
sudo systemctl stop apache2
```

### SSL não funciona

Verifique se o Nginx está rodando e os certificados existem:
```bash
sudo systemctl status nginx
ls -la /etc/letsencrypt/live/harmonista.org/
```

Se não existirem, instale o certbot e configure com Nginx:
```bash
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx
sudo certbot --nginx -d harmonista.org -d *.harmonista.org
```

## 📂 Estrutura de Arquivos no Servidor

```
/var/www/harmonista/
├── harmonista           # Binário executável
├── .env                 # Configurações (criado manualmente)
├── analytics.db         # Banco de analytics SQLite (opcional)
├── public/              # Arquivos estáticos
├── admin/               # Views do admin
│   └── views/
├── site/                # Views do site
│   └── views/
└── blog/                # Views do blog
    └── views/
```

## 🔄 Workflow do GitHub Actions

O arquivo [.github/workflows/deploy.yml](.github/workflows/deploy.yml) contém o workflow de deploy.

### Etapas do Deploy:

1. ✅ Checkout do código
2. ✅ Configuração do Go
3. ✅ Build da aplicação
4. ✅ Preparação dos arquivos
5. ✅ Deploy para o servidor
6. ✅ Configuração do serviço systemd
7. ✅ Inicialização do serviço

### Ver histórico de deploys

1. Acesse **Actions** no GitHub
2. Veja o histórico de execuções
3. Clique em uma execução para ver detalhes e logs

## 📊 Monitoramento

### Ver uso de CPU e memória
```bash
top
# ou
htop
```

### Ver conexões ativas
```bash
sudo netstat -tulpn | grep harmonista
```

### Ver logs de acesso (se configurado)
```bash
sudo journalctl -u harmonista --since "1 hour ago"
```

## 🔒 Segurança

### Recomendações:

1. **Use SSH com chave pública** em vez de senha
2. **Configure firewall**:
   ```bash
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw allow 22/tcp
   sudo ufw enable
   ```
3. **Mantenha o sistema atualizado**:
   ```bash
   sudo apt-get update
   sudo apt-get upgrade
   ```
4. **Proteja o arquivo .env**:
   ```bash
   sudo chmod 600 /var/www/harmonista/.env
   ```

## 📞 Comandos Úteis

```bash
# Ver versão do Go no servidor
go version

# Ver espaço em disco
df -h

# Ver memória disponível
free -h

# Reiniciar servidor (use com cuidado!)
sudo reboot
```

## ✅ Checklist de Deploy

- [ ] Secrets configurados no GitHub
- [ ] Servidor preparado com diretório `/var/www/harmonista`
- [ ] Arquivo `.env` criado no servidor
- [ ] Usuário `www-data` existe
- [ ] Permissões configuradas corretamente
- [ ] Firewall configurado (portas 80 e 443)
- [ ] Certificados SSL instalados (se usar HTTPS)
- [ ] Primeiro deploy executado com sucesso
- [ ] Serviço rodando (`systemctl status harmonista`)
- [ ] Aplicação acessível via browser

---

**Dúvidas?** Consulte os logs com `sudo journalctl -u harmonista -f`
