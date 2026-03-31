// Package generate derives configuration values and writes all ephemeral files.
package generate

import (
	"fmt"
	"net/url"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
)

// derived holds all computed values produced from the master config.
type derived struct {
	// DATABASE_URL for containers joining stack-net (uses service name as host).
	DatabaseURLContainer string
	// DATABASE_URL for the host (used by docs2vector run on host or in ingest container).
	DatabaseURLHost string

	AuthJWKSURL  string
	AuthIssuer   string
	AuthAudience string

	// embed.host value for rag-mcp-server and docs2vector config.toml.
	EmbedHost string
}

// deriveAll computes all derived values from cfg and eng.
func deriveAll(cfg *config.Config, eng engine.Engine) derived {
	d := derived{}

	d.DatabaseURLContainer = databaseURLContainer(cfg)
	d.DatabaseURLHost = databaseURLHost(cfg)
	d.AuthJWKSURL, d.AuthIssuer, d.AuthAudience = authFields(cfg)
	d.EmbedHost = embedHost(cfg, eng)

	return d
}

// databaseURLContainer builds the DATABASE_URL for containers on stack-net.
// When the postgres profile is active, the hostname is the compose service name.
func databaseURLContainer(cfg *config.Config) string {
	if cfg.PostgresActive() {
		u := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.Postgres.User, cfg.Postgres.Password),
			Host:     "stack-postgres:5432",
			Path:     "/" + cfg.Postgres.Database,
			RawQuery: "sslmode=disable",
		}
		return u.String()
	}
	return databaseURLHost(cfg)
}

// databaseURLHost builds the DATABASE_URL using the configured host and port.
func databaseURLHost(cfg *config.Config) string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.Postgres.User, cfg.Postgres.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Postgres.Host, cfg.Postgres.Port),
		Path:     "/" + cfg.Postgres.Database,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

// authFields derives jwks_url, issuer, and audience for the active auth provider.
// If the auth provider profile is not active, override fields from RagMCP are used.
func authFields(cfg *config.Config) (jwksURL, issuer, audience string) {
	switch cfg.RagMCP.AuthProvider {
	case "keycloak":
		if cfg.KeycloakActive() {
			// jwks_url uses the compose service name (container-to-container).
			// issuer uses the configured hostname (what JWT iss claim contains).
			kc := cfg.Keycloak
			jwksURL = fmt.Sprintf("http://keycloak:8080/realms/%s/protocol/openid-connect/certs", kc.Realm)
			issuer = fmt.Sprintf("http://%s:%d/realms/%s", kc.Hostname, kc.Port, kc.Realm)
			audience = kc.APIClientID
			return
		}
	case "logto":
		if cfg.LogtoActive() {
			lt := cfg.Logto
			jwksURL = fmt.Sprintf("http://logto:%d/oidc/jwks", lt.Port)
			// Extract host from endpoint for the issuer.
			issuerHost := logtoIssuerHost(lt.Endpoint, lt.Port)
			issuer = issuerHost + "/oidc"
			audience = lt.Audience
			return
		}
	}

	// Fall back to override fields (external provider).
	jwksURL = cfg.RagMCP.AuthJWKSURL
	issuer = cfg.RagMCP.AuthIssuer
	audience = cfg.RagMCP.AuthAudience
	return
}

// logtoIssuerHost derives the base URL for the Logto issuer from the endpoint
// field. Returns endpoint as-is if parsing fails.
func logtoIssuerHost(endpoint string, port int) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return endpoint
	}
	// Rebuild with only scheme and host (strip any path).
	return fmt.Sprintf("%s://%s:%d", parsed.Scheme, parsed.Hostname(), port)
}

// embedHost returns the value for [embed].host in rag-mcp-server and docs2vector
// config.toml. llama-server runs on the host, so containers reach it via the
// engine-specific host-gateway address.
func embedHost(cfg *config.Config, eng engine.Engine) string {
	port := cfg.Llama.HostPort
	if port == 0 {
		port = 16000 // default
	}
	if eng.IsPodman() {
		return fmt.Sprintf("http://host.containers.internal:%d", port)
	}
	// Docker uses host-gateway (requires extra_hosts in compose).
	return fmt.Sprintf("http://host-gateway:%d", port)
}
