BINARY  := ./bin/stack
CONFIG  ?= stack.toml
COMPONENT ?=

.DEFAULT_GOAL := help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*##"}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "  Host setup targets are in installers/  (cd installers && make help)"

$(BINARY): $(shell find cmd internal -name '*.go') go.mod go.sum
	go build -o $(BINARY) ./cmd/stack

build: $(BINARY) ## Build the stack CLI and rag-mcp-server tools
	$(MAKE) -C rag-mcp-server build

test: ## Run unit tests for the stack tool
	go test ./...

clean: ## Remove generated files and binaries
	rm -rf .stack
	rm -f keycloak-testing/.env logto-testing/.env
	rm -f rag-mcp-server/.env rag-mcp-server/config.toml
	rm -f docs2vector/.env docs2vector/config.toml
	rm -f $(BINARY)

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

status: ## Show status of all services (containers, inference servers, database)
	@echo "── MCP Server (container) ──"
	@if curl -sf http://localhost:$(MCP_PORT)/healthz >/dev/null 2>&1; then \
		echo "  rag-mcp-server :$(MCP_PORT)  UP"; \
	else \
		echo "  rag-mcp-server :$(MCP_PORT)  DOWN"; \
	fi
	@echo ""
	@echo "── Inference Servers (host) ──"
	@if curl -sf http://localhost:$(LLAMA_PORT)/health >/dev/null 2>&1; then \
		echo "  embed-server   :$(LLAMA_PORT)  UP"; \
	else \
		echo "  embed-server   :$(LLAMA_PORT)  DOWN"; \
	fi
	@if curl -sf http://localhost:$(RERANKER_PORT)/health >/dev/null 2>&1; then \
		echo "  reranker       :$(RERANKER_PORT)  UP"; \
	else \
		echo "  reranker       :$(RERANKER_PORT)  DOWN"; \
	fi
	@echo ""
	@echo "── Database ──"
	@if pg_isready -h $(PG_HOST) -p $(PG_PORT) >/dev/null 2>&1; then \
		echo "  postgresql     $(PG_HOST):$(PG_PORT)  UP"; \
	elif [ "$(PG_HOST)" = "host.containers.internal" ] && pg_isready -h localhost -p $(PG_PORT) >/dev/null 2>&1; then \
		echo "  postgresql     localhost:$(PG_PORT)  UP"; \
	else \
		echo "  postgresql     $(PG_HOST):$(PG_PORT)  DOWN"; \
	fi
	@echo ""
	@echo "── Containers ──"
	@$(BINARY) --config $(CONFIG) status 2>/dev/null || echo "  (stack not running)"

logs: $(BINARY) ## Tail logs (COMPONENT= to filter)
	$(BINARY) --config $(CONFIG) logs $(COMPONENT)

generate: $(BINARY) ## Generate component configs without starting
	$(BINARY) --config $(CONFIG) generate

validate: $(BINARY) ## Validate stack.toml
	$(BINARY) --config $(CONFIG) validate

# ── Config values parsed from stack.toml ─────────────────────────────────────

