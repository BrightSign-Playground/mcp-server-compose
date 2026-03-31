CONFIG  ?= ../stack.toml
MODELS_DIR ?= ../models

# Parse postgres credentials from stack.toml.
PG_USER     = $(shell awk '/^\[postgres\]/{f=1} f && /^user/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)
PG_PASSWORD = $(shell awk '/^\[postgres\]/{f=1} f && /^password/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)
PG_DATABASE = $(shell awk '/^\[postgres\]/{f=1} f && /^database/{gsub(/"/, "", $$3); print $$3; exit}' $(CONFIG) 2>/dev/null)

# Inference server ports from stack.toml.
LLAMA_PORT    = $(or $(shell awk '/^\[llama\]/{f=1} f && /^host_port/{print $$3; exit}' $(CONFIG) 2>/dev/null),16000)
RERANKER_PORT = $(or $(shell awk '/^\[reranker\]/{f=1} f && /^host_port/{print $$3; exit}' $(CONFIG) 2>/dev/null),16001)

.DEFAULT_GOAL := help

help: ## Show available targets
	@echo "Host setup and infrastructure targets (run from installers/):"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*##"}; {printf "  %-24s %s\n", $$1, $$2}'

prereqs: ## Install Python tool prerequisites (uv, huggingface_hub, podman-compose)
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
	git -C .. submodule update --init --recursive

install-postgres: ## Install PostgreSQL and pgvector (macOS or Linux)
	./scripts/install-postgres.sh

prep-database: ## Create database role, database, and enable pgvector (reads from stack.toml)
	@if [ ! -f "$(CONFIG)" ]; then \
		echo "Error: $(CONFIG) not found."; \
		echo "  Copy stack.toml.example to stack.toml and fill in credentials."; \
		exit 1; \
	fi
	@if [ -z "$(PG_USER)" ] || [ -z "$(PG_PASSWORD)" ] || [ -z "$(PG_DATABASE)" ]; then \
		echo "Error: could not read postgres.user, postgres.password, or postgres.database from $(CONFIG)"; \
		exit 1; \
	fi
	psql postgres -c "CREATE ROLE $(PG_USER) WITH LOGIN PASSWORD '$(PG_PASSWORD)';" || true
	psql postgres -c "CREATE DATABASE $(PG_DATABASE) OWNER $(PG_USER);" || true
	psql $(PG_DATABASE) -c "CREATE EXTENSION IF NOT EXISTS vector;"

build-llama: ## Build llama-server from the llama.cpp submodule and install to /usr/local/bin
	./scripts/build-llama.sh

download-models: ## Download all GGUF models into ./models
	@if ! command -v hf >/dev/null 2>&1; then \
		echo "Error: hf (Hugging Face CLI) is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo "  make prereqs"; \
		exit 1; \
	fi
	mkdir -p $(MODELS_DIR)
	hf download nomic-ai/nomic-embed-text-v1.5-GGUF \
		nomic-embed-text-v1.5.Q8_0.gguf \
		--local-dir $(MODELS_DIR)
	hf download gpustack/bge-reranker-v2-m3-GGUF \
		bge-reranker-v2-m3-Q8_0.gguf \
		--local-dir $(MODELS_DIR)
	hf download ChristianAzinn/mxbai-embed-large-v1-gguf \
		mxbai-embed-large-v1.Q8_0.gguf \
		--local-dir $(MODELS_DIR)

llama-server: ## Run llama-server with nomic-embed-text-v1.5
	@if ! command -v llama-server >/dev/null 2>&1; then \
		echo "Error: llama-server is not installed."; \
		echo ""; \
		echo "  Build it with:"; \
		echo "    make build-llama"; \
		exit 1; \
	fi
	@if [ ! -f $(MODELS_DIR)/nomic-embed-text-v1.5.Q8_0.gguf ]; then \
		echo "Model not found, downloading..."; \
		$(MAKE) download-models; \
	fi
	@eval $$(./scripts/detect-gpu.sh); \
	echo "starting embed server: parallel=$${LLAMA_PARALLEL} batch=$${LLAMA_BATCH} ubatch=$${LLAMA_UBATCH}"; \
	llama-server \
		--model $(MODELS_DIR)/nomic-embed-text-v1.5.Q8_0.gguf \
		--embeddings --pooling mean \
		--host 0.0.0.0 --port $(LLAMA_PORT) \
		--ctx-size 8192 \
		--n-gpu-layers 99 \
		$${LLAMA_GPU_FLAGS}

reranker-server: ## Run llama-server with bge-reranker-v2-m3
	@if ! command -v llama-server >/dev/null 2>&1; then \
		echo "Error: llama-server is not installed."; \
		echo ""; \
		echo "  Build it with:"; \
		echo "    make build-llama"; \
		exit 1; \
	fi
	@if [ ! -f $(MODELS_DIR)/bge-reranker-v2-m3-Q8_0.gguf ]; then \
		echo "Model not found, downloading..."; \
		$(MAKE) download-models; \
	fi
	@eval $$(./scripts/detect-gpu.sh); \
	echo "starting reranker: parallel=$${LLAMA_PARALLEL} batch=$${LLAMA_BATCH} ubatch=$${LLAMA_UBATCH}"; \
	llama-server \
		--model $(MODELS_DIR)/bge-reranker-v2-m3-Q8_0.gguf \
		--reranking \
		--host 0.0.0.0 --port $(RERANKER_PORT) \
		--ctx-size 8192 \
		--n-gpu-layers 99 \
		$${LLAMA_GPU_FLAGS}

.PHONY: help prereqs submodules install-postgres prep-database build-llama download-models llama-server reranker-server
