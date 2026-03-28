// Command stack orchestrates the MCP server stack via podman or docker compose.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brightsign-playground/stack/internal/compose"
	"github.com/brightsign-playground/stack/internal/config"
	"github.com/brightsign-playground/stack/internal/engine"
	"github.com/brightsign-playground/stack/internal/generate"
	"github.com/brightsign-playground/stack/internal/ingest"
)

// globals holds the parsed global flags.
type globals struct {
	configPath string
	engine     string
	dryRun     bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	g, rest, err := parseGlobals(args)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		printUsage()
		return errors.New("no command specified")
	}

	sub := rest[0]
	subArgs := rest[1:]

	// For validate, load and validate config only.
	if sub == "validate" {
		return cmdValidate(g)
	}

	// All other commands need a loaded + validated config + engine.
	cfg, eng, err := loadAll(g)
	if err != nil {
		return err
	}

	repoRoot := repoRootFromConfig(g.configPath)

	switch sub {
	case "up":
		return compose.Up(cfg, eng, repoRoot, g.dryRun)
	case "down":
		return compose.Down(cfg, eng, repoRoot, g.dryRun)
	case "restart":
		return compose.Restart(cfg, eng, repoRoot, g.dryRun)
	case "status":
		return compose.Status(cfg, eng, repoRoot)
	case "logs":
		component := ""
		if len(subArgs) > 0 {
			component = subArgs[0]
		}
		return compose.Logs(cfg, eng, repoRoot, component)
	case "generate":
		written, err := generate.All(cfg, eng, repoRoot)
		if err != nil {
			return err
		}
		for _, p := range written {
			fmt.Fprintln(os.Stdout, "wrote:", p)
		}
		return nil
	case "ingest":
		return cmdIngest(cfg, eng, repoRoot, subArgs)
	default:
		return fmt.Errorf("unknown command %q; run 'stack help' for usage", sub)
	}
}

func cmdValidate(g globals) error {
	cfg, err := config.Load(g.configPath)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "stack.toml is valid")
	return nil
}

func cmdIngest(cfg *config.Config, eng engine.Engine, repoRoot string, args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	noDrop := fs.Bool("no-drop", false, "skip dropping and recreating tables")
	docsDir := fs.String("docs-dir", "", "override docs directory from config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	effective := *docsDir
	if effective == "" {
		effective = cfg.Docs2Vector.DocsDir
	}
	if effective == "" {
		return fmt.Errorf("no docs directory specified\n\nUsage:\n  make ingest ARGS=\"--docs-dir /path/to/docs\"\n  stack ingest --docs-dir /path/to/docs")
	}

	opts := ingest.Options{
		Drop:    !*noDrop,
		DocsDir: *docsDir,
	}
	return ingest.Run(cfg, eng, repoRoot, opts)
}

func loadAll(g globals) (*config.Config, engine.Engine, error) {
	cfg, err := config.Load(g.configPath)
	if err != nil {
		return nil, engine.Engine{}, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, engine.Engine{}, err
	}
	eng, err := engine.Detect(cfg.Runtime.Engine, g.engine)
	if err != nil {
		return nil, engine.Engine{}, err
	}
	return cfg, eng, nil
}

// parseGlobals splits global flags from the remaining arguments.
func parseGlobals(args []string) (globals, []string, error) {
	fs := flag.NewFlagSet("stack", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var g globals
	fs.StringVar(&g.configPath, "config", "stack.toml", "path to stack.toml")
	fs.StringVar(&g.engine, "engine", "", "container engine: podman|docker (overrides config)")
	fs.BoolVar(&g.dryRun, "dry-run", false, "print compose commands without executing")

	// flag.Parse stops at the first non-flag argument and returns the rest
	// via fs.Args(), which is exactly what we want for subcommand parsing.
	if err := fs.Parse(args); err != nil {
		return globals{}, nil, err
	}
	g.configPath = fs.Lookup("config").Value.String()
	g.engine = fs.Lookup("engine").Value.String()

	return g, fs.Args(), nil
}

// repoRootFromConfig derives the repository root from the config file path.
// The config file is expected to live at the repo root.
func repoRootFromConfig(configPath string) string {
	abs, err := absolutePath(configPath)
	if err != nil {
		return "."
	}
	return parentDir(abs)
}

func absolutePath(path string) (string, error) {
	if len(path) == 0 {
		return "", errors.New("empty path")
	}
	if path[0] == '/' {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd + "/" + path, nil
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: stack [--config stack.toml] [--engine podman|docker] [--dry-run] <command>

Commands:
  up        Generate configs and start all enabled services
  down      Stop all services
  restart   Stop then start all services
  status    Show service status
  ingest    Run docs2vector ingestion  [--no-drop] [--docs-dir PATH]
  logs      Tail logs  [component]
  generate  Generate component configs without starting services
  validate  Validate stack.toml and exit`)
}
