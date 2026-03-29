BINARY  := ./bin/stack
CONFIG  ?= stack.toml
COMPONENT ?=

.DEFAULT_GOAL := help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*##"}; {printf "  %-12s %s\n", $$1, $$2}'

$(BINARY): $(shell find cmd internal -name '*.go') go.mod go.sum
	go build -o $(BINARY) ./cmd/stack

prereqs: ## Install Python tool prerequisites via uv
	@if ! command -v uv >/dev/null 2>&1; then \
		echo "Error: uv is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo "  curl -LsSf https://astral.sh/uv/install.sh | sh"; \
		exit 1; \
	fi
	uv tool install "huggingface_hub[cli]"
	uv tool install podman-compose

submodules: ## Initialize and update all git submodules
	git submodule update --init --recursive

build: $(BINARY) ## Build the stack CLI tool

test: ## Run unit tests for the stack tool
	go test ./...

up: $(BINARY) ## Start all enabled services
	$(BINARY) --config $(CONFIG) up
	@sleep 5; \
	KC=$$(podman ps --filter name=stack-keycloak_keycloak --format '{{.Status}}' 2>/dev/null); \
	if echo "$$KC" | grep -q '(starting)'; then \
		echo ""; \
		echo "NOTE: Keycloak is still starting. First boot can take 2-5 minutes."; \
		echo "      Monitor with: podman logs -f stack-keycloak_keycloak_1"; \
		echo ""; \
	fi; \
	if podman logs stack-keycloak_keycloak_1 2>&1 | grep -q 'Killed.*java' 2>/dev/null; then \
		echo ""; \
		echo "WARNING: Keycloak's JVM is being OOM-killed by the container memory limit."; \
		echo "         This is common on first start when Keycloak builds its Quarkus cache."; \
		echo ""; \
		echo "  Fix: increase the memory limit in keycloak-testing/compose.yml:"; \
		echo ""; \
		echo "    deploy:"; \
		echo "      resources:"; \
		echo "        limits:"; \
		echo "          memory: 2g    # increase from default"; \
		echo ""; \
		echo "  Then run: make down && make up"; \
		echo ""; \
	fi

down: $(BINARY) ## Stop all services
	$(BINARY) --config $(CONFIG) down

restart: $(BINARY) ## Restart all services
	$(BINARY) --config $(CONFIG) restart

status: $(BINARY) ## Show service status
	$(BINARY) --config $(CONFIG) status

