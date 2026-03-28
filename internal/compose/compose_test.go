package compose

import (
	"testing"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
)

func makeConfig(profiles ...string) *config.Config {
	return &config.Config{
		Profiles: profiles,
		Postgres: config.PostgresConfig{Host: "localhost", Port: 5432, User: "u", Password: "p", Database: "d"},
		RagMCP:   config.RagMCPConfig{AuthProvider: "keycloak"},
	}
}

func podmanEng() engine.Engine {
	return engine.NewForTest(engine.KindPodman, "podman")
}

func TestActiveProjects_alwaysIncludesRag(t *testing.T) {
	cfg := makeConfig()
	projects := activeProjects(cfg, "/repo")
	if len(projects) == 0 {
		t.Fatal("expected at least one project (stack-rag)")
	}
	last := projects[len(projects)-1]
	if last.name != "stack-rag" {
		t.Errorf("last project should be stack-rag, got %q", last.name)
	}
}

func TestActiveProjects_postgresAdded(t *testing.T) {
	cfg := makeConfig("postgres")
	projects := activeProjects(cfg, "/repo")
	if projects[0].name != "stack-postgres" {
		t.Errorf("expected stack-postgres first, got %q", projects[0].name)
	}
}

func TestActiveProjects_keycloakAdded(t *testing.T) {
	cfg := makeConfig("postgres", "keycloak")
	projects := activeProjects(cfg, "/repo")
	found := false
	for _, p := range projects {
		if p.name == "stack-keycloak" {
			found = true
			if p.profile != "keycloak" {
				t.Errorf("expected profile=keycloak, got %q", p.profile)
			}
		}
	}
	if !found {
		t.Error("expected stack-keycloak in active projects")
	}
}

func TestActiveProjects_logtoAdded(t *testing.T) {
	cfg := makeConfig("logto")
	projects := activeProjects(cfg, "/repo")
	found := false
	for _, p := range projects {
		if p.name == "stack-logto" {
			found = true
		}
	}
	if !found {
		t.Error("expected stack-logto in active projects")
	}
}

func TestActiveProjects_llamaAdded(t *testing.T) {
	cfg := makeConfig("llama")
	projects := activeProjects(cfg, "/repo")
	found := false
	for _, p := range projects {
		if p.name == "stack-llama" {
			found = true
		}
	}
	if !found {
		t.Error("expected stack-llama in active projects")
	}
}

func TestActiveProjects_ragIsLast(t *testing.T) {
	cfg := makeConfig("postgres", "keycloak", "llama")
	projects := activeProjects(cfg, "/repo")
	last := projects[len(projects)-1]
	if last.name != "stack-rag" {
		t.Errorf("expected rag last, got %q", last.name)
	}
}

func TestMatchesComponent(t *testing.T) {
	tests := []struct {
		projectName string
		component   string
		want        bool
	}{
		{"stack-rag", "rag", true},
		{"stack-postgres", "postgres", true},
		{"stack-keycloak", "keycloak", true},
		{"stack-rag", "postgres", false},
		{"stack-rag", "stack-rag", true},
	}
	for _, tc := range tests {
		got := matchesComponent(tc.projectName, tc.component)
		if got != tc.want {
			t.Errorf("matchesComponent(%q, %q) = %v, want %v", tc.projectName, tc.component, got, tc.want)
		}
	}
}

func TestProjectNames(t *testing.T) {
	_ = podmanEng() // ensure engine package is linked
	projects := []project{
		{name: "stack-postgres"},
		{name: "stack-rag"},
	}
	names := projectNames(projects)
	if names != "postgres, rag" {
		t.Errorf("projectNames: got %q", names)
	}
}
