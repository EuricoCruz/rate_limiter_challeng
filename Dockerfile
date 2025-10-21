# Stage 1: Builder
FROM golang:1.23-alpine AS builder

# Instala dependências de build
RUN apk add --no-cache git

WORKDIR /app

# Copia go.mod e go.sum primeiro (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copia código fonte
COPY . .

# Compila binário
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rate-limiter ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

# Instala ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copia binário do builder
COPY --from=builder /app/rate-limiter .

# Expõe porta
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Comando
CMD ["./rate-limiter"]
