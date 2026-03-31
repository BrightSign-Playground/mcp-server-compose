package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
)

func podmanEngine() engine.Engine {
	eng, _ := engine.Detect("podman", "podman")
	// Fallback: construct manually if podman not on PATH.
	if eng.Kind == "" {
		return engine.Engine{}
	}
	return eng
}

func makePodmanEngine() engine.Engine {
	// Construct a podman engine for testing without needing the binary on PATH.
	return buildTestEngine(engine.KindPodman, "podman")
}

func makeDockerEngine() engine.Engine {
	return buildTestEngine(engine.KindDocker, "docker")
}

// buildTestEngine builds an Engine value for testing using exported fields.
// We use the exported IsPodman() method to verify behavior.
func buildTestEngine(kind engine.Kind, binary string) engine.Engine {
	// Use reflect-free construction via a wrapper function in the engine package.
	// Since Engine has unexported fields, we test via Detect with a mock.
	return engine.NewForTest(kind, binary)
}

func baseConfig() *config.Config {
	return &config.Config{
		Profiles: []string{"postgres", "keycloak"},
		Postgres: config.PostgresConfig{
			Image:    "docker.io/pgvector/pgvector:pg17",
			Host:     "localhost",
			Port:     5432,
			User:     "support",
			Password: "secret",
			Database: "support",
		},
		Llama: config.LlamaConfig{
			HostPort: 16000,
		},
		Keycloak: config.KeycloakConfig{
			Port:            8080,
			DBPort:          5434,
			AdminUser:       "admin",
			AdminPassword:   "admin",
			Realm:           "dev",
			APIClientID:     "my-api",
			M2MClientID:     "my-app",
			M2MClientSecret: "secret",
			TokenLifetime:   3600,
			Hostname:        "localhost",
		},
		Logto: config.LogtoConfig{
			Port:          3001,
			AdminPort:     3002,
			DBPort:        5435,
			Endpoint:      "http://localhost:3001",
			AdminEndpoint: "http://localhost:3002",
			Audience:      "https://my-service",
		},
		RagMCP: config.RagMCPConfig{
			Port:         15080,
			LogLevel:     "info",
			AuthProvider: "keycloak",
			Search: config.SearchConfig{
				Probes:            4,
				RetrievalPoolSize: 20,
				RRFConstant:       60,
			},
			Reranker:   config.RerankerConfig{Host: "http://localhost:8081"},
			Guardrails: config.GuardrailsConfig{MinTopicScore: 0.25},
			HyDE:       config.HyDEConfig{Model: "claude-haiku-4-5-20251001"},
		},
		Docs2Vector: config.Docs2VectorConfig{
			DocsDir:    "/docs",
			ChunkSize:  512,
			EmbedModel: "mxbai-embed-large-v1",
		},
	}
}

// ─── derive tests ─────────────────────────────────────────────────────────────

func TestDeriveAll_postgresActive_containerURL(t *testing.T) {
	cfg := baseConfig()
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)
	want := "postgres://support:secret@stack-postgres:5432/support?sslmode=disable"
	if d.DatabaseURLContainer != want {
		t.Errorf("DatabaseURLContainer:\n got  %q\n want %q", d.DatabaseURLContainer, want)
	}
}

func TestDeriveAll_postgresInactive_usesHostConfig(t *testing.T) {
	cfg := baseConfig()
	cfg.Profiles = []string{} // no postgres
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)
	if !strings.Contains(d.DatabaseURLContainer, "localhost:5432") {
		t.Errorf("expected host URL, got %q", d.DatabaseURLContainer)
	}
}

func TestDeriveAll_keycloakAuth(t *testing.T) {
	cfg := baseConfig()
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)

	wantJWKS := "http://keycloak:8080/realms/dev/protocol/openid-connect/certs"
	wantIssuer := "http://localhost:8080/realms/dev"
	wantAudience := "my-api"

	if d.AuthJWKSURL != wantJWKS {
		t.Errorf("AuthJWKSURL:\n got  %q\n want %q", d.AuthJWKSURL, wantJWKS)
	}
	if d.AuthIssuer != wantIssuer {
		t.Errorf("AuthIssuer:\n got  %q\n want %q", d.AuthIssuer, wantIssuer)
	}
	if d.AuthAudience != wantAudience {
		t.Errorf("AuthAudience:\n got  %q\n want %q", d.AuthAudience, wantAudience)
	}
}

func TestDeriveAll_logtoAuth(t *testing.T) {
	cfg := baseConfig()
	cfg.Profiles = []string{"postgres", "logto"}
	cfg.RagMCP.AuthProvider = "logto"
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)

	if !strings.Contains(d.AuthJWKSURL, "logto:3001") {
		t.Errorf("logto AuthJWKSURL should use service name, got %q", d.AuthJWKSURL)
	}
	if !strings.Contains(d.AuthIssuer, "/oidc") {
		t.Errorf("logto AuthIssuer should end in /oidc, got %q", d.AuthIssuer)
	}
}

