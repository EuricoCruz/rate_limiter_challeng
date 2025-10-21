package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/usecase/check_rate_limit"
)

// Config interface para permitir mock em testes
type Config interface {
	GetIPLimit() int
	GetIPWindow() time.Duration
	GetIPBlockTime() time.Duration
	GetTokenConfig(token string) (TokenConfig, bool)
}

type TokenConfig struct {
	Limit     int
	Window    time.Duration
	BlockTime time.Duration
}

// UseCase interface para permitir mock em testes
type UseCase interface {
	Execute(ctx context.Context, input check_rate_limit.Input) (*check_rate_limit.Output, error)
}

type RateLimiterMiddleware struct {
	useCase UseCase
	config  Config
}

func NewRateLimiterMiddleware(useCase UseCase, config Config) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		useCase: useCase,
		config:  config,
	}
}

func (m *RateLimiterMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Extrai IP do request
		ip := extractIP(r)

		// 2. Extrai API_KEY do header
		apiKey := r.Header.Get("API_KEY")

		// 3. Determina qual configuração usar com prioridade Token > IP
		input := m.buildRateLimitInput(ip, apiKey)

		// Log da configuração utilizada
		keyType := "ip"
		if input.Key.Type == entity.KeyTypeToken {
			keyType = "token"
		}
		log.Printf("Rate limiter: using %s key '%s' with limit %d req/%v",
			keyType, input.Key.Value, input.Limit, input.Window)

		// 4. Executa use case
		output, err := m.useCase.Execute(ctx, input)
		if err != nil {
			// Log do erro interno
			log.Printf("Rate limiter error: %v for key %s", err, input.Key.Value)
			m.sendInternalServerError(w)
			return
		}

		// 5. Se não permitido, bloqueia com 429
		if !output.Allowed {
			log.Printf("Rate limit exceeded: %s for key %s (tokens: %.2f/%d)",
				output.Message, input.Key.Value, output.CurrentTokens, output.Limit)
			m.sendRateLimitExceeded(w, output.Message)
			return
		}

		// 6. Permitido - continua para próximo handler
		log.Printf("Rate limit OK: %s for key %s (tokens remaining: %.2f/%d)",
			"allowed", input.Key.Value, output.CurrentTokens, output.Limit)
		next.ServeHTTP(w, r)
	})
}

// buildRateLimitInput constrói o input baseado na prioridade Token > IP
func (m *RateLimiterMiddleware) buildRateLimitInput(ip, apiKey string) check_rate_limit.Input {
	// Prioridade: Token > IP
	// Se tem API_KEY, tenta usar configuração do token primeiro
	if apiKey != "" {
		if tokenConfig, exists := m.config.GetTokenConfig(apiKey); exists {
			// Usa configuração do token (prioridade alta)
			return check_rate_limit.Input{
				Key:       entity.NewTokenKey(apiKey),
				Limit:     tokenConfig.Limit,
				Window:    tokenConfig.Window,
				BlockTime: tokenConfig.BlockTime,
			}
		}
	}

	// Fallback: usa configuração do IP (prioridade baixa)
	return check_rate_limit.Input{
		Key:       entity.NewIPKey(ip),
		Limit:     m.config.GetIPLimit(),
		Window:    m.config.GetIPWindow(),
		BlockTime: m.config.GetIPBlockTime(),
	}
}

// sendInternalServerError envia resposta de erro interno 500
func (m *RateLimiterMiddleware) sendInternalServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	response := map[string]string{
		"error": "Internal Server Error",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Se o JSON encoding falhar, envia erro simples
		log.Printf("Failed to encode JSON error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// sendRateLimitExceeded envia resposta de rate limit exceeded 429
func (m *RateLimiterMiddleware) sendRateLimitExceeded(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]string{
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Se o JSON encoding falhar, envia erro simples
		log.Printf("Failed to encode JSON rate limit response: %v", err)
		http.Error(w, message, http.StatusTooManyRequests)
	}
}

// extractIP extrai o IP real do cliente considerando proxies
func extractIP(r *http.Request) string {
	// 1. Tenta X-Forwarded-For (proxy, load balancer)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// Pega o primeiro IP da lista (cliente original)
		ips := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// 2. Tenta X-Real-IP (nginx, cloudflare)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// 3. Usa RemoteAddr (conexão direta)
	// Remove porta: "192.168.1.1:12345" → "192.168.1.1"
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// RateLimiterMiddlewareFunc é uma função temporária para compatibilidade com testes
// Agora usa o método Handle() do struct RateLimiterMiddleware
func RateLimiterMiddlewareFunc(useCase UseCase, config Config) func(http.Handler) http.Handler {
	middleware := NewRateLimiterMiddleware(useCase, config)
	return middleware.Handle
}

// RateLimiterMiddlewareHandler é uma função temporária para compatibilidade com testes
// Será substituída pelo método Handle() do struct RateLimiterMiddleware
func RateLimiterMiddlewareHandler(useCase UseCase, config Config) func(http.Handler) http.Handler {
	return RateLimiterMiddlewareFunc(useCase, config)
}

// RateLimiterMiddlewareFunc é a função que os testes devem usar diretamente
// Para evitar conflito de nomes com o struct RateLimiterMiddleware
func RateLimiterMiddlewareHandlerWrapper(useCase UseCase, config Config) func(http.Handler) http.Handler {
	return RateLimiterMiddlewareFunc(useCase, config)
}

// RateLimiterMiddleware é o nome que os testes esperam - será criado no arquivo de teste
// Este nome está em conflito com o struct, então será resolvido nos testes
