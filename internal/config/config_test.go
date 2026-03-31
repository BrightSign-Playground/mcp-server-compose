package config

import (
	"os"
	"path/filepath"
	"testing"
)

const validTOML = `
profiles = ["postgres", "keycloak"]

[runtime]
engine = "podman"

[postgres]
image    = "docker.io/pgvector/pgvector:pg17"
host     = "localhost"
port     = 5432
user     = "support"
password = "secret"
database = "support"
data_volume = "stack-pgdata"

[llama]
host_port   = 16000

[keycloak]
port             = 8080
db_port          = 5434
admin_user       = "admin"
admin_password   = "admin"
realm            = "dev"
api_client_id    = "my-api"
m2m_client_id    = "my-app"
m2m_client_secret = "real-secret"
token_lifetime   = 3600
hostname         = "localhost"

[logto]
port           = 3001
admin_port     = 3002
db_port        = 5435
endpoint       = "http://localhost:3001"
admin_endpoint = "http://localhost:3002"
audience       = "https://my-service"

[rag_mcp_server]
port          = 15080
log_level     = "info"
auth_provider = "keycloak"

[rag_mcp_server.search]
probes              = 4
retrieval_pool_size = 20
rrf_constant        = 60

[rag_mcp_server.reranker]
enabled = false
host    = "http://localhost:8081"

[rag_mcp_server.guardrails]
corpus_topic    = ""
min_topic_score = 0.25
min_match_score = 0.0

[rag_mcp_server.hyde]
enabled  = false
provider = "anthropic"
model    = "claude-haiku-4-5-20251001"

[docs2vector]
docs_dir   = "/docs"
chunk_size = 512
embed_model = "mxbai-embed-large-v1"

`

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.toml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing temp TOML: %v", err)
	}
	return path
}

func TestLoad_valid(t *testing.T) {
	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Runtime.Engine != "podman" {
		t.Errorf("runtime.engine: got %q, want %q", cfg.Runtime.Engine, "podman")
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("profiles length: got %d, want 2", len(cfg.Profiles))
	}
	if cfg.Postgres.Port != 5432 {
		t.Errorf("postgres.port: got %d, want 5432", cfg.Postgres.Port)
	}
	if cfg.Keycloak.Realm != "dev" {
		t.Errorf("keycloak.realm: got %q, want %q", cfg.Keycloak.Realm, "dev")
	}
	if cfg.RagMCP.Search.Probes != 4 {
		t.Errorf("rag_mcp_server.search.probes: got %d, want 4", cfg.RagMCP.Search.Probes)
	}
}

func TestLoad_notFound(t *testing.T) {
	_, err := Load("/nonexistent/stack.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_invalidTOML(t *testing.T) {
	path := writeTempTOML(t, "not valid toml ][")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestValidate_valid(t *testing.T) {
	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_mutualExclusion(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"keycloak", "logto"},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for keycloak+logto in profiles")
	}
}

func TestValidate_authProviderInvalid(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"postgres"},
		RagMCP:   RagMCPConfig{AuthProvider: "okta"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for invalid auth_provider")
	}
}

func TestValidate_postgresInactiveFieldsMissing(t *testing.T) {
	cfg := &Config{
		Profiles: []string{},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak", AuthJWKSURL: "x", AuthIssuer: "x", AuthAudience: "x"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error when postgres inactive and host not set")
	}
}

func TestValidate_postgresInactiveAllSet(t *testing.T) {
	cfg := &Config{
		Profiles:    []string{},
		Postgres:    PostgresConfig{Host: "db.local", Port: 5432, User: "u", Password: "p", Database: "d"},
		RagMCP:      RagMCPConfig{AuthProvider: "keycloak", AuthJWKSURL: "x", AuthIssuer: "x", AuthAudience: "x"},
		Docs2Vector: Docs2VectorConfig{EmbedModel: "mxbai-embed-large-v1"},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_hydeRequiresKey(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"postgres", "keycloak"},
		Postgres: PostgresConfig{Host: "h", Port: 1, User: "u", Password: "p", Database: "d"},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak"},
	}
	cfg.RagMCP.HyDE.Enabled = true
	cfg.RagMCP.HyDE.Provider = "anthropic"
	cfg.Secrets.AnthropicAPIKey = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error when hyde enabled without api key")
	}
}

func TestValidate_hydeBedrockNoKeyRequired(t *testing.T) {
	cfg := &Config{
		Profiles:    []string{"postgres", "keycloak"},
		Postgres:    PostgresConfig{Host: "h", Port: 1, User: "u", Password: "p", Database: "d"},
		Docs2Vector: Docs2VectorConfig{EmbedModel: "mxbai-embed-large-v1"},
		RagMCP:      RagMCPConfig{AuthProvider: "keycloak"},
	}
	cfg.RagMCP.HyDE.Enabled = true
	cfg.RagMCP.HyDE.Provider = "bedrock"
	cfg.RagMCP.HyDE.AWSRegion = "us-east-1"
	cfg.Secrets.AnthropicAPIKey = ""
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error for bedrock without api key: %v", err)
	}
}

func TestValidate_hydeBedrockRequiresRegion(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"postgres", "keycloak"},
		Postgres: PostgresConfig{Host: "h", Port: 1, User: "u", Password: "p", Database: "d"},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak"},
	}
	cfg.RagMCP.HyDE.Enabled = true
	cfg.RagMCP.HyDE.Provider = "bedrock"
	cfg.RagMCP.HyDE.AWSRegion = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error when bedrock has no aws_region")
	}
}

func TestValidate_authProviderNotInProfiles(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"postgres"},
		Postgres: PostgresConfig{Host: "h", Port: 1, User: "u", Password: "p", Database: "d"},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak"},
	}
	// keycloak not in profiles and no override fields
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error when auth_provider not in profiles and no overrides")
	}
}

func TestValidate_authProviderOverridesProvided(t *testing.T) {
	cfg := &Config{
		Profiles:    []string{"postgres"},
		Postgres:    PostgresConfig{Host: "h", Port: 1, User: "u", Password: "p", Database: "d"},
		Docs2Vector: Docs2VectorConfig{EmbedModel: "mxbai-embed-large-v1"},
		RagMCP: RagMCPConfig{
			AuthProvider: "keycloak",
			AuthJWKSURL:  "https://kc/jwks",
			AuthIssuer:   "https://kc",
			AuthAudience: "my-api",
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_portOutOfRange(t *testing.T) {
	cfg := &Config{
		Profiles: []string{"postgres"},
		Postgres: PostgresConfig{Host: "h", Port: 99999, User: "u", Password: "p", Database: "d"},
		RagMCP:   RagMCPConfig{AuthProvider: "keycloak", AuthJWKSURL: "x", AuthIssuer: "x", AuthAudience: "x"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for out-of-range port")
	}
}

func TestHasProfile(t *testing.T) {
	cfg := &Config{Profiles: []string{"postgres", "keycloak"}}
	if !cfg.HasProfile("postgres") {
		t.Error("expected postgres to be present")
	}
	if cfg.HasProfile("logto") {
		t.Error("expected logto to be absent")
	}
}
