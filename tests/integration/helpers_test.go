//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

// setupRedis conecta no Redis de teste e limpa todos os dados
func setupRedis(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380", // Porta do docker-compose.test.yml
		DB:   0,
	})

	ctx := context.Background()

	// Testa conexão
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Limpa TODOS os dados antes de cada teste
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush Redis: %v", err)
	}

	// Cleanup: fecha conexão após teste
	t.Cleanup(func() {
		client.Close()
	})

	return client
}
