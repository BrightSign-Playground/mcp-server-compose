// Package config loads and validates stack.toml.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the parsed, validated master configuration.
type Config struct {
	Runtime     RuntimeConfig
	Profiles    []string
	Postgres    PostgresConfig
	Llama       LlamaConfig
	Keycloak    KeycloakConfig
	Logto       LogtoConfig
	RagMCP      RagMCPConfig
	Docs2Vector Docs2VectorConfig
	Secrets     SecretsConfig
}

type RuntimeConfig struct {
	Engine string
}

type PostgresConfig struct {
	Image      string
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	DataVolume string
	DataDir    string
}

type LlamaConfig struct {
	Image      string
	HostPort   int
	ModelsDir  string
	EmbedModel string
	GenModel   string
	ExtraFlags string
}

type KeycloakConfig struct {
	Port            int
	DBPort          int
	AdminUser       string
	AdminPassword   string
	Realm           string
	APIClientID     string
	M2MClientID     string
	M2MClientSecret string
	TokenLifetime   int
	Hostname        string
}

type LogtoConfig struct {
	Port          int
	AdminPort     int
	DBPort        int
	Endpoint      string
	AdminEndpoint string
	AppID         string
	AppSecret     string
	Audience      string
	MgmtAppID     string
	MgmtAppSecret string
}

type RagMCPConfig struct {
	Port         int
	LogLevel     string
	AuthProvider string
	AuthJWKSURL  string
	AuthIssuer   string
	AuthAudience string
	Search       SearchConfig
	Reranker     RerankerConfig
	Guardrails   GuardrailsConfig
	HyDE         HyDEConfig
}

type SearchConfig struct {
	Probes            int
	RetrievalPoolSize int
	RRFConstant       int
}

type RerankerConfig struct {
	Enabled bool
	Host    string
}

type GuardrailsConfig struct {
	CorpusTopic   string
	MinTopicScore float64
	MinMatchScore float64
}

type HyDEConfig struct {
	Enabled      bool
	Model        string
	BaseURL      string
	SystemPrompt string
}

type Docs2VectorConfig struct {
	DocsDir       string
	ChunkSize     int
	EmbedModel    string
	EmbedDim      int // 0 = auto-resolve from EmbedModel via KnownEmbedDims
	QueryPrefix   string
	PassagePrefix string
}

// KnownEmbedDims maps known embedding model names (and their GGUF filename
// variants) to the output vector dimension. Used to auto-resolve embed_dim
// when it is not explicitly set in stack.toml.
var KnownEmbedDims = map[string]int{
	"nomic-embed-text-v1.5":           768,
	"nomic-embed-text-v1.5.Q8_0.gguf": 768,
	"mxbai-embed-large-v1":            1024,
	"mxbai-embed-large-v1-f16.gguf":   1024,
}

type SecretsConfig struct {
	AnthropicAPIKey string
}