func TestDeriveAll_embedHost_podman(t *testing.T) {
	cfg := baseConfig()
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)
	want := "http://host.containers.internal:16000"
	if d.EmbedHost != want {
		t.Errorf("EmbedHost (podman): got %q want %q", d.EmbedHost, want)
	}
}

func TestDeriveAll_embedHost_docker(t *testing.T) {
	cfg := baseConfig()
	eng := makeDockerEngine()
	d := deriveAll(cfg, eng)
	want := "http://host-gateway:16000"
	if d.EmbedHost != want {
		t.Errorf("EmbedHost (docker): got %q want %q", d.EmbedHost, want)
	}
}

func TestDeriveAll_urlEncodesCredentials(t *testing.T) {
	cfg := baseConfig()
	cfg.Postgres.User = "user@domain"
	cfg.Postgres.Password = "p@ss:w/ord"
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)

	// Neither the literal @ nor : from the password should appear unencoded in userinfo.
	// The URL should be parseable and credentials round-trip correctly.
	for _, rawURL := range []string{d.DatabaseURLContainer, d.DatabaseURLHost} {
		if strings.Contains(rawURL, "p@ss:w/ord") {
			t.Errorf("unencoded password in URL: %q", rawURL)
		}
	}
}

func TestDeriveAll_authOverride_whenProviderNotInProfiles(t *testing.T) {
	cfg := baseConfig()
	cfg.Profiles = []string{"postgres"}
	cfg.RagMCP.AuthProvider = "keycloak"
	cfg.RagMCP.AuthJWKSURL = "https://ext/jwks"
	cfg.RagMCP.AuthIssuer = "https://ext"
	cfg.RagMCP.AuthAudience = "my-api"
	eng := makePodmanEngine()
	d := deriveAll(cfg, eng)
	if d.AuthJWKSURL != "https://ext/jwks" {
		t.Errorf("expected override JWKS URL, got %q", d.AuthJWKSURL)
	}
}

// ─── writeEnvFile tests ───────────────────────────────────────────────────────

func TestWriteEnvFile_roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.env")
	vars := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}
	if err := writeEnvFile(path, vars); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "BAZ=qux") {
		t.Errorf("expected BAZ=qux in output, got:\n%s", content)
	}
	if !strings.Contains(content, "FOO=bar") {
		t.Errorf("expected FOO=bar in output, got:\n%s", content)
	}
}

func TestWriteEnvFile_rejectsNewlines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.env")
	vars := map[string]string{"KEY": "val\ninjected"}
	if err := writeEnvFile(path, vars); err == nil {
		t.Fatal("expected error for newline in env value")
	}
}

func TestWriteEnvFile_permissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.env")
	if err := writeEnvFile(path, map[string]string{"K": "v"}); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions: got %o, want 0600", info.Mode().Perm())
	}
}

// ─── compose YAML tests ───────────────────────────────────────────────────────

func TestPostgresComposeYAML_containsServiceName(t *testing.T) {
	yaml := postgresComposeYAML()
	if !strings.Contains(yaml, "stack-postgres:") {
		t.Errorf("expected service name 'stack-postgres' in compose YAML")
	}
	if !strings.Contains(yaml, "stack-net") {
		t.Errorf("expected stack-net network in compose YAML")
	}
	if !strings.Contains(yaml, "127.0.0.1") {
		t.Errorf("expected loopback port binding in compose YAML")
	}
}

// ─── config TOML generation tests ────────────────────────────────────────────

func TestRagConfigTOML_containsAllSections(t *testing.T) {
	cfg := baseConfig()
	d := deriveAll(cfg, makePodmanEngine())
	toml := ragConfigTOML(cfg, d)

	sections := []string{"[embed]", "[server]", "[search]", "[reranker]", "[guardrails]", "[hyde]", "[auth]"}
	for _, section := range sections {
		if !strings.Contains(toml, section) {
			t.Errorf("missing section %q in rag config TOML", section)
		}
	}
}

func TestDocs2VectorConfigTOML_containsEmbedSection(t *testing.T) {
	cfg := baseConfig()
	d := deriveAll(cfg, makePodmanEngine())
	toml := docs2vectorConfigTOML(cfg, d)
	if !strings.Contains(toml, "[embed]") {
		t.Errorf("missing [embed] section in docs2vector TOML")
	}
	if !strings.Contains(toml, "[ingest]") {
		t.Errorf("missing [ingest] section in docs2vector TOML")
	}
}