DOCS_DIR := $(shell awk '/^\[docs2vector\]/{f=1} f && /^docs_dir/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))
PG_HOST  := $(shell awk '/^\[postgres\]/{f=1} f && /^host/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))
PG_PORT  := $(shell awk '/^\[postgres\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG))
LLAMA_HOST := $(shell awk '/^\[llama\]/{f=1} f && /^host[^_]/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG))
LLAMA_PORT := $(shell awk '/^\[llama\]/{f=1} f && /^host_port/{print $$3; exit}' $(CONFIG))

# Reusable pre-flight check for ingest targets.
# Verifies docs_dir exists, PostgreSQL is reachable, and llama-server is up.
define ingest_preflight
	@if [ -n "$(DOCS_DIR)" ] && [ ! -d "$(DOCS_DIR)" ] && ! echo "$(ARGS)" | grep -q '\-\-docs-dir'; then \
		echo ""; \
		echo "ERROR: docs_dir '$(DOCS_DIR)' (from $(CONFIG)) does not exist."; \
		echo ""; \
		echo "  Either create it and populate it with documents:"; \
		echo "    sudo mkdir -p $(DOCS_DIR)"; \
		echo "    cp /path/to/your/docs/* $(DOCS_DIR)/"; \
		echo ""; \
		echo "  Or override it on the command line:"; \
		echo "    make $(1) ARGS=\"--docs-dir /path/to/your/docs\""; \
		echo ""; \
		echo "  Or change docs_dir in $(CONFIG) under [docs2vector]."; \
		echo ""; \
		exit 1; \
	fi
	@if [ "$(PG_HOST)" = "host.containers.internal" ]; then \
		NET_IP=$$(hostname -I 2>/dev/null | awk '{print $$1}'); \
		if pg_isready -h "$$NET_IP" -p $(PG_PORT) >/dev/null 2>&1; then :; \
		elif pg_isready -h localhost -p $(PG_PORT) >/dev/null 2>&1; then \
			echo ""; \
			echo "ERROR: PostgreSQL is running but only listening on localhost."; \
			echo "       Containers connect via host.containers.internal ($$NET_IP),"; \
			echo "       so PostgreSQL must accept non-loopback connections."; \
			echo ""; \
			echo "  Fix: edit postgresql.conf and set:"; \
			echo "    listen_addresses = '*'          # or '0.0.0.0' for IPv4 only"; \
			echo ""; \
			echo "  Then add a line to pg_hba.conf:"; \
			echo "    host  all  all  0.0.0.0/0  md5"; \
			echo ""; \
			echo "  Then restart PostgreSQL:"; \
			echo "    sudo systemctl restart postgresql"; \
			echo ""; \
			exit 1; \
		else \
			echo ""; \
			echo "ERROR: PostgreSQL is not reachable on port $(PG_PORT)."; \
			echo ""; \
			echo "  1. Check that PostgreSQL is running:"; \
			echo "       sudo systemctl status postgresql"; \
			echo ""; \
			echo "  2. If it is not installed yet:"; \
			echo "       make install-postgres"; \
			echo ""; \
			exit 1; \
		fi; \
	else \
		if ! pg_isready -h "$(PG_HOST)" -p $(PG_PORT) >/dev/null 2>&1; then \
			echo ""; \
			echo "ERROR: PostgreSQL is not reachable at $(PG_HOST):$(PG_PORT)."; \
			echo ""; \
			echo "  Check that PostgreSQL is running and accepting connections"; \
			echo "  on $(PG_HOST):$(PG_PORT), or update [postgres] in $(CONFIG)."; \
			echo ""; \
			exit 1; \
		fi; \
	fi
	@if [ "$(LLAMA_HOST)" = "host.containers.internal" ]; then \
		NET_IP=$$(hostname -I 2>/dev/null | awk '{print $$1}'); \
		if curl -sf "http://$$NET_IP:$(LLAMA_PORT)/health" >/dev/null 2>&1; then :; \
		elif curl -sf "http://localhost:$(LLAMA_PORT)/health" >/dev/null 2>&1; then \
			echo ""; \
			echo "ERROR: llama-server is running but only listening on localhost."; \
			echo "       Containers connect via host.containers.internal ($$NET_IP),"; \
			echo "       so llama-server must bind to 0.0.0.0."; \
			echo ""; \
			echo "  Start it with:"; \
			echo "    make llama-server     # already binds to 0.0.0.0"; \
			echo ""; \
			exit 1; \
		else \
			echo ""; \
			echo "ERROR: llama-server is not reachable at localhost:$(LLAMA_PORT)."; \
			echo ""; \
			echo "  Start the embedding server first:"; \
			echo "    make llama-server"; \
			echo ""; \
			exit 1; \
		fi; \
	else \
		if ! curl -sf "http://$(LLAMA_HOST):$(LLAMA_PORT)/health" >/dev/null 2>&1; then \
			echo ""; \
			echo "ERROR: llama-server is not reachable at $(LLAMA_HOST):$(LLAMA_PORT)."; \
			echo ""; \
			echo "  Start the embedding server first:"; \
			echo "    make llama-server"; \
			echo ""; \
			echo "  If it is running on a different host/port, update [llama] in $(CONFIG)."; \
			echo ""; \
			exit 1; \
		fi; \
	fi
endef

ingest: $(BINARY) ## Drop and reingest docs (ARGS="--docs-dir /path/to/docs")
	$(call ingest_preflight,ingest)
	$(BINARY) --config $(CONFIG) ingest $(ARGS)

ingest-add: $(BINARY) ## Add/upsert docs without dropping existing data (ARGS="--docs-dir /path/to/docs")
	$(call ingest_preflight,ingest-add)
	$(BINARY) --config $(CONFIG) ingest --no-drop $(ARGS)

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

eval-stability: ## Run evals N times and report pass-rate stats (EVAL_FILE=..., RUNS=25)
	./scripts/eval-stability.sh $(RUNS) $(EVAL_FILE)

eval: ## Run RAG evals (EVAL_FILE=path/to/evals.json, optional ARGS="--verbose")
ifndef EVAL_FILE
	$(error EVAL_FILE is required: make eval EVAL_FILE=path/to/evals.json)
endif
	MCP_SERVER_URL=http://localhost:$(MCP_PORT) \
	OIDC_PROVIDER=keycloak \
	KEYCLOAK_ISSUER=http://localhost:$(KC_PORT)/realms/$(KC_REALM) \
	KEYCLOAK_CLIENT_ID=$(KC_CLIENT_ID) \
	KEYCLOAK_CLIENT_SECRET=$(KC_CLIENT_SECRET) \
	EVAL_LIMIT=$${EVAL_LIMIT:-20} \
	./rag-mcp-server/scripts/eval.sh $(ARGS) $(EVAL_FILE)

install-postgres: ## Install PostgreSQL and pgvector (macOS or Linux)
	./scripts/install-postgres.sh

build-llama: ## Build llama-server from the llama.cpp submodule and install to /usr/local/bin
	./scripts/build-llama.sh

prep-database: ## Create raguser, ragdb, and enable pgvector on the host postgres
	psql postgres -c "CREATE ROLE raguser WITH LOGIN PASSWORD 'xP9#mQv7rL2kNw4J';"
	psql postgres -c "CREATE DATABASE ragdb OWNER raguser;"
	psql ragdb   -c "CREATE EXTENSION IF NOT EXISTS vector;"

llama-server: ## Run llama-server with nomic-embed-text-v1.5 on port 16000
	@if [ ! -f ./models/nomic-embed-text-v1.5.Q8_0.gguf ]; then \
		echo "Model not found, downloading..."; \
		$(MAKE) download-nomic-model; \
	fi
	llama-server \
		--model ./models/nomic-embed-text-v1.5.Q8_0.gguf \
		--embeddings --pooling mean \
		--host 0.0.0.0 --port 16000 \
		--ctx-size 8192 \
		--ubatch-size 2048 \
		--n-gpu-layers 99

llama-server-mxbai: ## Run llama-server with mxbai-embed-large-v1 on port 16000 (legacy)
	llama-server \
		--model ./models/mxbai-embed-large-v1.Q8_0.gguf \
		--embeddings --pooling cls \
		--host 0.0.0.0 --port 16000 \
		--n-gpu-layers 99

reranker-server: ## Run llama-server with bge-reranker-v2-m3 on port 16001
	@if [ ! -f ./models/bge-reranker-v2-m3-Q8_0.gguf ]; then \
		echo "Model not found, downloading..."; \
		$(MAKE) download-reranker-model; \
	fi
	llama-server \
		--model ./models/bge-reranker-v2-m3-Q8_0.gguf \
		--reranking \
		--host 0.0.0.0 --port 16001 \
		--n-gpu-layers 99

download-nomic-model: ## Download nomic-embed-text-v1.5 GGUF model into ./models
	@if ! command -v hf >/dev/null 2>&1; then \
		echo "Error: hf (Hugging Face CLI) is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo "  uv tool install \"huggingface_hub[cli]\""; \
		exit 1; \
	fi
	mkdir -p ./models
	hf download nomic-ai/nomic-embed-text-v1.5-GGUF \
		nomic-embed-text-v1.5.Q8_0.gguf \
		--local-dir ./models

download-reranker-model: ## Download bge-reranker-v2-m3 GGUF model into ./models
	@if ! command -v hf >/dev/null 2>&1; then \
		echo "Error: hf (Hugging Face CLI) is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo "  uv tool install \"huggingface_hub[cli]\""; \
		exit 1; \
	fi
	mkdir -p ./models
	hf download gpustack/bge-reranker-v2-m3-GGUF \
		bge-reranker-v2-m3-Q8_0.gguf \
		--local-dir ./models

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
