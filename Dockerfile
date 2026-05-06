# --- Estágio 1: Builder ---
# Usamos a imagem oficial do Go para compilar o binário
FROM golang:1.22-alpine AS builder

# Instala git (necessário para baixar algumas dependências do Go)
RUN apk add --no-cache git

# Define o diretório de trabalho
WORKDIR /app

# Copia os arquivos de dependências primeiro (otimiza o cache do Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copia o restante do código fonte
COPY . .

# Compila o binário.
# CGO_ENABLED=0 garante que o binário seja estático (roda em qualquer Linux)
RUN CGO_ENABLED=0 GOOS=linux go build -o bot-gocase ./cmd/api/main.go

# --- Estágio 2: Final (Runtime) ---
# Usamos uma imagem Alpine limpa para o tamanho final ser pequeno
FROM alpine:latest

# Instala certificados CA (necessário para chamadas HTTPS à API do Gemini)
# Instala tzdata para que o bot use o fuso horário correto do Brasil
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copia apenas o binário compilado do estágio anterior
COPY --from=builder /app/bot-gocase .

# Expõe a porta que o Echo está ouvindo
EXPOSE 8080

# Comando para rodar a aplicação
CMD ["./bot-gocase"]