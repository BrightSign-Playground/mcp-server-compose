// Package compose orchestrates container lifecycle via podman/docker compose.
// Each component runs as a separate compose project to avoid service name
// collisions. All projects share the external stack-net network.
package compose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
	"github.com/brightsign-playground/stack/internal/generate"
)

const (
	sharedNetwork = "stack-net"
	healthTimeout = 180 * time.Second
	healthPoll    = 5 * time.Second
)

// project describes a single compose project.
type project struct {
	name         string
	envFile      string
	composeFiles []string
	profile      string // compose --profile flag; empty if not needed
}

// activeProjects returns the ordered list of compose projects to manage
// based on the active profiles. rag-mcp-server is always last.
func activeProjects(cfg *config.Config, repoRoot string) []project {
	stackDir := filepath.Join(repoRoot, ".stack")
	var projects []project

	if cfg.PostgresActive() {
		projects = append(projects, project{
			name:         "stack-postgres",
			envFile:      filepath.Join(stackDir, "postgres.env"),
			composeFiles: []string{filepath.Join(stackDir, "compose.postgres.yml")},
		})
	}

	if cfg.KeycloakActive() {
		projects = append(projects, project{
			name:         "stack-keycloak",
			envFile:      filepath.Join(repoRoot, "keycloak-testing", ".env"),
			composeFiles: []string{filepath.Join(repoRoot, "keycloak-testing", "compose.yml")},
			profile:      "keycloak",
		})
	}

	if cfg.LogtoActive() {
		projects = append(projects, project{
			name:         "stack-logto",
			envFile:      filepath.Join(repoRoot, "logto-testing", ".env"),
			composeFiles: []string{filepath.Join(repoRoot, "logto-testing", "compose.yml")},
			profile:      "logto",
		})
	}

	if cfg.LlamaActive() {
		projects = append(projects, project{
			name:         "stack-llama",
			envFile:      filepath.Join(stackDir, "llama.env"),
			composeFiles: []string{filepath.Join(stackDir, "compose.llama.yml")},
		})
	}

	// rag-mcp-server is always included.
	projects = append(projects, project{
		name:         "stack-rag",
		envFile:      filepath.Join(repoRoot, "rag-mcp-server", ".env"),
		composeFiles: []string{filepath.Join(repoRoot, "rag-mcp-server", "compose.yaml")},
	})

	return projects
}

