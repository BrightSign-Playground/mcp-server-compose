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
