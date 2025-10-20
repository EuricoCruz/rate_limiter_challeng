package redis

import "github.com/redis/go-redis/v9"

// tokenBucketScript implementa o algoritmo Token Bucket em Lua para execução atômica no Redis.
// Este é o CORE do rate limiter, garantindo operações thread-safe e consistentes.
//
// O script Lua é executado atomicamente no Redis, evitando race conditions
// quando múltiplas requisições simultâneas tentam consumir tokens.
//
// Estrutura das KEYS:
// - KEYS[1]: tokens_key - armazena o número atual de tokens (ex: "rate_limit:ip:192.168.1.1:tokens")
// - KEYS[2]: last_refill_key - armazena o timestamp do último refill (ex: "rate_limit:ip:192.168.1.1:last_refill")
//
// Estrutura dos ARGV:
// - ARGV[1]: capacity - capacidade máxima do bucket (ex: 10 tokens)
// - ARGV[2]: window_seconds - duração da janela em segundos (ex: 1 segundo)
// - ARGV[3]: now - timestamp atual em segundos (ex: 1729252800)
//
// Retorno: [allowed, current_tokens, capacity]
// - allowed: 1 se permitido, 0 se bloqueado
// - current_tokens: número atual de tokens no bucket
// - capacity: capacidade máxima do bucket
var tokenBucketScript = redis.NewScript(`
-- ============================================================================
-- TOKEN BUCKET ALGORITHM - Implementação Lua para Redis
-- ============================================================================
-- Este script implementa o algoritmo Token Bucket para rate limiting de forma
-- atômica e thread-safe, garantindo consistência mesmo com alta concorrência.
--
-- Algoritmo Token Bucket:
-- 1. O bucket tem uma capacidade máxima (ex: 10 tokens)
-- 2. Tokens são adicionados continuamente a uma taxa fixa (ex: 10 tokens/segundo)
-- 3. Cada requisição consome 1 token
-- 4. Se não há tokens disponíveis, a requisição é bloqueada
-- ============================================================================

-- ============================================================================
-- CONFIGURAÇÃO DAS CHAVES E PARÂMETROS
-- ============================================================================

-- Chaves Redis onde serão armazenados os dados do rate limiter
local tokens_key = KEYS[1]       -- Chave para armazenar tokens atuais (ex: "rate_limit:ip:192.168.1.1:tokens")
local last_refill_key = KEYS[2]  -- Chave para armazenar timestamp do último refill (ex: "rate_limit:ip:192.168.1.1:last_refill")

-- Parâmetros de configuração do rate limiter
local capacity = tonumber(ARGV[1])      -- Capacidade máxima do bucket (ex: 10 tokens)
local window_seconds = tonumber(ARGV[2]) -- Janela de tempo em segundos (ex: 1 segundo)
local now = tonumber(ARGV[3])           -- Timestamp atual em segundos (ex: 1729252800)

-- ============================================================================
-- RECUPERAÇÃO DO ESTADO ATUAL
-- ============================================================================

-- Busca o número atual de tokens no Redis, ou usa a capacidade máxima se não existir
-- Isso significa que um bucket novo começa "cheio" de tokens
local tokens = tonumber(redis.call('GET', tokens_key)) or capacity

-- Busca o timestamp do último refill, ou usa o timestamp atual se não existir
-- Isso significa que um bucket novo é criado com o timestamp atual
local last_refill = tonumber(redis.call('GET', last_refill_key)) or now

-- ============================================================================
-- TOKEN BUCKET ALGORITHM - CORE LOGIC
-- ============================================================================

-- PASSO 1: Calcula o tempo decorrido desde o último refill em segundos
-- Esta é a base para calcular quantos tokens devem ser adicionados
local elapsed = now - last_refill

-- PASSO 2: Calcula a taxa de refill (tokens adicionados por segundo)
-- Exemplo: se capacity=10 e window_seconds=1, então refill_rate=10 tokens/segundo
local refill_rate = capacity / window_seconds

-- PASSO 3: Calcula quantos tokens devem ser adicionados baseado no tempo decorrido
-- Exemplo: se elapsed=0.5s e refill_rate=10, então tokens_to_add=5 tokens
local tokens_to_add = elapsed * refill_rate

-- PASSO 4: Adiciona tokens ao bucket, mas nunca excede a capacidade máxima
-- Esta é uma característica fundamental do Token Bucket: o bucket tem limite máximo
tokens = math.min(capacity, tokens + tokens_to_add)

-- ============================================================================
-- DECISÃO DE PERMISSÃO E CONSUMO DE TOKEN
-- ============================================================================

-- PASSO 5: Tenta consumir 1 token para esta requisição
if tokens >= 1 then
    -- ========================================================================
    -- ✅ REQUISIÇÃO PERMITIDA: há tokens suficientes
    -- ========================================================================
    
    -- Consome 1 token do bucket
    tokens = tokens - 1
    
    -- Salva o novo estado no Redis com TTL de 1 hora para evitar acúmulo de chaves órfãs
    -- TTL de 3600 segundos (1 hora) é suficiente para a maioria dos casos de uso
    redis.call('SETEX', tokens_key, 3600, tostring(tokens))
    redis.call('SETEX', last_refill_key, 3600, tostring(now))
    
    -- Retorna resultado de sucesso: [allowed=1, current_tokens, capacity]
    return {1, tokens, capacity}
    
else
    -- ========================================================================
    -- ❌ REQUISIÇÃO BLOQUEADA: não há tokens disponíveis
    -- ========================================================================
    
    -- Mesmo quando bloqueado, atualiza o timestamp para calcular corretamente
    -- o próximo refill na próxima requisição
    redis.call('SETEX', last_refill_key, 3600, tostring(now))
    
    -- Retorna resultado de bloqueio: [allowed=0, current_tokens, capacity]
    -- O valor de current_tokens pode ser útil para debugging e monitoramento
    return {0, tokens, capacity}
end
`)
