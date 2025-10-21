package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithValidEnv_LoadsCorrectly(t *testing.T) {
	// Define variáveis de ambiente via t.Setenv para isolar entre testes
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "0")
	t.Setenv("IP_RATE_LIMIT", "10")
	t.Setenv("IP_RATE_WINDOW", "1s")
	t.Setenv("IP_BLOCK_TIME", "5m")

	// Carrega config
	cfg, err := Load()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Valida todos os campos
	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "localhost", cfg.RedisHost)
	assert.Equal(t, 6379, cfg.RedisPort)
	assert.Equal(t, "", cfg.RedisPassword)
	assert.Equal(t, 0, cfg.RedisDB)
	assert.Equal(t, 10, cfg.IPLimit)
	assert.Equal(t, time.Second, cfg.IPWindow)
	assert.Equal(t, 5*time.Minute, cfg.IPBlockTime)
}

func TestLoad_WithMissingRequired_ReturnsError(t *testing.T) {
	// Não define SERVER_PORT - apenas define outras variáveis necessárias
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("IP_RATE_LIMIT", "10")
	t.Setenv("IP_RATE_WINDOW", "1s")
	t.Setenv("IP_BLOCK_TIME", "5m")

	// Garante que SERVER_PORT não está definido
	os.Unsetenv("SERVER_PORT")

	// Load deve retornar erro
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WithInvalidDuration_ReturnsError(t *testing.T) {
	// Define variáveis com duration inválida
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("IP_RATE_LIMIT", "10")
	t.Setenv("IP_RATE_WINDOW", "invalid") // Duration inválida
	t.Setenv("IP_BLOCK_TIME", "5m")

	// Load deve retornar erro
	cfg, err := Load()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetTokenConfig_NonExistingToken_ReturnsFalse(t *testing.T) {
	// Configura variáveis básicas
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("IP_RATE_LIMIT", "10")
	t.Setenv("IP_RATE_WINDOW", "1s")
	t.Setenv("IP_BLOCK_TIME", "5m")

	// Carrega config
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// GetTokenConfig("notfound") deve retornar (config, false)
	tokenConfig, exists := cfg.GetTokenConfig("notfound")

	// Assert
	assert.False(t, exists)
	assert.Zero(t, tokenConfig)
}