DOCS_DIR      = $(shell awk '/^\[docs2vector\]/{f=1} f && /^docs_dir/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)
PG_HOST       = $(or $(shell awk '/^\[postgres\]/{f=1} f && /^host/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null),localhost)
PG_PORT       = $(or $(shell awk '/^\[postgres\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG) 2>/dev/null),5432)
LLAMA_HOST    = $(or $(shell awk '/^\[llama\]/{f=1} f && /^host[^_]/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null),localhost)
LLAMA_PORT    = $(or $(shell awk '/^\[llama\]/{f=1} f && /^host_port/{print $$3; exit}' $(CONFIG) 2>/dev/null),16000)
RERANKER_PORT = $(or $(shell awk '/^\[reranker\]/{f=1} f && /^host_port/{print $$3; exit}' $(CONFIG) 2>/dev/null),16001)
MCP_PORT      = $(or $(shell awk '/^\[rag_mcp_server\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG) 2>/dev/null),15080)
KC_PORT       = $(or $(shell awk '/^\[keycloak\]/{f=1} f && /^port/{print $$3; exit}' $(CONFIG) 2>/dev/null),8080)
KC_REALM      = $(or $(shell awk '/^\[keycloak\]/{f=1} f && /^realm/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null),dev)
KC_CLIENT_ID  = $(shell awk '/^\[keycloak\]/{f=1} f && /^m2m_client_id/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)
KC_CLIENT_SECRET = $(shell awk '/^\[keycloak\]/{f=1} f && /^m2m_client_secret/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)
DEFAULT_EVAL_FILE = $(shell awk '/^\[rag_mcp_server\]/{f=1} f && /^eval_file/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)

# ── Ingest pre-flight checks ─────────────────────────────────────────────────
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
			echo "  Check that PostgreSQL is running:"; \
			echo "    sudo systemctl status postgresql"; \
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
			echo "  You need to have llama-server installed and listening on 0.0.0.0."; \
			echo ""; \
			exit 1; \
		else \
			echo ""; \
			echo "ERROR: llama-server is not reachable at localhost:$(LLAMA_PORT)."; \
			echo ""; \
			echo "  You need to have llama-server installed and running."; \
			echo ""; \
			exit 1; \
		fi; \
	else \
		if ! curl -sf "http://$(LLAMA_HOST):$(LLAMA_PORT)/health" >/dev/null 2>&1; then \
			echo ""; \
			echo "ERROR: llama-server is not reachable at $(LLAMA_HOST):$(LLAMA_PORT)."; \
			echo ""; \
			echo "  You need to have llama-server installed and running,"; \
			echo "  or update [llama] in $(CONFIG)."; \
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

# ── Eval targets ─────────────────────────────────────────────────────────────

EVAL_FILE ?= $(DEFAULT_EVAL_FILE)

eval: ## Run RAG evals (EVAL_FILE=path or rag_mcp_server.eval_file in stack.toml)
	@if [ -z "$(EVAL_FILE)" ]; then \
		echo "ERROR: no eval file specified."; \
		echo ""; \
		echo "  Either pass it on the command line:"; \
		echo "    make eval EVAL_FILE=path/to/evals.json"; \
		echo ""; \
		echo "  Or set rag_mcp_server.eval_file in $(CONFIG)."; \
		exit 1; \
	fi
	MCP_SERVER_URL=http://localhost:$(MCP_PORT) \
	OIDC_PROVIDER=keycloak \
	KEYCLOAK_ISSUER=http://localhost:$(KC_PORT)/realms/$(KC_REALM) \
	KEYCLOAK_CLIENT_ID=$(KC_CLIENT_ID) \
	KEYCLOAK_CLIENT_SECRET=$(KC_CLIENT_SECRET) \
	EVAL_LIMIT=$${EVAL_LIMIT:-20} \
	./rag-mcp-server/bin/eval $(ARGS) $(EVAL_FILE)

eval-stability: ## Run evals N times and report pass-rate stats (EVAL_FILE=..., RUNS=25)
	./scripts/eval-stability.sh $(RUNS) $(EVAL_FILE)

# ── Inference servers ────────────────────────────────────────────────────────

stop-inference-servers: ## Stop embedding and reranker background servers
	@echo "stopping llama-server processes..."
	@pkill -f 'llama-server.*--embeddings' 2>/dev/null && echo "  embed server stopped" || echo "  embed server not running"
	@pkill -f 'llama-server.*--reranking' 2>/dev/null && echo "  reranker stopped" || echo "  reranker not running"

run-inference-servers: ## Start embedding and reranker servers in the background
	@if ! command -v llama-server >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: You need to have llama-server installed."; \
		echo "  See: cd installers && make build-llama"; \
		echo ""; \
		exit 1; \
	fi
	@if [ ! -f ./models/nomic-embed-text-v1.5.Q8_0.gguf ] || [ ! -f ./models/bge-reranker-v2-m3-Q8_0.gguf ]; then \
		echo ""; \
		echo "ERROR: Model files not found in ./models/."; \
		echo "  Download them with: cd installers && make download-models"; \
		echo ""; \
		exit 1; \
	fi
	@eval $$(./installers/scripts/detect-gpu.sh); \
	echo "starting embed server on port $(LLAMA_PORT)..."; \
	nohup llama-server \
		--model ./models/nomic-embed-text-v1.5.Q8_0.gguf \
		--embeddings --pooling mean \
		--host 0.0.0.0 --port $(LLAMA_PORT) \
		--ctx-size 8192 \
		--n-gpu-layers 99 \
		$${LLAMA_GPU_FLAGS} \
		> /tmp/llama-embed.log 2>&1 & \
	echo "  PID: $$!  log: /tmp/llama-embed.log"; \
	echo "starting reranker server on port $(RERANKER_PORT)..."; \
	nohup llama-server \
		--model ./models/bge-reranker-v2-m3-Q8_0.gguf \
		--reranking \
		--host 0.0.0.0 --port $(RERANKER_PORT) \
		--ctx-size 8192 \
		--n-gpu-layers 99 \
		$${LLAMA_GPU_FLAGS} \
		> /tmp/llama-reranker.log 2>&1 & \
	echo "  PID: $$!  log: /tmp/llama-reranker.log"

.PHONY: help build test clean up down restart status logs generate validate ingest ingest-add eval eval-stability run-inference-servers stop-inference-servers