// Up generates configs, ensures the shared network exists, then starts all
// active compose projects in dependency order.
func Up(cfg *config.Config, eng engine.Engine, repoRoot string, dryRun bool) error {
	written, err := generate.All(cfg, eng, repoRoot)
	if err != nil {
		return fmt.Errorf("generating configs: %w", err)
	}
	fmt.Fprintf(os.Stderr, "generated %d config files\n", len(written))

	if err := ensureNetwork(eng, dryRun); err != nil {
		return err
	}

	projects := activeProjects(cfg, repoRoot)
	for _, p := range projects {
		if err := projectUp(eng, p, dryRun); err != nil {
			return fmt.Errorf("starting %s: %w", p.name, err)
		}
		// Wait for health on postgres and auth providers before proceeding.
		if !dryRun {
			switch p.name {
			case "stack-postgres":
				if err := waitHealthy(eng, p.name+"-stack-postgres-1", healthTimeout); err != nil {
					return fmt.Errorf("postgres health: %w", err)
				}
			case "stack-keycloak":
				if err := waitHealthy(eng, p.name+"-keycloak-1", healthTimeout); err != nil {
					return fmt.Errorf("keycloak health: %w", err)
				}
			case "stack-logto":
				if err := waitHealthy(eng, p.name+"-logto-1", healthTimeout); err != nil {
					return fmt.Errorf("logto health: %w", err)
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr, "stack is up")
	return nil
}

// Down stops all active compose projects in reverse order.
func Down(cfg *config.Config, eng engine.Engine, repoRoot string, dryRun bool) error {
	projects := activeProjects(cfg, repoRoot)
	for i := len(projects) - 1; i >= 0; i-- {
		if err := projectDown(eng, projects[i], dryRun); err != nil {
			return fmt.Errorf("stopping %s: %w", projects[i].name, err)
		}
	}
	fmt.Fprintln(os.Stderr, "stack is down")
	return nil
}

// Restart stops then starts the stack.
func Restart(cfg *config.Config, eng engine.Engine, repoRoot string, dryRun bool) error {
	if err := Down(cfg, eng, repoRoot, dryRun); err != nil {
		return err
	}
	return Up(cfg, eng, repoRoot, dryRun)
}

// Status prints the container status for all active projects.
func Status(cfg *config.Config, eng engine.Engine, repoRoot string) error {
	projects := activeProjects(cfg, repoRoot)
	for _, p := range projects {
		args := append(eng.ProjectCmd(p.name, p.envFile, p.composeFiles), "ps")
		if err := runCmd(args); err != nil {
			fmt.Fprintf(os.Stderr, "status %s: %v\n", p.name, err)
		}
	}
	return nil
}

// Logs tails logs from the named component, or all components if component is "".
func Logs(cfg *config.Config, eng engine.Engine, repoRoot string, component string) error {
	projects := activeProjects(cfg, repoRoot)

	if component != "" {
		for _, p := range projects {
			if matchesComponent(p.name, component) {
				args := append(eng.ProjectCmd(p.name, p.envFile, p.composeFiles), "logs", "-f")
				return runCmd(args)
			}
		}
		return fmt.Errorf("unknown component %q; valid components: %s", component, projectNames(projects))
	}

	// Tail all — run each in a goroutine and block until cancelled.
	done := make(chan error, len(projects))
	for _, p := range projects {
		p := p
		go func() {
			args := append(eng.ProjectCmd(p.name, p.envFile, p.composeFiles), "logs", "-f")
			done <- runCmd(args)
		}()
	}
	// Wait for first exit (likely Ctrl-C).
	return <-done
}

// ─── internal helpers ─────────────────────────────────────────────────────────

func ensureNetwork(eng engine.Engine, dryRun bool) error {
	args := eng.NetworkCreateCmd(sharedNetwork)
	if dryRun {
		fmt.Fprintln(os.Stderr, "[dry-run]", strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // args built from controlled constants
	output, err := cmd.CombinedOutput()
	if err != nil {
		// "already exists" is not an error.
		if strings.Contains(string(output), "already exists") {
			return nil
		}
		return fmt.Errorf("creating network %s: %w (%s)", sharedNetwork, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func projectUp(eng engine.Engine, p project, dryRun bool) error {
	args := eng.ProjectCmd(p.name, p.envFile, p.composeFiles)
	if p.profile != "" {
		args = append(args, "--profile", p.profile)
	}
	args = append(args, "up", "-d")
	if dryRun {
		fmt.Fprintln(os.Stderr, "[dry-run]", strings.Join(args, " "))
		return nil
	}
	fmt.Fprintf(os.Stderr, "starting %s...\n", p.name)
	return runCmd(args)
}

func projectDown(eng engine.Engine, p project, dryRun bool) error {
	args := eng.ProjectCmd(p.name, p.envFile, p.composeFiles)
	if p.profile != "" {
		args = append(args, "--profile", p.profile)
	}
	args = append(args, "down")
	if dryRun {
		fmt.Fprintln(os.Stderr, "[dry-run]", strings.Join(args, " "))
		return nil
	}
	fmt.Fprintf(os.Stderr, "stopping %s...\n", p.name)
	return runCmd(args)
}

func runCmd(args []string) error {
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // args built from controlled values
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitHealthy(eng engine.Engine, containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	inspectArgs := eng.InspectHealthCmd(containerName)

	fmt.Fprintf(os.Stderr, "waiting for %s to be healthy...\n", containerName)
	for time.Now().Before(deadline) {
		cmd := exec.Command(inspectArgs[0], inspectArgs[1:]...) //nolint:gosec
		out, err := cmd.Output()
		if err == nil {
			status := strings.TrimSpace(string(out))
			switch status {
			case "healthy":
				fmt.Fprintf(os.Stderr, "%s is healthy\n", containerName)
				return nil
			case "unhealthy":
				return fmt.Errorf("container %s is unhealthy", containerName)
			}
		}
		time.Sleep(healthPoll)
	}
	return fmt.Errorf("timed out waiting for %s to become healthy after %s", containerName, timeout)
}

func matchesComponent(projectName, component string) bool {
	return projectName == component ||
		projectName == "stack-"+component ||
		strings.HasSuffix(projectName, "-"+component)
}

func projectNames(projects []project) string {
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = strings.TrimPrefix(p.name, "stack-")
	}
	return strings.Join(names, ", ")
}
