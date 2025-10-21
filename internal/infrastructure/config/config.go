package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerPort int

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// IP Rate Limiting
	IPLimit     int
	IPWindow    time.Duration
	IPBlockTime time.Duration

	// Token Configs (mapa token → configuração)
	TokenConfigs map[string]TokenConfig
}

type TokenConfig struct {
	Limit     int
	Window    time.Duration
	BlockTime time.Duration
}

// GetIPLimit implementa interface do middleware
func (c *Config) GetIPLimit() int {
	return c.IPLimit
}

func (c *Config) GetIPWindow() time.Duration {
	return c.IPWindow
}

func (c *Config) GetIPBlockTime() time.Duration {
	return c.IPBlockTime
}

func (c *Config) GetTokenConfig(token string) (TokenConfig, bool) {
	cfg, exists := c.TokenConfigs[token]
	return cfg, exists
}

func Load() (*Config, error) {
	// Limpa configurações anteriores do viper
	viper.Reset()

	// Configura viper
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	// Tenta ler .env (ignora erro se não existir, usa env vars)
	_ = viper.ReadInConfig()

	// Carrega configurações básicas
	cfg := &Config{
		ServerPort:    viper.GetInt("SERVER_PORT"),
		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetInt("REDIS_PORT"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),
		RedisDB:       viper.GetInt("REDIS_DB"),
		IPLimit:       viper.GetInt("IP_RATE_LIMIT"),
		IPWindow:      viper.GetDuration("IP_RATE_WINDOW"),
		IPBlockTime:   viper.GetDuration("IP_BLOCK_TIME"),
		TokenConfigs:  make(map[string]TokenConfig),
	}

	// Valida campos obrigatórios
	if cfg.ServerPort <= 0 {
		return nil, fmt.Errorf("SERVER_PORT is required and must be positive")
	}
	if cfg.RedisHost == "" {
		return nil, fmt.Errorf("REDIS_HOST is required")
	}
	if cfg.IPLimit <= 0 {
		return nil, fmt.Errorf("IP_RATE_LIMIT must be positive")
	}
	if cfg.IPWindow <= 0 {
		return nil, fmt.Errorf("IP_RATE_WINDOW must be positive")
	}

	// Carrega tokens configurados dinamicamente
	// Formato: TOKEN_{nome}_LIMIT, TOKEN_{nome}_WINDOW, TOKEN_{nome}_BLOCK_TIME
	tokenNames := make(map[string]bool)

	// Busca todas as variáveis de ambiente que começam com TOKEN_
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]

		if strings.HasPrefix(key, "TOKEN_") {
			keyParts := strings.Split(key, "_")
			if len(keyParts) >= 3 {
				tokenName := strings.ToLower(keyParts[1])
				// Detecta qualquer variável TOKEN_* que tenha pelo menos 3 partes (TOKEN_abc123_*)
				// Isso inclui TOKEN_abc123=abc123 e TOKEN_abc123_LIMIT=100, etc.
				tokenNames[tokenName] = true
			}
		}
	}

	// Fallback: tenta buscar via viper.AllKeys() se não encontrou via os.Environ()
	if len(tokenNames) == 0 {
		for _, key := range viper.AllKeys() {
			upperKey := strings.ToUpper(key)
			if strings.HasPrefix(upperKey, "TOKEN_") && (strings.HasSuffix(upperKey, "_LIMIT") || strings.HasSuffix(upperKey, "_WINDOW") || strings.HasSuffix(upperKey, "_BLOCK_TIME")) {
				parts := strings.Split(upperKey, "_")
				if len(parts) >= 3 {
					tokenName := strings.ToLower(parts[1])
					tokenNames[tokenName] = true
				}
			}
		}
	}

	// Para cada token descoberto, carrega sua configuração
	for tokenName := range tokenNames {
		prefix := fmt.Sprintf("TOKEN_%s", strings.ToUpper(tokenName))

		// Busca diretamente pelas variáveis de ambiente usando os.Getenv
		// que funciona melhor em produção e com t.Setenv() dos testes
		limitStr := os.Getenv(prefix + "_LIMIT")
		windowStr := os.Getenv(prefix + "_WINDOW")
		blockTimeStr := os.Getenv(prefix + "_BLOCK_TIME")

		// Fallback para viper se os.Getenv não retornar valores
		if limitStr == "" {
			limitStr = viper.GetString(prefix + "_LIMIT")
		}
		if windowStr == "" {
			windowStr = viper.GetString(prefix + "_WINDOW")
		}
		if blockTimeStr == "" {
			blockTimeStr = viper.GetString(prefix + "_BLOCK_TIME")
		}

		limit := parseInt(limitStr)
		window := parseDuration(windowStr)
		blockTime := parseDuration(blockTimeStr)

		// Valida configuração do token
		if limit <= 0 || window <= 0 {
			continue // Ignora tokens mal configurados
		}

		// Busca o valor real do token (ex: TOKEN_test123=test123)
		tokenValue := os.Getenv(prefix)
		if tokenValue == "" {
			// Fallback para viper se os.Getenv não retornar valores
			tokenValue = viper.GetString(prefix)
		}

		// Se não encontrou o valor, usa o nome como fallback
		if tokenValue == "" {
			tokenValue = tokenName
		}

		// Usa o valor real do token como chave
		cfg.TokenConfigs[tokenValue] = TokenConfig{
			Limit:     limit,
			Window:    window,
			BlockTime: blockTime,
		}
	}

	return cfg, nil
}

// parseInt converte string para int, retorna 0 se falhar
func parseInt(s string) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}

// parseDuration converte string para duration, retorna 0 se falhar
func parseDuration(s string) time.Duration {
	if duration, err := time.ParseDuration(s); err == nil {
		return duration
	}
	return 0
}
