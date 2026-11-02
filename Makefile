GO ?= go
BIN := bin

.PHONY: help run fmt vet test build tidy clean check docker

help: ## Affiche cette aide
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

run: ## Lance le serveur (PORT=8080 par défaut)
	$(GO) run ./cmd/server

fmt: ## Formate le code
	$(GO) fmt ./...

vet: ## Analyse statique
	$(GO) vet ./...

test: ## Lance les tests
	$(GO) test ./...

build: ## Compile le serveur dans bin/
	$(GO) build -o $(BIN)/server ./cmd/server

tidy: ## Nettoie go.mod / go.sum
	$(GO) mod tidy

check: fmt vet test ## fmt + vet + test

clean: ## Supprime les binaires
	rm -rf $(BIN)

docker: ## Lance le serveur via Docker Compose
	docker compose up --build
