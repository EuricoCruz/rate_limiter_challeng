package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/repository"
)

// RedisStorage implementa a interface repository.Storage usando Redis como backend
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage cria uma nova instância de RedisStorage usando dependency injection
func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

// Close fecha a conexão com o Redis
func (r *RedisStorage) Close() error {
	return r.client.Close()
}

// CheckAndConsume implementa o método da interface Storage
// Executa o algoritmo Token Bucket usando script Lua para operação atômica
func (r *RedisStorage) CheckAndConsume(
	ctx context.Context,
	key entity.LimiterKey,
	limit int,
	window time.Duration,
) (*repository.CheckResult, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got: %d", limit)
	}
	if window <= 0 {
		return nil, fmt.Errorf("window must be positive, got: %v", window)
	}

	now := time.Now().Unix()
	keyStr := key.String()

	// Chaves para tokens e timestamp
	tokensKey, lastRefillKey := r.generateTokenKeys(keyStr)

	// Executa Lua script atomicamente
	result, err := r.executeTokenBucketScript(ctx, tokensKey, lastRefillKey, limit, window, now)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token bucket script for key %s: %w", keyStr, err)
	}

	// Parseia resultado do Lua: {allowed, tokens, capacity}
	allowed, tokens, err := r.parseScriptResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script result for key %s: %w", keyStr, err)
	}

	return &repository.CheckResult{
		Allowed:       allowed,
		CurrentTokens: tokens,
		Limit:         limit,
	}, nil
}

// generateTokenKeys gera as chaves Redis necessárias para o algoritmo Token Bucket
func (r *RedisStorage) generateTokenKeys(keyStr string) (tokensKey, lastRefillKey string) {
	return keyStr + ":tokens", keyStr + ":last_refill"
}

// executeTokenBucketScript executa o script Lua do Token Bucket
func (r *RedisStorage) executeTokenBucketScript(
	ctx context.Context,
	tokensKey, lastRefillKey string,
	limit int,
	window time.Duration,
	now int64,
) (interface{}, error) {
	result, err := tokenBucketScript.Run(
		ctx,
		r.client,
		[]string{tokensKey, lastRefillKey}, // KEYS
		limit, window.Seconds(), now,       // ARGV
	).Result()

	if err != nil {
		return nil, fmt.Errorf("redis script execution failed: %w", err)
	}

	return result, nil
}

// parseScriptResult parseia o resultado retornado pelo script Lua
// Espera formato: [allowed (int64), currentTokens (number), capacity (int64)]
func (r *RedisStorage) parseScriptResult(result interface{}) (allowed bool, tokens float64, err error) {
	resultSlice, ok := result.([]interface{})
	if !ok {
		return false, 0, fmt.Errorf("expected array result, got: %T", result)
	}

	if len(resultSlice) != 3 {
		return false, 0, fmt.Errorf("expected 3 elements in result array, got: %d", len(resultSlice))
	}

	// Parse allowed flag
	allowedValue, ok := resultSlice[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("expected int64 for allowed flag, got: %T", resultSlice[0])
	}
	allowed = allowedValue == 1

	// Parse current tokens
	tokens, err = r.parseTokensValue(resultSlice[1])
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse tokens value: %w", err)
	}

	return allowed, tokens, nil
}

// parseTokensValue parseia o valor de tokens que pode vir em diferentes tipos do Lua
func (r *RedisStorage) parseTokensValue(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case string:
		tokens, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse string tokens value '%s': %w", v, err)
		}
		return tokens, nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected tokens value type %T with value %v", v, v)
	}
}

// SetBlock implementa o método da interface Storage
// Bloqueia uma chave por um período específico usando TTL do Redis
func (r *RedisStorage) SetBlock(ctx context.Context, key entity.LimiterKey, blockTime time.Duration) error {
	if blockTime <= 0 {
		return fmt.Errorf("block time must be positive, got: %v", blockTime)
	}

	blockKey := r.generateBlockKey(key)

	err := r.client.Set(ctx, blockKey, "1", blockTime).Err()
	if err != nil {
		return fmt.Errorf("failed to set block for key %s: %w", key.String(), err)
	}

	return nil
}

// IsBlocked implementa o método da interface Storage
// Verifica se uma chave está bloqueada consultando o Redis
func (r *RedisStorage) IsBlocked(ctx context.Context, key entity.LimiterKey) (bool, error) {
	blockKey := r.generateBlockKey(key)

	result, err := r.client.Exists(ctx, blockKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check block status for key %s: %w", key.String(), err)
	}

	return result > 0, nil
}

// generateBlockKey gera a chave Redis para bloqueio
func (r *RedisStorage) generateBlockKey(key entity.LimiterKey) string {
	return key.String() + ":blocked"
}
