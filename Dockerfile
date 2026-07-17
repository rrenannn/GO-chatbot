# ---------- STAGE 1: frontend build ----------
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# ---------- STAGE 2: backend build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/api

# ---------- STAGE 3: runtime ----------
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/db/migration ./db/migration
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

EXPOSE 8080

CMD ["./app"]