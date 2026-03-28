BINARY  := ./bin/stack
CONFIG  ?= stack.toml
COMPONENT ?=

.DEFAULT_GOAL := help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*##"}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build the stack CLI tool
	go build -o $(BINARY) ./cmd/stack

test: ## Run unit tests for the stack tool
	go test ./...

up: build ## Start all enabled services
	$(BINARY) --config $(CONFIG) up

down: build ## Stop all services
	$(BINARY) --config $(CONFIG) down

restart: build ## Restart all services
	$(BINARY) --config $(CONFIG) restart

status: build ## Show service status
	$(BINARY) --config $(CONFIG) status

ingest: build ## Run docs2vector ingestion
	$(BINARY) --config $(CONFIG) ingest

logs: build ## Tail logs (COMPONENT= to filter)
	$(BINARY) --config $(CONFIG) logs $(COMPONENT)

generate: build ## Generate component configs without starting
	$(BINARY) --config $(CONFIG) generate

validate: build ## Validate stack.toml
	$(BINARY) --config $(CONFIG) validate

clean: ## Remove generated files and binary
	rm -rf .stack
	rm -f keycloak-testing/.env logto-testing/.env
	rm -f rag-mcp-server/.env rag-mcp-server/config.toml
	rm -f docs2vector/.env docs2vector/config.toml
	rm -f $(BINARY)
