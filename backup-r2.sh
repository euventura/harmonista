#!/bin/bash

# ========================================
# Script de Backup dos Bancos de Dados para Cloudflare R2
# ========================================
# Este script faz backup dos bancos SQLite e envia para o R2
# Configuração: Execute 2x por dia via cron
#
# Para configurar o cron, adicione as linhas abaixo no crontab:
# crontab -e
#
# Executar às 6h e 18h todos os dias:
# 0 6,18 * * * /home/euventura/Dev/harmonista/backup-r2.sh >> /home/euventura/Dev/harmonista/backup.log 2>&1
#
# Ou se preferir horários diferentes:
# 0 3 * * * /home/euventura/Dev/harmonista/backup-r2.sh >> /home/euventura/Dev/harmonista/backup.log 2>&1  # 3h da manhã
# 0 15 * * * /home/euventura/Dev/harmonista/backup-r2.sh >> /home/euventura/Dev/harmonista/backup.log 2>&1  # 15h da tarde
# ========================================

set -e  # Parar em caso de erro

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Diretório do script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Carregar variáveis do .env
if [ ! -f .env ]; then
    echo -e "${RED}[ERRO] Arquivo .env não encontrado!${NC}"
    exit 1
fi

# Exportar variáveis do .env
set -a
source .env
set +a

# Validar variáveis obrigatórias
if [ -z "$R2_ENDPOINT" ] || [ -z "$R2_ACCESS_KEY" ] || [ -z "$R2_SECRET_KEY" ] || [ -z "$R2_BUCKET" ]; then
    echo -e "${RED}[ERRO] Variáveis R2 não configuradas no .env!${NC}"
    echo "Por favor, adicione no .env:"
    echo "R2_ENDPOINT=https://ACCOUNT_ID.r2.cloudflarestorage.com"
    echo "R2_ACCESS_KEY=seu_access_key"
    echo "R2_SECRET_KEY=seu_secret_key"
    echo "R2_BUCKET=nome_do_bucket"
    exit 1
fi

# Exportar credenciais para o AWS CLI
export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_KEY"

# Verificar se aws-cli está instalado
if ! command -v aws &> /dev/null; then
    echo -e "${RED}[ERRO] aws-cli não está instalado!${NC}"
    echo "Instale com: sudo pacman -S aws-cli"
    echo "Ou: pip install awscli"
    exit 1
fi

# Timestamp para os arquivos
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DATE=$(date +"%Y-%m-%d %H:%M:%S")

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Iniciando backup - $DATE${NC}"
echo -e "${GREEN}========================================${NC}"

# Diretório temporário para backups
BACKUP_DIR="/tmp/harmonista_backup_$TIMESTAMP"
mkdir -p "$BACKUP_DIR"

# Lista de bancos de dados para backup
DATABASES=("$sqlite_db" "$analytics_db")

# Fazer backup de cada banco
for DB in "${DATABASES[@]}"; do
    if [ -f "$DB" ]; then
        DB_NAME=$(basename "$DB")
        BACKUP_FILE="$BACKUP_DIR/${DB_NAME%.db}_$TIMESTAMP.db"

        echo -e "${YELLOW}[INFO] Fazendo backup de $DB_NAME...${NC}"

        # Copiar arquivo do banco
        cp "$DB" "$BACKUP_FILE"

        # Comprimir o backup
        gzip "$BACKUP_FILE"
        BACKUP_FILE="${BACKUP_FILE}.gz"

        # Upload para R2
        echo -e "${YELLOW}[INFO] Enviando para R2...${NC}"

        aws s3 cp "$BACKUP_FILE" \
            "s3://${R2_BUCKET}/backups/$(basename $BACKUP_FILE)" \
            --endpoint-url "$R2_ENDPOINT" \
            --region auto \
            2>&1

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}[OK] $DB_NAME enviado com sucesso!${NC}"
        else
            echo -e "${RED}[ERRO] Falha ao enviar $DB_NAME${NC}"
        fi

        # Remover arquivo local
        rm -f "$BACKUP_FILE"
    else
        echo -e "${YELLOW}[AVISO] Banco $DB não encontrado, pulando...${NC}"
    fi
done

# Limpar diretório temporário
rm -rf "$BACKUP_DIR"

# Listar backups recentes no R2 (opcional)
echo ""
echo -e "${YELLOW}[INFO] Listando últimos 5 backups no R2:${NC}"
aws s3 ls "s3://${R2_BUCKET}/backups/" \
    --endpoint-url "$R2_ENDPOINT" \
    --region auto \
    2>&1 | tail -n 5

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Backup concluído!${NC}"
echo -e "${GREEN}========================================${NC}"

# Opcional: Limpar backups antigos (manter apenas últimos 30 dias)
echo ""
echo -e "${YELLOW}[INFO] Removendo backups com mais de 30 dias...${NC}"

# Listar todos os backups
aws s3 ls "s3://${R2_BUCKET}/backups/" \
    --endpoint-url "$R2_ENDPOINT" \
    --region auto \
    2>&1 | while read -r line; do

    # Extrair data e nome do arquivo
    FILE_DATE=$(echo "$line" | awk '{print $1}')
    FILE_NAME=$(echo "$line" | awk '{print $4}')

    # Calcular diferença em dias
    if [ ! -z "$FILE_DATE" ] && [ ! -z "$FILE_NAME" ]; then
        FILE_TIMESTAMP=$(date -d "$FILE_DATE" +%s 2>/dev/null || echo "0")
        CURRENT_TIMESTAMP=$(date +%s)
        DAYS_DIFF=$(( ($CURRENT_TIMESTAMP - $FILE_TIMESTAMP) / 86400 ))

        # Remover se tiver mais de 30 dias
        if [ $DAYS_DIFF -gt 30 ]; then
            echo -e "${YELLOW}[INFO] Removendo backup antigo: $FILE_NAME${NC}"
            aws s3 rm "s3://${R2_BUCKET}/backups/$FILE_NAME" \
                --endpoint-url "$R2_ENDPOINT" \
                --region auto \
                2>&1
        fi
    fi
done

echo -e "${GREEN}[OK] Limpeza concluída!${NC}"
