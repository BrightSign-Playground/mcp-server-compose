package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
)

// All generates all ephemeral configuration files. Creates .stack/ if absent.
// Returns a list of paths written.
func All(cfg *config.Config, eng engine.Engine, repoRoot string) ([]string, error) {
	d := deriveAll(cfg, eng)

	stackDir := filepath.Join(repoRoot, ".stack")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		return nil, fmt.Errorf("creating .stack dir: %w", err)
	}

	var written []string
	write := func(path string, fn func() error) error {
		if err := fn(); err != nil {
			return fmt.Errorf("generating %s: %w", path, err)
		}
		written = append(written, path)
		return nil
	}

	// Shared postgres files.
	postgresEnvPath := filepath.Join(stackDir, "postgres.env")
	if err := write(postgresEnvPath, func() error {
		return writeEnvFile(postgresEnvPath, postgresEnvVars(cfg))
	}); err != nil {
		return nil, err
	}

	postgresComposePath := filepath.Join(stackDir, "compose.postgres.yml")
	if err := write(postgresComposePath, func() error {
		return writeTextFile(postgresComposePath, postgresComposeYAML(), 0644)
	}); err != nil {
		return nil, err
	}

	// Keycloak .env.
	keycloakEnvPath := filepath.Join(repoRoot, "keycloak-testing", ".env")
	if err := write(keycloakEnvPath, func() error {
		return writeEnvFile(keycloakEnvPath, keycloakEnvVars(cfg))
	}); err != nil {
		return nil, err
	}

	// Logto .env.
	logtoEnvPath := filepath.Join(repoRoot, "logto-testing", ".env")
	if err := write(logtoEnvPath, func() error {
		return writeEnvFile(logtoEnvPath, logtoEnvVars(cfg))
	}); err != nil {
		return nil, err
	}

	// rag-mcp-server .env and config.toml.
	ragEnvPath := filepath.Join(repoRoot, "rag-mcp-server", ".env")
	if err := write(ragEnvPath, func() error {
		return writeEnvFile(ragEnvPath, ragEnvVars(cfg, d))
	}); err != nil {
		return nil, err
	}

	ragConfigPath := filepath.Join(repoRoot, "rag-mcp-server", "config.toml")
	if err := write(ragConfigPath, func() error {
		return writeTextFile(ragConfigPath, ragConfigTOML(cfg, d), 0644)
	}); err != nil {
		return nil, err
	}

	// docs2vector .env and config.toml.
	docs2vEnvPath := filepath.Join(repoRoot, "docs2vector", ".env")
	if err := write(docs2vEnvPath, func() error {
		return writeEnvFile(docs2vEnvPath, docs2vectorEnvVars(cfg, d))
	}); err != nil {
		return nil, err
	}

	docs2vConfigPath := filepath.Join(repoRoot, "docs2vector", "config.toml")
	if err := write(docs2vConfigPath, func() error {
		return writeTextFile(docs2vConfigPath, docs2vectorConfigTOML(cfg, d), 0644)
	}); err != nil {
		return nil, err
	}

	return written, nil
}

// writeEnvFile writes a KEY=value env file. Values must not contain newlines.
// File is written with 0600 permissions (may contain secrets).
func writeEnvFile(path string, vars map[string]string) error {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		v := vars[k]
		if strings.ContainsRune(v, '\n') {
			return fmt.Errorf("env var %s: value must not contain newlines", k)
		}
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(v)
		builder.WriteByte('\n')
	}

	return writeTextFile(path, builder.String(), 0600)
}

// writeTextFile atomically writes content to path with the given permissions.
func writeTextFile(path, content string, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ─── env var maps ────────────────────────────────────────────────────────────

func postgresEnvVars(cfg *config.Config) map[string]string {
	pg := cfg.Postgres
	image := pg.Image
	if image == "" {
		image = "docker.io/pgvector/pgvector:pg17"
	}
	return map[string]string{
		"POSTGRES_IMAGE":    image,
		"POSTGRES_USER":     pg.User,
		"POSTGRES_PASSWORD": pg.Password,
		"POSTGRES_DB":       pg.Database,
		"POSTGRES_PORT":     fmt.Sprintf("%d", pg.Port),
	}
}

func keycloakEnvVars(cfg *config.Config) map[string]string {
	kc := cfg.Keycloak
	return map[string]string{
		"KC_PORT":             fmt.Sprintf("%d", kc.Port),
		"KC_DB_PORT":          fmt.Sprintf("%d", kc.DBPort),
		"KC_ADMIN_USER":       kc.AdminUser,
		"KC_ADMIN_PASSWORD":   kc.AdminPassword,
		"KC_REALM":            kc.Realm,
		"KC_API_CLIENT_ID":    kc.APIClientID,
		"KC_M2M_CLIENT_ID":    kc.M2MClientID,
		"KC_M2M_CLIENT_SECRET": kc.M2MClientSecret,
		"KC_TOKEN_LIFETIME":   fmt.Sprintf("%d", kc.TokenLifetime),
		"KC_HOSTNAME":         kc.Hostname,
	}
}

func logtoEnvVars(cfg *config.Config) map[string]string {
	lt := cfg.Logto
	return map[string]string{
		"LOGTO_PORT":           fmt.Sprintf("%d", lt.Port),
		"LOGTO_ADMIN_PORT":     fmt.Sprintf("%d", lt.AdminPort),
		"LOGTO_DB_PORT":        fmt.Sprintf("%d", lt.DBPort),
		"LOGTO_ENDPOINT":       lt.Endpoint,
		"LOGTO_ADMIN_ENDPOINT": lt.AdminEndpoint,
	}
}

func ragEnvVars(cfg *config.Config, d derived) map[string]string {
	port := cfg.RagMCP.Port
	if port == 0 {
		port = 15080
	}
	vars := map[string]string{
		"DATABASE_URL": d.DatabaseURLContainer,
		"MCP_PORT":     fmt.Sprintf("%d", port),
	}
	if cfg.Secrets.AnthropicAPIKey != "" {
		vars["ANTHROPIC_API_KEY"] = cfg.Secrets.AnthropicAPIKey
	}
	// Pass AWS credential env vars through for Bedrock provider.
	// On IAM-role environments (EC2/ECS/EKS) these are empty and the SDK
	// resolves credentials via the instance metadata service instead.
	if cfg.RagMCP.HyDE.Provider == "bedrock" {
		for _, key := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
			if v := os.Getenv(key); v != "" {
				vars[key] = v
			}
		}
	}
	return vars
}

