# ---------- STAGE 1: build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 👇 CORREÇÃO AQUI
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/api

# ---------- STAGE 2: runtime ----------
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/db/migration ./db/migration
COPY --from=builder /app/public ./public

EXPOSE 8080

CMD ["./app"]