// fileConfig mirrors the TOML structure with snake_case tags.
type fileConfig struct {
	Runtime  struct {
		Engine string `toml:"engine"`
	} `toml:"runtime"`
	Profiles []string `toml:"profiles"`
	Postgres struct {
		Image      string `toml:"image"`
		Host       string `toml:"host"`
		Port       int    `toml:"port"`
		User       string `toml:"user"`
		Password   string `toml:"password"`
		Database   string `toml:"database"`
		DataVolume string `toml:"data_volume"`
		DataDir    string `toml:"data_dir"`
	} `toml:"postgres"`
	Llama struct {
		Image      string `toml:"image"`
		HostPort   int    `toml:"host_port"`
		ModelsDir  string `toml:"models_dir"`
		EmbedModel string `toml:"embed_model"`
		GenModel   string `toml:"gen_model"`
		ExtraFlags string `toml:"extra_flags"`
	} `toml:"llama"`
	Keycloak struct {
		Port            int    `toml:"port"`
		DBPort          int    `toml:"db_port"`
		AdminUser       string `toml:"admin_user"`
		AdminPassword   string `toml:"admin_password"`
		Realm           string `toml:"realm"`
		APIClientID     string `toml:"api_client_id"`
		M2MClientID     string `toml:"m2m_client_id"`
		M2MClientSecret string `toml:"m2m_client_secret"`
		TokenLifetime   int    `toml:"token_lifetime"`
		Hostname        string `toml:"hostname"`
	} `toml:"keycloak"`
	Logto struct {
		Port          int    `toml:"port"`
		AdminPort     int    `toml:"admin_port"`
		DBPort        int    `toml:"db_port"`
		Endpoint      string `toml:"endpoint"`
		AdminEndpoint string `toml:"admin_endpoint"`
		AppID         string `toml:"app_id"`
		AppSecret     string `toml:"app_secret"`
		Audience      string `toml:"audience"`
		MgmtAppID     string `toml:"mgmt_app_id"`
		MgmtAppSecret string `toml:"mgmt_app_secret"`
	} `toml:"logto"`
	RagMCPServer struct {
		Port         int    `toml:"port"`
		LogLevel     string `toml:"log_level"`
		AuthProvider string `toml:"auth_provider"`
		AuthJWKSURL  string `toml:"auth_jwks_url"`
		AuthIssuer   string `toml:"auth_issuer"`
		AuthAudience string `toml:"auth_audience"`
		Search       struct {
			Probes            int `toml:"probes"`
			RetrievalPoolSize int `toml:"retrieval_pool_size"`
			RRFConstant       int `toml:"rrf_constant"`
		} `toml:"search"`
		Reranker struct {
			Enabled bool   `toml:"enabled"`
			Host    string `toml:"host"`
		} `toml:"reranker"`
		Guardrails struct {
			CorpusTopic   string  `toml:"corpus_topic"`
			MinTopicScore float64 `toml:"min_topic_score"`
			MinMatchScore float64 `toml:"min_match_score"`
		} `toml:"guardrails"`
		HyDE struct {
			Enabled      bool   `toml:"enabled"`
			Model        string `toml:"model"`
			BaseURL      string `toml:"base_url"`
			SystemPrompt string `toml:"system_prompt"`
		} `toml:"hyde"`
	} `toml:"rag_mcp_server"`
	Docs2Vector struct {
		DocsDir       string `toml:"docs_dir"`
		ChunkSize     int    `toml:"chunk_size"`
		EmbedModel    string `toml:"embed_model"`
		EmbedDim      int    `toml:"embed_dim"`
		QueryPrefix   string `toml:"query_prefix"`
		PassagePrefix string `toml:"passage_prefix"`
	} `toml:"docs2vector"`
}

