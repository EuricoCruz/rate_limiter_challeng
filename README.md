# 🚦 Rate Limiter em Go - Middleware HTTP

Rate limiter robusto implementado em Go usando **Token Bucket Algorithm**, seguindo **Clean Architecture** e **princípios SOLID**. Pronto para ser usado como middleware em qualquer aplicação web Go.

## 📋 Índice

- [O que é?](#-o-que-é)
- [Como funciona?](#-como-funciona)
- [Instalação Rápida](#-instalação-rápida)
- [Usando como Middleware](#-usando-como-middleware)
- [Configuração](#️-configuração)
- [Testando a Aplicação](#-testando-a-aplicação)
- [Exemplos Práticos](#-exemplos-práticos)
- [API Reference](#-api-reference)
- [Troubleshooting](#-troubleshooting)

---

## 🎯 O que é?

Um **middleware HTTP injetável** que limita o número de requisições por:
- **Endereço IP**: Limite padrão para todos os clientes
- **Token de API**: Limites personalizados por cliente/token

### ✨ Features

- ✅ **Injetável**: Use em qualquer aplicação Go (Chi, Gin, Echo, net/http)
- ✅ **Token Bucket Algorithm**: Suaviza tráfego, permite bursts controlados
- ✅ **Prioridade**: Token sobrescreve limite de IP
- ✅ **Bloqueio Temporário**: Após exceder limite, bloqueia por tempo configurável
- ✅ **Redis**: Armazenamento rápido e distribuído
- ✅ **Atômico**: Lua scripts garantem operações sem race conditions
- ✅ **Testável**: Clean Architecture facilita testes
- ✅ **Produção Ready**: Docker, graceful shutdown, logs estruturados

---

## 🔧 Como funciona?

### Token Bucket Algorithm

```
┌─────────────────────────────────┐
│     Bucket (Balde)              │
│                                 │
│  ⚪⚪⚪⚪⚪⚪⚪⚪⚪⚪  (10 tokens)  │
│                                 │
│  Rate: 10 tokens/segundo        │
│  Refill: constante              │
└─────────────────────────────────┘
         ↓
    Requisição consome 1 token
         ↓
    Se tokens >= 1: ✅ Permite
    Se tokens < 1:  ❌ Bloqueia (429)
```

### Fluxo de Requisição

```
HTTP Request
    ↓
Middleware extrai IP/Token
    ↓
Use Case verifica rate limit
    ↓
Redis (Lua Script atômico)
    ↓
Permite (200) ou Bloqueia (429)
```

---

## 🚀 Instalação Rápida

### Pré-requisitos

- **Go 1.23+**
- **Docker & Docker Compose**
- **Make** (opcional, mas recomendado)

### Passos

```bash
# 1. Clone o repositório
git clone https://github.com/EuricoCruz/rate_limiter_challeng.git
cd rate_limiter_challeng

# 2. Configure as variáveis de ambiente
cp configs/.env.example .env
# Edite .env com suas configurações

# 3. Suba os containers
make docker-up

# 4. Teste se está funcionando
curl http://localhost:8080/health
# Resposta esperada: OK
```

**Pronto!** A aplicação está rodando em `http://localhost:8080`

---

## 🔌 Usando como Middleware

### ✅ Você pode usar este rate limiter em QUALQUER aplicação Go!

### Instalação como Dependência

```bash
# Adicione ao seu projeto
go get github.com/EuricoCruz/rate_limiter_challeng
```

---

## 📝 Integração em Sua Aplicação

### Exemplo 1: Chi Router (Framework usado no projeto)

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/EuricoCruz/rate_limiter_challeng/internal/adapter/http/middleware"
    redisAdapter "github.com/EuricoCruz/rate_limiter_challeng/internal/adapter/storage/redis"
    "github.com/EuricoCruz/rate_limiter_challeng/internal/infrastructure/config"
    infraRedis "github.com/EuricoCruz/rate_limiter_challeng/internal/infrastructure/redis"
    "github.com/EuricoCruz/rate_limiter_challeng/internal/usecase/check_rate_limit"
)

// configAdapter adapta config.Config para implementar middleware.Config
type configAdapter struct {
    *config.Config
}

func (c *configAdapter) GetTokenConfig(token string) (middleware.TokenConfig, bool) {
    cfg, exists := c.Config.GetTokenConfig(token)
    if !exists {
        return middleware.TokenConfig{}, false
    }
    return middleware.TokenConfig{
        Limit:     cfg.Limit,
        Window:    cfg.Window,
        BlockTime: cfg.BlockTime,
    }, true
}

func main() {
    // 1. Carrega configuração
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // 2. Conecta Redis
    redisClient, err := infraRedis.NewClient(cfg)
    if err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }
    defer redisClient.Close()
    
    // 3. Monta as camadas
    storage := redisAdapter.NewRedisStorage(redisClient)
    useCase := check_rate_limit.NewUseCase(storage)
    
    // 4. Cria adapter para configurar interface do middleware
    cfgAdapter := &configAdapter{Config: cfg}
    rateLimiter := middleware.NewRateLimiterMiddleware(useCase, cfgAdapter)
    
    // 5. Cria seu router
    r := chi.NewRouter()
    
    // 6. ✅ INJETA O MIDDLEWARE (TODAS AS ROTAS)
    r.Use(rateLimiter.Handle)
    
    // 7. Define suas rotas (todas terão rate limiting)
    r.Get("/api/users", getUsersHandler)
    r.Post("/api/orders", createOrderHandler)
    
    // 8. Inicia servidor
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

---

### Exemplo 2: Aplicar Apenas em Rotas Específicas

```go
func main() {
    r := chi.NewRouter()
    
    // ✅ Rate limiting APENAS nas rotas /api/*
    r.Route("/api", func(r chi.Router) {
        r.Use(rateLimiter.Handle)  // Aplica apenas neste grupo
        
        r.Get("/users", getUsersHandler)
        r.Post("/orders", createOrderHandler)
    })
    
    // ❌ Estas rotas NÃO têm rate limiting
    r.Get("/health", healthHandler)
    r.Get("/metrics", metricsHandler)
    
    http.ListenAndServe(":8080", r)
}
```

---

### Exemplo 3: Rate Limiting por Endpoint

```go
func main() {
    r := chi.NewRouter()
    
    // ✅ Apenas endpoint sensível tem rate limiting
    r.With(rateLimiter.Handle).Post("/api/payment", paymentHandler)
    
    // ❌ Outros endpoints sem rate limiting
    r.Get("/api/products", getProductsHandler)
    r.Get("/health", healthHandler)
    
    http.ListenAndServe(":8080", r)
}
```

---

### Exemplo 4: Múltiplos Rate Limiters

```go
func main() {
    // Rate limiter RESTRITIVO para API pública
    publicLimiter := middleware.NewRateLimiterMiddleware(
        publicUseCase,
        &config.Config{
            IPLimit:   10,              // 10 requisições
            IPWindow:  time.Second,     // por segundo
            IPBlockTime: 5 * time.Minute, // bloqueia 5 minutos
        },
    )
    
    // Rate limiter PERMISSIVO para API interna
    internalLimiter := middleware.NewRateLimiterMiddleware(
        internalUseCase,
        &config.Config{
            IPLimit:   1000,             // 1000 requisições
            IPWindow:  time.Second,      // por segundo
            IPBlockTime: time.Minute,    // bloqueia 1 minuto
        },
    )
    
    r := chi.NewRouter()
    
    // API pública (limite rígido)
    r.Route("/api/public", func(r chi.Router) {
        r.Use(publicLimiter.Handle)
        r.Get("/data", publicDataHandler)
    })
    
    // API interna (limite flexível)
    r.Route("/api/internal", func(r chi.Router) {
        r.Use(internalLimiter.Handle)
        r.Get("/admin", adminHandler)
    })
    
    http.ListenAndServe(":8080", r)
}
```

---

### Exemplo 5: Compatibilidade com Outros Frameworks

#### 🔷 Gin Framework

```go
import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()
    
    // Adapter para Gin
    r.Use(func(c *gin.Context) {
        handler := rateLimiter.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
            c.Next()
        }))
        handler.ServeHTTP(c.Writer, c.Request)
    })
    
    r.GET("/api/users", getUsersHandler)
    r.Run(":8080")
}
```

#### 🔷 Echo Framework

```go
import "github.com/labstack/echo/v4"

func main() {
    e := echo.New()
    
    // Adapter para Echo
    e.Use(echo.WrapMiddleware(rateLimiter.Handle))
    
    e.GET("/api/users", getUsersHandler)
    e.Start(":8080")
}
```

#### 🔷 HTTP Padrão (net/http)

```go
import "net/http"

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/users", getUsersHandler)
    
    // Wrap com rate limiter
    handler := rateLimiter.Handle(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

---

## ⚙️ Configuração

### Arquivo .env

```bash
# Servidor
SERVER_PORT=8080

# Redis (obrigatório)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate Limiting por IP (obrigatório)
IP_RATE_LIMIT=10           # Máximo de requisições
IP_RATE_WINDOW=1s          # Janela de tempo (1s, 1m, 1h)
IP_BLOCK_TIME=5m           # Tempo de bloqueio após exceder

# Tokens de API (opcional - quantos quiser)
# Formato: TOKEN_{nome}={valor_do_token}
#          TOKEN_{nome}_LIMIT=100
#          TOKEN_{nome}_WINDOW=1s
#          TOKEN_{nome}_BLOCK_TIME=10m

TOKEN_cliente1=abc123
TOKEN_cliente1_LIMIT=100
TOKEN_cliente1_WINDOW=1s
TOKEN_cliente1_BLOCK_TIME=10m

TOKEN_cliente2=xyz789
TOKEN_cliente2_LIMIT=50
TOKEN_cliente2_WINDOW=1s
TOKEN_cliente2_BLOCK_TIME=5m
```

### Variáveis Obrigatórias

| Variável | Descrição | Exemplo |
|----------|-----------|---------|
| `SERVER_PORT` | Porta do servidor | `8080` |
| `REDIS_HOST` | Host do Redis | `localhost` ou `redis` (Docker) |
| `REDIS_PORT` | Porta do Redis | `6379` |
| `IP_RATE_LIMIT` | Limite de req/janela por IP | `10` |
| `IP_RATE_WINDOW` | Janela de tempo | `1s`, `1m`, `1h` |
| `IP_BLOCK_TIME` | Tempo de bloqueio | `5m`, `1h` |

### Configurando Tokens

Para cada token de API:

```bash
# 1. Defina o token
TOKEN_meucliente=token_secreto_aqui

# 2. Configure o limite
TOKEN_meucliente_LIMIT=100

# 3. Configure a janela
TOKEN_meucliente_WINDOW=1s

# 4. Configure o bloqueio
TOKEN_meucliente_BLOCK_TIME=10m
```

---

## 🧪 Testando a Aplicação

### Teste 1: Health Check

```bash
curl http://localhost:8080/health
# Resposta esperada: OK
```

### Teste 2: Rate Limiting por IP

```bash
# Faz 15 requisições rápidas (limite é 10)
for i in {1..15}; do 
    echo "Request $i:"
    curl -w "\nStatus: %{http_code}\n" http://localhost:8080/
    echo "---"
done

# Resultado esperado:
# Requisições 1-10: Status 200 ✅
# Requisições 11-15: Status 429 ❌
```

### Teste 3: Rate Limiting com Token

```bash
# Com token válido (limite maior)
for i in {1..20}; do 
    echo "Request $i com token:"
    curl -w "\nStatus: %{http_code}\n" \
         -H "API_KEY: abc123" \
         http://localhost:8080/
    echo "---"
done

# Resultado esperado:
# Requisições 1-100: Status 200 ✅ (limite do token é 100)
# Requisições 101+: Status 429 ❌
```

### Teste 4: Token Sobrescreve IP

```bash
# 1. Esgota limite por IP
for i in {1..11}; do curl http://localhost:8080/ > /dev/null 2>&1; done

# 2. Próxima requisição SEM token (bloqueada)
curl -w "\nStatus: %{http_code}\n" http://localhost:8080/
# Status: 429 ❌

# 3. Requisição COM token (passa!)
curl -w "\nStatus: %{http_code}\n" \
     -H "API_KEY: abc123" \
     http://localhost:8080/
# Status: 200 ✅ (token sobrescreve limite de IP)
```

### Teste 5: Bloqueio Temporário

```bash
# 1. Esgota limite
for i in {1..11}; do curl http://localhost:8080/ > /dev/null 2>&1; done

# 2. Tenta novamente (ainda bloqueado)
curl -w "\nStatus: %{http_code}\n" http://localhost:8080/
# Status: 429 ❌

# 3. Aguarda 5 minutos (IP_BLOCK_TIME) e tenta novamente
sleep 300
curl -w "\nStatus: %{http_code}\n" http://localhost:8080/
# Status: 200 ✅ (bloqueio expirou)
```

### Teste 6: Verificar Redis

```bash
# Conecta no Redis
docker-compose exec rate-limiter-redis redis-cli

# Lista todas as chaves de rate limiting
127.0.0.1:6379> KEYS rate_limit:*

# Exemplo de saída:
# 1) "rate_limit:ip:192.168.1.1:tokens"
# 2) "rate_limit:ip:192.168.1.1:last_refill"
# 3) "rate_limit:ip:192.168.1.1:blocked"
# 4) "rate_limit:token:abc123:tokens"

# Verifica tokens restantes de um IP
127.0.0.1:6379> GET rate_limit:ip:192.168.1.1:tokens
# Exemplo: "7.5" (ainda tem 7.5 tokens)

# Verifica se está bloqueado
127.0.0.1:6379> EXISTS rate_limit:ip:192.168.1.1:blocked
# 0 = não bloqueado, 1 = bloqueado

# Limpa TODOS os dados (útil para testes)
127.0.0.1:6379> FLUSHDB
```

---

## 📊 Testando em Produção

### Load Test com k6

O load test verifica a performance do rate limiter sob carga real e confirma que o bloqueio está funcionando corretamente.

#### Instalação do k6

```bash
# macOS: brew install k6
# Linux: sudo apt install k6
# Windows: choco install k6
```

#### Executando o Teste

```bash
# 1. Certifique-se que a aplicação está rodando
make docker-up

# 2. Rode o load test
make load-test
# ou: k6 run tests/load/load_test.js
```

#### O que o Teste Faz

- **100 usuários concorrentes** por 1 minuto (ramp up/down incluído)
- **90% requests normais** (testam rate limiting por IP)  
- **10% requests com token** (testam rate limiting por token)
- **Mede métricas** de performance e bloqueios

#### Resultado Esperado

```
✓ http_req_duration p(95)<500ms    (95% das requests < 500ms)
✓ blocked_requests rate>0.3        (>30% das requests bloqueadas)
✓ throughput                       (~10k-50k req/s)
```

**Interpretação:**
- ✅ **p95 < 500ms**: Performance adequada
- ✅ **blocked_requests > 30%**: Rate limiter está funcionando (bloqueando requests)
- ✅ **throughput alto**: Sistema não é um gargalo

#### Exemplo de Saída

```
running (1m30.0s), 000/100 VUs, 8553 complete and 0 interrupted iterations
default ✓ [=====================================] 100 VUs  1m0s

     ✓ status is 200 or 429
     ✓ response time < 500ms  
     ✓ blocked_requests rate>0.3

     blocked_requests: 68.57% (rate)
     http_req_duration: avg=45ms min=2ms med=25ms max=485ms p(95)=124ms
     http_req_rate: 95.03 req/s
```

### Teste de Stress

```bash
# Apache Bench (alternativa ao k6)
ab -n 10000 -c 100 http://localhost:8080/

# Resultado esperado:
# - Requests per second: > 10000
# - Non-2xx responses: ~40-50% (bloqueadas por rate limit)
```

---

## 📡 API Reference

### Endpoints

| Método | Path | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Health check (não tem rate limiting) |
| `GET` | `/` | Endpoint de exemplo (rate limited) |
| `*` | `*` | Qualquer rota sua (se middleware aplicado) |

### Headers de Request

| Header | Obrigatório | Descrição |
|--------|-------------|-----------|
| `API_KEY` | Não | Token de API (sobrescreve limite de IP) |

### Response Codes

| Code | Descrição | Body |
|------|-----------|------|
| `200` | Requisição permitida | Seu conteúdo |
| `429` | Rate limit excedido | `{"message": "you have reached the maximum..."}` |
| `500` | Erro interno | `Internal Server Error` |

### Exemplo de Response 429

```json
{
  "message": "you have reached the maximum number of requests or actions allowed within a certain time frame"
}
```

---

## 🐛 Troubleshooting

### Problema 1: Rate limiter não bloqueia

**Sintoma:** Todas as requisições passam, mesmo após limite

**Diagnóstico:**
```bash
# 1. Verifica se Redis está rodando
docker-compose ps rate-limiter-redis
# Status deve ser "Up (healthy)"

# 2. Verifica configuração
cat .env | grep RATE_LIMIT
# IP_RATE_LIMIT deve ser > 0

# 3. Verifica logs da aplicação
docker-compose logs app | grep "rate limit"

# 4. Verifica se middleware está aplicado
# No seu código, certifique-se que r.Use(rateLimiter.Handle) está presente
```

**Solução:**
- Certifique-se que `IP_RATE_LIMIT > 0`
- Verifique se Redis está acessível
- Confirme que middleware foi injetado nas rotas

---

### Problema 2: Todas requisições retornam 429

**Sintoma:** Primeira requisição já retorna 429

**Diagnóstico:**
```bash
# 1. Verifica se está bloqueado no Redis
docker-compose exec rate-limiter-redis redis-cli
127.0.0.1:6379> KEYS *:blocked
# Se retornar chaves, está bloqueado

# 2. Limpa bloqueios
127.0.0.1:6379> FLUSHDB
# Tenta novamente

# 3. Verifica se IP_RATE_LIMIT não está muito baixo
cat .env | grep IP_RATE_LIMIT
# Deve ser >= 1
```

**Solução:**
- Limpe Redis: `docker-compose exec rate-limiter-redis redis-cli FLUSHDB`
- Aumente `IP_RATE_LIMIT` no .env
- Reduza `IP_BLOCK_TIME` para testes

---

### Problema 3: Token não funciona

**Sintoma:** Header `API_KEY` não sobrescreve limite de IP

**Diagnóstico:**
```bash
# 1. Verifica se token está configurado
cat .env | grep TOKEN_abc123
# Deve ter TOKEN_abc123=abc123 e _LIMIT, _WINDOW, _BLOCK_TIME

# 2. Testa requisição com token
curl -v -H "API_KEY: abc123" http://localhost:8080/
# Verifica se header está sendo enviado

# 3. Verifica logs
docker-compose logs app | grep "token"
```

**Solução:**
- Certifique-se que formato no .env está correto
- Token deve estar em MAIÚSCULAS: `TOKEN_abc123` não `token_abc123`
- Header deve ser exatamente `API_KEY` (case-sensitive)

---

### Problema 4: Redis connection refused

**Sintoma:** Erro "dial tcp: connection refused"

**Diagnóstico:**
```bash
# 1. Verifica se Redis está rodando
docker-compose ps rate-limiter-redis

# 2. Verifica REDIS_HOST no .env
cat .env | grep REDIS_HOST
# Deve ser "redis" (Docker) ou "localhost" (local)
```

**Solução:**
- Se rodando com Docker: `REDIS_HOST=redis`
- Se rodando local: `REDIS_HOST=localhost`
- Certifique-se que Redis está na mesma rede Docker

---

### Problema 5: Performance lenta

**Sintoma:** Requisições demorando >100ms

**Diagnóstico:**
```bash
# 1. Verifica latência do Redis
docker-compose exec rate-limiter-redis redis-cli --latency
# Deve ser < 1ms

# 2. Verifica se Lua script está otimizado
# (já está otimizado por padrão)

# 3. Verifica load do Redis
docker stats
```

**Solução:**
- Use Redis local ao invés de remoto
- Considere Redis Cluster para alta carga
- Aumente `PoolSize` no Redis client (padrão: 10)

---

## 🔧 Comandos Úteis

### Docker

```bash
# Subir containers
make docker-up
# ou: docker-compose up -d

# Parar containers
make docker-down
# ou: docker-compose down -v

# Ver logs
make logs
# ou: docker-compose logs -f app

# Rebuild
docker-compose build --no-cache
docker-compose up -d
```

### Redis

```bash
# Conectar no Redis CLI
make redis-cli
# ou: docker-compose exec rate-limiter-redis redis-cli

# Limpar todos os dados
redis-cli FLUSHDB

# Ver todas as chaves
redis-cli KEYS *

# Monitorar comandos em tempo real
redis-cli MONITOR
```

### Testes

```bash
# Testes unitários
make test-unit

# Testes de integração
make test-integration

# Todos os testes
make test

# Coverage
make coverage

# Load test (requer k6 instalado)
make load-test
# ou: k6 run tests/load/load_test.js
```

---

## 📞 Suporte

Se tiver problemas:

1. **Verifique os logs:** `make logs`
2. **Limpe o Redis:** `docker-compose exec rate-limiter-redis redis-cli FLUSHDB`
3. **Recrie os containers:** `docker-compose down -v && docker-compose up -d`
4. **Verifique o .env:** Todas as variáveis obrigatórias estão configuradas?

---

## 📄 Licença

MIT

---

## 🎯 Checklist de Testes para Avaliação

Para quem vai avaliar o projeto, siga este checklist:

- [ ] 1. Clone e suba: `git clone && make docker-up`
- [ ] 2. Health check: `curl http://localhost:8080/health` (deve retornar `OK`)
- [ ] 3. Rate limit IP: 11 requisições rápidas (11ª deve retornar 429)
- [ ] 4. Rate limit Token: requisição com `API_KEY: abc123` (deve passar mesmo após bloquear IP)
- [ ] 5. Bloqueio persiste: após bloquear, aguardar 30s e tentar novamente (ainda bloqueado)
- [ ] 6. Redis: `make redis-cli` e `KEYS rate_limit:*` (deve mostrar chaves)
- [ ] 7. Config: token configurado tem limite diferente de IP
- [ ] 8. Middleware: código mostra injeção clara do middleware
- [ ] 9. Testes: `make test` (todos devem passar)
- [ ] 10. Load test: `make load-test` (p95 < 500ms, >30% blocked)