BINARY  := ./bin/stack
CONFIG  ?= stack.toml
COMPONENT ?=

.DEFAULT_GOAL := help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*##"}; {printf "  %-12s %s\n", $$1, $$2}'

$(BINARY): $(shell find cmd internal -name '*.go') go.mod go.sum
	go build -o $(BINARY) ./cmd/stack

build: $(BINARY) ## Build the stack CLI tool

test: ## Run unit tests for the stack tool
	go test ./...

up: $(BINARY) ## Start all enabled services
	$(BINARY) --config $(CONFIG) up

down: $(BINARY) ## Stop all services
	$(BINARY) --config $(CONFIG) down

restart: $(BINARY) ## Restart all services
	$(BINARY) --config $(CONFIG) restart

status: $(BINARY) ## Show service status
	$(BINARY) --config $(CONFIG) status

ingest: $(BINARY) ## Run docs2vector ingestion (ARGS="--docs-dir /path/to/docs")
	$(BINARY) --config $(CONFIG) ingest $(ARGS)

logs: $(BINARY) ## Tail logs (COMPONENT= to filter)
	$(BINARY) --config $(CONFIG) logs $(COMPONENT)

generate: $(BINARY) ## Generate component configs without starting
	$(BINARY) --config $(CONFIG) generate

validate: $(BINARY) ## Validate stack.toml
	$(BINARY) --config $(CONFIG) validate

MCP_PORT          := $(shell awk '/^\[rag_mcp_server\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG))
KC_PORT           := $(shell awk '/^\[keycloak\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG))
KC_REALM          := $(shell awk '/^\[keycloak\]/{f=1} f && /^realm/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))
KC_CLIENT_ID      := $(shell awk '/^\[keycloak\]/{f=1} f && /^m2m_client_id/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))
KC_CLIENT_SECRET  := $(shell awk '/^\[keycloak\]/{f=1} f && /^m2m_client_secret/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))

eval: ## Run RAG evals (EVAL_FILE=path/to/evals.json, optional ARGS="--verbose")
ifndef EVAL_FILE
	$(error EVAL_FILE is required: make eval EVAL_FILE=path/to/evals.json)
endif
	MCP_SERVER_URL=http://localhost:$(MCP_PORT) \
	OIDC_PROVIDER=keycloak \
	KEYCLOAK_ISSUER=http://localhost:$(KC_PORT)/realms/$(KC_REALM) \
	KEYCLOAK_CLIENT_ID=$(KC_CLIENT_ID) \
	KEYCLOAK_CLIENT_SECRET=$(KC_CLIENT_SECRET) \
	./rag-mcp-server/scripts/eval.sh $(ARGS) $(EVAL_FILE)

prep-database: ## Create raguser, ragdb, and enable pgvector on the host postgres
	psql postgres -c "CREATE ROLE raguser WITH LOGIN PASSWORD 'xP9#mQv7rL2kNw4J';"
	psql postgres -c "CREATE DATABASE ragdb OWNER raguser;"
	psql ragdb   -c "CREATE EXTENSION IF NOT EXISTS vector;"

llama-server: ## Run llama-server with mxbai-embed-large-v1 on port 16000
	llama-server \
		--model ./models/mxbai-embed-large-v1.Q8_0.gguf \
		--embeddings --pooling cls \
		--host 0.0.0.0 --port 16000 \
		--n-gpu-layers 99

download-model: ## Download mxbai-embed-large-v1 GGUF model into ./models
	@if ! command -v hf >/dev/null 2>&1; then \
		echo "Error: hf (Hugging Face CLI) is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo "  uv tool install \"huggingface_hub[cli]\""; \
		echo ""; \
		echo "If you don't have uv:"; \
		echo "  curl -LsSf https://astral.sh/uv/install.sh | sh"; \
		exit 1; \
	fi
	mkdir -p ./models
	hf download ChristianAzinn/mxbai-embed-large-v1-gguf \
		mxbai-embed-large-v1.Q8_0.gguf \
		--local-dir ./models

clean: ## Remove generated files and binary
	rm -rf .stack
	rm -f keycloak-testing/.env logto-testing/.env
	rm -f rag-mcp-server/.env rag-mcp-server/config.toml
	rm -f docs2vector/.env docs2vector/config.toml
	rm -f $(BINARY)