// Load reads and parses the stack.toml file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var fc fileConfig
	if _, err := toml.Decode(string(data), &fc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	cfg := &Config{
		Runtime:  RuntimeConfig{Engine: fc.Runtime.Engine},
		Profiles: fc.Profiles,
		Postgres: PostgresConfig{
			Image:      fc.Postgres.Image,
			Host:       fc.Postgres.Host,
			Port:       fc.Postgres.Port,
			User:       fc.Postgres.User,
			Password:   fc.Postgres.Password,
			Database:   fc.Postgres.Database,
			DataVolume: fc.Postgres.DataVolume,
			DataDir:    fc.Postgres.DataDir,
		},
		Llama: LlamaConfig{
			Image:      fc.Llama.Image,
			HostPort:   fc.Llama.HostPort,
			ModelsDir:  fc.Llama.ModelsDir,
			EmbedModel: fc.Llama.EmbedModel,
			GenModel:   fc.Llama.GenModel,
			ExtraFlags: fc.Llama.ExtraFlags,
		},
		Keycloak: KeycloakConfig{
			Port:            fc.Keycloak.Port,
			DBPort:          fc.Keycloak.DBPort,
			AdminUser:       fc.Keycloak.AdminUser,
			AdminPassword:   fc.Keycloak.AdminPassword,
			Realm:           fc.Keycloak.Realm,
			APIClientID:     fc.Keycloak.APIClientID,
			M2MClientID:     fc.Keycloak.M2MClientID,
			M2MClientSecret: fc.Keycloak.M2MClientSecret,
			TokenLifetime:   fc.Keycloak.TokenLifetime,
			Hostname:        fc.Keycloak.Hostname,
		},
		Logto: LogtoConfig{
			Port:          fc.Logto.Port,
			AdminPort:     fc.Logto.AdminPort,
			DBPort:        fc.Logto.DBPort,
			Endpoint:      fc.Logto.Endpoint,
			AdminEndpoint: fc.Logto.AdminEndpoint,
			AppID:         fc.Logto.AppID,
			AppSecret:     fc.Logto.AppSecret,
			Audience:      fc.Logto.Audience,
			MgmtAppID:     fc.Logto.MgmtAppID,
			MgmtAppSecret: fc.Logto.MgmtAppSecret,
		},
		RagMCP: RagMCPConfig{
			Port:         fc.RagMCPServer.Port,
			LogLevel:     fc.RagMCPServer.LogLevel,
			AuthProvider: fc.RagMCPServer.AuthProvider,
			AuthJWKSURL:  fc.RagMCPServer.AuthJWKSURL,
			AuthIssuer:   fc.RagMCPServer.AuthIssuer,
			AuthAudience: fc.RagMCPServer.AuthAudience,
			Search: SearchConfig{
				Probes:            fc.RagMCPServer.Search.Probes,
				RetrievalPoolSize: fc.RagMCPServer.Search.RetrievalPoolSize,
				RRFConstant:       fc.RagMCPServer.Search.RRFConstant,
			},
			Reranker: RerankerConfig{
				Enabled: fc.RagMCPServer.Reranker.Enabled,
				Host:    fc.RagMCPServer.Reranker.Host,
			},
			Guardrails: GuardrailsConfig{
				CorpusTopic:   fc.RagMCPServer.Guardrails.CorpusTopic,
				MinTopicScore: fc.RagMCPServer.Guardrails.MinTopicScore,
				MinMatchScore: fc.RagMCPServer.Guardrails.MinMatchScore,
			},
			HyDE: HyDEConfig{
				Enabled:      fc.RagMCPServer.HyDE.Enabled,
				Model:        fc.RagMCPServer.HyDE.Model,
				BaseURL:      fc.RagMCPServer.HyDE.BaseURL,
				SystemPrompt: fc.RagMCPServer.HyDE.SystemPrompt,
			},
		},
		Docs2Vector: Docs2VectorConfig{
			DocsDir:       fc.Docs2Vector.DocsDir,
			ChunkSize:     fc.Docs2Vector.ChunkSize,
			EmbedModel:    fc.Docs2Vector.EmbedModel,
			EmbedDim:      fc.Docs2Vector.EmbedDim,
			QueryPrefix:   fc.Docs2Vector.QueryPrefix,
			PassagePrefix: fc.Docs2Vector.PassagePrefix,
		},
		Secrets: SecretsConfig{
			AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		},
	}

	return cfg, nil
}

