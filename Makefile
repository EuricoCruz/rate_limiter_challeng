.PHONY: help build test test-unit test-integration run docker-up docker-down docker-build logs clean

help: ## Mostra este help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Compila a aplicação
	@echo "🔨 Building..."
	@go build -o bin/rate-limiter cmd/server/main.go
	@echo "✅ Build complete: bin/rate-limiter"

test-unit: ## Roda testes unitários
	@echo "🧪 Running unit tests..."
	@go test ./internal/... -v -cover -race

test-integration: ## Roda testes de integração (precisa Redis)
	@echo "🧪 Running integration tests..."
	@docker-compose -f docker-compose.test.yml up -d
	@sleep 3
	@go test -tags=integration ./tests/integration/... -v
	@docker-compose -f docker-compose.test.yml down -v

test: test-unit test-integration ## Roda todos os testes

run: ## Roda aplicação localmente
	@echo "🚀 Starting application..."
	@go run cmd/server/main.go

docker-build: ## Build da imagem Docker
	@echo "🐳 Building Docker image..."
	@docker build -t rate-limiter:latest .

docker-up: ## Sobe containers com docker-compose
	@echo "🐳 Starting containers..."
	@docker-compose up -d
	@echo "✅ Containers started!"
	@echo "📊 Check status: make docker-ps"
	@echo "📋 Check logs: make logs"

docker-down: ## Para e remove containers
	@echo "🛑 Stopping containers..."
	@docker-compose down -v

docker-ps: ## Lista containers rodando
	@docker-compose ps

logs: ## Mostra logs da aplicação
	@docker-compose logs -f app

redis-cli: ## Conecta no Redis CLI
	@docker-compose exec redis redis-cli

clean: ## Limpa arquivos gerados
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@docker-compose down -v
	@docker-compose -f docker-compose.test.yml down -v
	@echo "✅ Cleaned!"

coverage: ## Gera relatório de coverage
	@echo "📊 Generating coverage report..."
	@go test ./internal/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

lint: ## Roda linters (precisa golangci-lint instalado)
	@echo "🔍 Running linters..."
	@golangci-lint run ./...
