# harmonista

Uma rede de publicação de texto minimalista para quem gosta de ler e escrever. Sem analytics, sem trackers, sem cookies. Apenas texto, conteúdo e silêncio.

harmonista.org

## Filosofia
- Sem distrações, sem coleta de dados
- Respeito à privacidade

## Como rodar
Pré‑requisitos: Go 1.22+ (para compilar) ou binário já construído.

- Usando o binário:
  ```bash
  ./harmonista
  ```

- Compilar e rodar:
  ```bash
  go build -o harmonista .
  ./harmonista
  ```

- Alternativa (sem construir binário):
  ```bash
  go run main.go
  ```

Por padrão, o servidor inicia na porta 8080

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
