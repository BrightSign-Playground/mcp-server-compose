// Package engine detects the available container engine and builds argv slices
// for compose and container run operations. All functions return explicit
// []string argument lists; no sh -c interpolation is ever used.
package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Kind identifies the container engine.
type Kind string

const (
	KindPodman Kind = "podman"
	KindDocker Kind = "docker"
)

// Engine holds the resolved binary and compose invocation style.
type Engine struct {
	Kind           Kind
	Binary         string // absolute path or bare name
	composeInline  bool   // true = "docker compose" plugin, false = standalone docker-compose
	lookPathFn     func(string) (string, error)
}

// RunOptions configures a one-shot container run.
type RunOptions struct {
	Image      string
	Name       string // optional container name
	Remove     bool   // --rm
	Network    string
	EnvVars    map[string]string
	Volumes    []string // "host:container[:options]"
	ExtraHosts []string // "hostname:ip" pairs (docker only for host-gateway)
	Args       []string // command arguments passed after the image
}

// Detect resolves the container engine. Priority:
//  1. flagOverride (from --engine CLI flag)
//  2. configEngine (from runtime.engine in stack.toml)
//  3. PATH auto-detection (prefer podman, fall back to docker)
func Detect(configEngine, flagOverride string) (Engine, error) {
	return detect(configEngine, flagOverride, exec.LookPath)
}

// detect is the testable inner implementation with an injectable lookPath.
func detect(configEngine, flagOverride string, lookPath func(string) (string, error)) (Engine, error) {
	chosen := flagOverride
	if chosen == "" {
		chosen = configEngine
	}

	if chosen != "" {
		return resolveEngine(chosen, lookPath)
	}

	// Auto-detect: prefer podman.
	if path, err := lookPath("podman"); err == nil {
		eng := Engine{Kind: KindPodman, Binary: path, lookPathFn: lookPath}
		fmt.Fprintln(os.Stderr, "using container engine: podman")
		return eng, nil
	}
	if path, err := lookPath("docker"); err == nil {
		eng, err := buildDockerEngine(path, lookPath)
		if err != nil {
			return Engine{}, err
		}
		fmt.Fprintln(os.Stderr, "using container engine: docker")
		return eng, nil
	}

	return Engine{}, errors.New("no container engine found: install podman or docker")
}

func resolveEngine(name string, lookPath func(string) (string, error)) (Engine, error) {
	switch strings.ToLower(name) {
	case "podman":
		path, err := lookPath("podman")
		if err != nil {
			return Engine{}, fmt.Errorf("podman not found on PATH: %w", err)
		}
		fmt.Fprintln(os.Stderr, "using container engine: podman")
		return Engine{Kind: KindPodman, Binary: path, lookPathFn: lookPath}, nil
	case "docker":
		path, err := lookPath("docker")
		if err != nil {
			return Engine{}, fmt.Errorf("docker not found on PATH: %w", err)
		}
		eng, err := buildDockerEngine(path, lookPath)
		if err != nil {
			return Engine{}, err
		}
		fmt.Fprintln(os.Stderr, "using container engine: docker")
		return eng, nil
	default:
		return Engine{}, fmt.Errorf("unknown engine %q: must be \"podman\" or \"docker\"", name)
	}
}

// buildDockerEngine determines whether to use "docker compose" plugin or
// standalone "docker-compose".
func buildDockerEngine(dockerBin string, lookPath func(string) (string, error)) (Engine, error) {
	// Check for "docker compose" plugin by attempting to find docker-compose
	// as a fallback. We prefer the plugin (inline).
	_, standaloneErr := lookPath("docker-compose")
	inline := standaloneErr != nil // if standalone not found, use inline plugin

	return Engine{
		Kind:          KindDocker,
		Binary:        dockerBin,
		composeInline: inline || true, // prefer plugin always; standalone is legacy
		lookPathFn:    lookPath,
	}, nil
}

// ComposeCmd returns the argv prefix for a compose invocation.
// e.g. ["podman", "compose"] or ["docker", "compose"]
func (e Engine) ComposeCmd() []string {
	switch e.Kind {
	case KindPodman:
		return []string{e.Binary, "compose"}
	case KindDocker:
		if e.composeInline {
			return []string{e.Binary, "compose"}
		}
		// Standalone docker-compose (legacy).
		if path, err := e.lookPathFn("docker-compose"); err == nil {
			return []string{path}
		}
		return []string{e.Binary, "compose"}
	default:
		return []string{e.Binary, "compose"}
	}
}

// ProjectCmd builds a full compose argv for a given project, env file, and
// set of compose files. Callers append subcommand arguments (e.g. "up", "-d").
func (e Engine) ProjectCmd(projectName, envFile string, composeFiles []string) []string {
	args := e.ComposeCmd()
	args = append(args, "-p", projectName)
	if envFile != "" {
		args = append(args, "--env-file", envFile)
	}
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	return args
}

// NetworkCreateCmd returns argv for creating a named network.
func (e Engine) NetworkCreateCmd(name string) []string {
	return []string{e.Binary, "network", "create", name}
}

// NetworkExistsCmd returns argv for checking if a network exists.
func (e Engine) NetworkExistsCmd(name string) []string {
	return []string{e.Binary, "network", "inspect", name}
}

// InspectHealthCmd returns argv to get the health status of a container.
func (e Engine) InspectHealthCmd(containerName string) []string {
	return []string{
		e.Binary, "inspect",
		"--format", "{{.State.Health.Status}}",
		containerName,
	}
}

// BuildCmd returns argv for building an image from a Dockerfile directory.
func (e Engine) BuildCmd(tag, contextDir, dockerfile string) []string {
	args := []string{e.Binary, "build", "-t", tag}
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}
	args = append(args, contextDir)
	return args
}

// ImageExistsCmd returns argv to check whether an image exists locally.
func (e Engine) ImageExistsCmd(tag string) []string {
	return []string{e.Binary, "image", "inspect", tag}
}

// RunCmd returns argv for a one-shot container run.
func (e Engine) RunCmd(opts RunOptions) []string {
	args := []string{e.Binary, "run"}
	if opts.Remove {
		args = append(args, "--rm")
	}
	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}
	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}
	for _, h := range opts.ExtraHosts {
		args = append(args, "--add-host", h)
	}
	// Sort env vars for determinism in tests.
	for _, key := range sortedKeys(opts.EnvVars) {
		args = append(args, "-e", key+"="+opts.EnvVars[key])
	}
	for _, v := range opts.Volumes {
		args = append(args, "-v", v)
	}
	args = append(args, opts.Image)
	args = append(args, opts.Args...)
	return args
}

// HostGatewayExtraHost returns the extra_hosts entry needed for Docker to
// resolve host-gateway. Not needed for Podman which uses host.containers.internal.
func (e Engine) HostGatewayExtraHost() string {
	if e.Kind == KindDocker {
		return "host-gateway:host-gateway"
	}
	return ""
}

// IsPodman reports whether the engine is podman.
func (e Engine) IsPodman() bool { return e.Kind == KindPodman }

// NewForTest constructs an Engine for use in tests without requiring binaries on PATH.
func NewForTest(kind Kind, binary string) Engine {
	return Engine{Kind: kind, Binary: binary, composeInline: true}
}

// sortedKeys returns map keys in sorted order for deterministic output.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — map is always small.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
