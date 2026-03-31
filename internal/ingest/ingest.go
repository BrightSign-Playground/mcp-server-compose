// Package ingest runs the docs2vector ingestion job as a one-shot container.
package ingest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
	"github.com/brightsign-playground/stack/internal/generate"
)

// Options configures an ingest run.
type Options struct {
	// Drop passes --drop to docs2vector (recreates tables). Default: true.
	Drop bool
	// DocsDir overrides cfg.Docs2Vector.DocsDir when non-empty.
	DocsDir string
}

const (
	imageTag      = "docs2vector:latest"
	sharedNetwork = "stack-net"
	llamaTimeout  = 5 * time.Second
)

// Run builds the docs2vector image and runs it as a one-shot container.
func Run(cfg *config.Config, eng engine.Engine, repoRoot string, opts Options) error {
	docsDir := opts.DocsDir
	if docsDir == "" {
		docsDir = cfg.Docs2Vector.DocsDir
	}

	// Validate docs directory.
	docsDir, err := validateDir(docsDir)
	if err != nil {
		return fmt.Errorf("docs_dir: %w", err)
	}

	// Generate (or refresh) component config files.
	if _, err := generate.All(cfg, eng, repoRoot); err != nil {
		return fmt.Errorf("generating configs: %w", err)
	}

	// Check llama-server reachability.
	embedHost := embedHostFromConfig(cfg)
	if err := checkLlamaReachable(embedHost); err != nil {
		return fmt.Errorf("llama-server not reachable at %s: %w", embedHost, err)
	}

	// Build the docs2vector image.
	contextDir := filepath.Join(repoRoot, "docs2vector")
	if err := buildImage(eng, contextDir); err != nil {
		return fmt.Errorf("building docs2vector image: %w", err)
	}

	// Derive DATABASE_URL for the container (uses stack-postgres service name).
	dbURL := databaseURLForContainer(cfg)

	configTomlPath := filepath.Join(repoRoot, "docs2vector", "config.toml")

	// Build run args.
	containerArgs := []string{"--dir", "/docs"}
	if opts.Drop {
		containerArgs = append(containerArgs, "--drop")
	}

	runOpts := engine.RunOptions{
		Image:   imageTag,
		Remove:  true,
		Network: sharedNetwork,
		EnvVars: map[string]string{
			"DATABASE_URL":        dbURL,
			"DOCS2VECTOR_CONFIG":  "/etc/docs2vector/config.toml",
		},
		Volumes: []string{
			configTomlPath + ":/etc/docs2vector/config.toml:ro",
			docsDir + ":/docs:ro",
		},
		Args: containerArgs,
	}

	// Docker needs extra_hosts for host-gateway since llama runs on the host.
	if !eng.IsPodman() {
		if h := eng.HostGatewayExtraHost(); h != "" {
			runOpts.ExtraHosts = []string{h}
		}
	}

	argv := eng.RunCmd(runOpts)
	fmt.Fprintln(os.Stderr, "running docs2vector ingest...")
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv built from controlled values
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("docs2vector exited with code %d", exitErr.ExitCode())
		}
		return err
	}

	fmt.Fprintln(os.Stderr, "ingest: completed successfully")
	return nil
}

// validateDir resolves path to an absolute path and verifies it is a directory.
func validateDir(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}
	if strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("path contains null byte: %q", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", abs)
	}
	return abs, nil
}

// checkLlamaReachable issues a GET to the llama-server health endpoint.
func checkLlamaReachable(embedHost string) error {
	client := &http.Client{Timeout: llamaTimeout}
	// Try /health first, fall back to /v1/models.
	for _, path := range []string{"/health", "/v1/models"} {
		url := embedHost + path
		resp, err := client.Get(url) //nolint:noctx // short-lived health check
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
	}
	return fmt.Errorf("no healthy response from %s", embedHost)
}

func buildImage(eng engine.Engine, contextDir string) error {
	argv := eng.BuildCmd(imageTag, contextDir, "")
	fmt.Fprintf(os.Stderr, "building %s...\n", imageTag)
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func databaseURLForContainer(cfg *config.Config) string {
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
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.Postgres.User, cfg.Postgres.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Postgres.Host, cfg.Postgres.Port),
		Path:     "/" + cfg.Postgres.Database,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

// embedHostFromConfig returns the embed host for reachability checks.
// llama-server runs on the host, so always use localhost.
func embedHostFromConfig(cfg *config.Config) string {
	port := cfg.Llama.HostPort
	if port == 0 {
		port = 16000
	}
	return fmt.Sprintf("http://localhost:%d", port)
}