func docs2vectorEnvVars(cfg *config.Config, d derived) map[string]string {
	return map[string]string{
		"DATABASE_URL": d.DatabaseURLHost,
	}
}

// ─── TOML generators ─────────────────────────────────────────────────────────

func ragConfigTOML(cfg *config.Config, d derived) string {
	rag := cfg.RagMCP
	// The internal container port is always 15080; the host-side port is
	// controlled by MCP_PORT in the .env and the compose port mapping.
	const port = 15080
	logLevel := rag.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	var buf strings.Builder
	buf.WriteString("# Generated by stack — do not edit directly\n\n")

	fmt.Fprintf(&buf, "[embed]\nhost = %q\nmodel = %q\nembed_dim = %d\nquery_prefix = %q\npassage_prefix = %q\n\n",
		d.EmbedHost,
		cfg.Docs2Vector.EmbedModel,
		cfg.Docs2Vector.EmbedDim,
		cfg.Docs2Vector.QueryPrefix,
		cfg.Docs2Vector.PassagePrefix,
	)

	fmt.Fprintf(&buf, "[server]\nport = %q\nlog_level = %q\n\n",
		fmt.Sprintf("%d", port), logLevel)

	fmt.Fprintf(&buf, "[search]\nprobes = %d\nretrieval_pool_size = %d\nrrf_constant = %d\n\n",
		rag.Search.Probes, rag.Search.RetrievalPoolSize, rag.Search.RRFConstant)

	fmt.Fprintf(&buf, "[reranker]\nenabled = %v\nhost = %q\n\n",
		rag.Reranker.Enabled, d.RerankerHost)

	fmt.Fprintf(&buf, "[guardrails]\ncorpus_topic = %q\nmin_topic_score = %g\nmin_match_score = %g\n\n",
		rag.Guardrails.CorpusTopic, rag.Guardrails.MinTopicScore, rag.Guardrails.MinMatchScore)

	hydeProvider := rag.HyDE.Provider
	if hydeProvider == "" {
		hydeProvider = "anthropic"
	}
	fmt.Fprintf(&buf, "[hyde]\nenabled = %v\nprovider = %q\nmodel = %q\nbase_url = %q\naws_region = %q\nsystem_prompt = %q\n\n",
		rag.HyDE.Enabled, hydeProvider, rag.HyDE.Model, rag.HyDE.BaseURL, rag.HyDE.AWSRegion, rag.HyDE.SystemPrompt)

	fmt.Fprintf(&buf, "[auth]\njwks_url = %q\nissuer   = %q\naudience = %q\n",
		d.AuthJWKSURL, d.AuthIssuer, d.AuthAudience)

	return buf.String()
}

func docs2vectorConfigTOML(cfg *config.Config, d derived) string {
	dv := cfg.Docs2Vector
	chunkSize := dv.ChunkSize
	if chunkSize == 0 {
		chunkSize = 512
	}
	embedModel := dv.EmbedModel
	if embedModel == "" {
		embedModel = "mxbai-embed-large-v1"
	}

	var buf strings.Builder
	buf.WriteString("# Generated by stack — do not edit directly\n\n")

	fmt.Fprintf(&buf, "[embed]\nhost = %q\nmodel = %q\nembed_dim = %d\nquery_prefix = %q\npassage_prefix = %q\n\n",
		d.EmbedHost, embedModel, cfg.Docs2Vector.EmbedDim, dv.QueryPrefix, dv.PassagePrefix)

	fmt.Fprintf(&buf, "[ingest]\nchunk_size = %d\nmodel = %q\nembedding_model = %q\n",
		chunkSize, embedModel, embedModel)

	return buf.String()
}

// ─── Compose YAML templates ───────────────────────────────────────────────────

// postgresComposeYAML returns the YAML for the shared postgres service.
// Values are expressed as environment variable references — compose reads
// them from the sibling .stack/postgres.env file passed via --env-file.
func postgresComposeYAML() string {
	return `name: stack-postgres

networks:
  stack-net:
    external: true

volumes:
  stack-pgdata:

services:
  stack-postgres:
    image: ${POSTGRES_IMAGE}
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports:
      - "127.0.0.1:${POSTGRES_PORT}:5432"
    volumes:
      - stack-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - stack-net
    restart: unless-stopped
`
}