// Validate checks that cfg satisfies all requirements. Returns the first error
// found. Warnings (non-fatal issues) are printed to os.Stderr.
func Validate(cfg *Config) error {
	// Rule 1: keycloak and logto are mutually exclusive.
	if cfg.HasProfile("keycloak") && cfg.HasProfile("logto") {
		return errors.New("profiles: keycloak and logto are mutually exclusive")
	}

	// Rule 6: auth_provider must be one of the known values.
	switch cfg.RagMCP.AuthProvider {
	case "keycloak", "logto":
	default:
		return fmt.Errorf("rag_mcp_server.auth_provider: must be \"keycloak\" or \"logto\", got %q",
			cfg.RagMCP.AuthProvider)
	}

	// Rule 2: if postgres inactive, external connection fields must be set.
	if !cfg.PostgresActive() {
		if cfg.Postgres.Host == "" {
			return errors.New("postgres.host: required when postgres profile is inactive")
		}
		if cfg.Postgres.Port == 0 {
			return errors.New("postgres.port: required when postgres profile is inactive")
		}
		if cfg.Postgres.User == "" {
			return errors.New("postgres.user: required when postgres profile is inactive")
		}
		if cfg.Postgres.Password == "" {
			return errors.New("postgres.password: required when postgres profile is inactive")
		}
		if cfg.Postgres.Database == "" {
			return errors.New("postgres.database: required when postgres profile is inactive")
		}
	}

	// Rule 3: if llama active, models_dir and embed_model must be set.
	if cfg.LlamaActive() {
		if cfg.Llama.ModelsDir == "" {
			return errors.New("llama.models_dir: required when llama profile is active")
		}
		if cfg.Llama.EmbedModel == "" {
			return errors.New("llama.embed_model: required when llama profile is active")
		}
		info, err := os.Stat(cfg.Llama.ModelsDir)
		if err != nil {
			return fmt.Errorf("llama.models_dir: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("llama.models_dir: %q is not a directory", cfg.Llama.ModelsDir)
		}
	}

	// Rule 4: hyde requires anthropic_api_key.
	if cfg.RagMCP.HyDE.Enabled && cfg.Secrets.AnthropicAPIKey == "" {
		return errors.New("secrets.anthropic_api_key: required when rag_mcp_server.hyde.enabled is true")
	}

	// Rule 5: if auth_provider profile is not active, override fields must be set.
	providerActive := cfg.HasProfile(cfg.RagMCP.AuthProvider)
	if !providerActive {
		if cfg.RagMCP.AuthJWKSURL == "" {
			return errors.New("rag_mcp_server.auth_jwks_url: required when auth_provider profile is not active")
		}
		if cfg.RagMCP.AuthIssuer == "" {
			return errors.New("rag_mcp_server.auth_issuer: required when auth_provider profile is not active")
		}
		if cfg.RagMCP.AuthAudience == "" {
			return errors.New("rag_mcp_server.auth_audience: required when auth_provider profile is not active")
		}
	}

	// Rule 8: extra_flags must not contain shell metacharacters.
	if err := validateExtraFlags(cfg.Llama.ExtraFlags); err != nil {
		return fmt.Errorf("llama.extra_flags: %w", err)
	}

	// Port range checks: validate any explicitly set port value.
	portChecks := []struct {
		name  string
		value int
	}{
		{"postgres.port", cfg.Postgres.Port},
		{"llama.host_port", cfg.Llama.HostPort},
		{"keycloak.port", cfg.Keycloak.Port},
		{"keycloak.db_port", cfg.Keycloak.DBPort},
		{"logto.port", cfg.Logto.Port},
		{"logto.admin_port", cfg.Logto.AdminPort},
		{"logto.db_port", cfg.Logto.DBPort},
		{"rag_mcp_server.port", cfg.RagMCP.Port},
	}
	for _, pc := range portChecks {
		if err := validatePort(pc.name, pc.value); err != nil {
			return err
		}
	}

	// Rule 9: resolve embed_dim from model name if not explicitly set.
	if cfg.Docs2Vector.EmbedDim == 0 {
		dim, ok := KnownEmbedDims[cfg.Docs2Vector.EmbedModel]
		if !ok {
			return fmt.Errorf("unknown embedding model %q — set docs2vector.embed_dim explicitly", cfg.Docs2Vector.EmbedModel)
		}
		cfg.Docs2Vector.EmbedDim = dim
	}
	if cfg.Docs2Vector.EmbedDim < 1 {
		return fmt.Errorf("docs2vector.embed_dim must be a positive integer, got %d", cfg.Docs2Vector.EmbedDim)
	}

	// Rule 7 (warnings only).
	if cfg.Keycloak.M2MClientSecret == "changeme-dev-secret" {
		fmt.Fprintln(os.Stderr, "warning: keycloak.m2m_client_secret is the default placeholder; change it for shared environments")
	}
	if cfg.Postgres.Password == "changeme" {
		fmt.Fprintln(os.Stderr, "warning: postgres.password is the default placeholder; change it for shared environments")
	}

	return nil
}

// HasProfile reports whether name is in cfg.Profiles.
func (cfg *Config) HasProfile(name string) bool {
	for _, p := range cfg.Profiles {
		if p == name {
			return true
		}
	}
	return false
}

// PostgresActive reports whether the postgres profile is enabled.
func (cfg *Config) PostgresActive() bool { return cfg.HasProfile("postgres") }

// LlamaActive reports whether the llama profile is enabled.
func (cfg *Config) LlamaActive() bool { return cfg.HasProfile("llama") }

// KeycloakActive reports whether the keycloak profile is enabled.
func (cfg *Config) KeycloakActive() bool { return cfg.HasProfile("keycloak") }

// LogtoActive reports whether the logto profile is enabled.
func (cfg *Config) LogtoActive() bool { return cfg.HasProfile("logto") }

// shellMetachars are characters that must not appear in extra_flags.
const shellMetachars = ";|`$><&"

func validateExtraFlags(flags string) error {
	if strings.ContainsAny(flags, shellMetachars) {
		return errors.New("shell metacharacters are not allowed")
	}
	return nil
}

// validatePort returns an error if the port is set (non-zero) and out of range.
func validatePort(name string, value int) error {
	if value != 0 && (value < 1 || value > 65535) {
		return fmt.Errorf("%s: port %d is out of range (1–65535)", name, value)
	}
	return nil
}
