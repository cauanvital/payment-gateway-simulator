.DEFAULT_GOAL := help
BINARY := bin/server
PKG := ./...
INTEGRATION_COVERAGE := coverage-integration

.PHONY: help
help: ## Lista os comandos disponíveis
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Roda o servidor localmente
	go run ./cmd/server

.PHONY: build
build: ## Compila o binário em bin/server
	go build -o $(BINARY) ./cmd/server

.PHONY: test
test: ## Roda os testes
	go test -race -count=1 $(PKG)

.PHONY: test-integration
test-integration: ## Roda os testes de integraÃ§Ã£o com Postgres descartÃ¡vel
	@trap 'docker compose -f tests/integration/docker-compose.yml down -v' EXIT; \
		docker compose -f tests/integration/docker-compose.yml up -d --wait; \
		TEST_DATABASE_URL='postgres://pgsim:pgsim@localhost:55432/pgsim_test?sslmode=disable' go test -tags=integration -count=1 -covermode=atomic -coverpkg=./internal/... -coverprofile=$(INTEGRATION_COVERAGE) ./tests/integration

.PHONY: cover
cover: ## Roda os testes com relatório de cobertura
	go test -race -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage-unit $(PKG)
	go tool cover -func=coverage-unit

.PHONY: fmt
fmt: ## Formata o código
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Falha se houver arquivos não formatados
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "Arquivos não formatados:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Roda o go vet
	go vet $(PKG)

.PHONY: tidy
tidy: ## Sincroniza as dependências do go.mod
	go mod tidy

.PHONY: check
check: fmt-check vet test ## Roda todas as verificações de qualidade

.PHONY: up
up: ## Sobe app + banco via docker compose
	docker compose up --build

.PHONY: down
down: ## Derruba os containers
	docker compose down

.PHONY: clean
clean: ## Remove artefatos de build
	rm -rf bin coverage coverage-unit coverage-integration coverage-all coverage-unit.txt coverage-integration.txt coverage-all.txt
