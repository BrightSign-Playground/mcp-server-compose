package engine

import (
	"errors"
	"testing"
)

// fakeLookPath simulates exec.LookPath with a fixed map of name → path.
func fakeLookPath(available map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if path, ok := available[name]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

func TestDetect_flagOverridePodman(t *testing.T) {
	lookup := fakeLookPath(map[string]string{"podman": "/usr/bin/podman"})
	eng, err := detect("", "podman", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindPodman {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindPodman)
	}
}

func TestDetect_flagOverrideDocker(t *testing.T) {
	lookup := fakeLookPath(map[string]string{"docker": "/usr/bin/docker"})
	eng, err := detect("", "docker", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindDocker {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindDocker)
	}
}

func TestDetect_configEngine(t *testing.T) {
	lookup := fakeLookPath(map[string]string{"podman": "/usr/bin/podman"})
	eng, err := detect("podman", "", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindPodman {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindPodman)
	}
}

func TestDetect_flagOverrideWinsOverConfig(t *testing.T) {
	lookup := fakeLookPath(map[string]string{
		"podman": "/usr/bin/podman",
		"docker": "/usr/bin/docker",
	})
	eng, err := detect("podman", "docker", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindDocker {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindDocker)
	}
}

func TestDetect_autoDetectPrefersPodman(t *testing.T) {
	lookup := fakeLookPath(map[string]string{
		"podman": "/usr/bin/podman",
		"docker": "/usr/bin/docker",
	})
	eng, err := detect("", "", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindPodman {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindPodman)
	}
}

func TestDetect_autoDetectFallsBackToDocker(t *testing.T) {
	lookup := fakeLookPath(map[string]string{"docker": "/usr/bin/docker"})
	eng, err := detect("", "", lookup)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if eng.Kind != KindDocker {
		t.Errorf("kind: got %q, want %q", eng.Kind, KindDocker)
	}
}

func TestDetect_nothingFound(t *testing.T) {
	lookup := fakeLookPath(map[string]string{})
	_, err := detect("", "", lookup)
	if err == nil {
		t.Fatal("expected error when no engine found")
	}
}

func TestDetect_unknownEngine(t *testing.T) {
	lookup := fakeLookPath(map[string]string{})
	_, err := detect("containerd", "", lookup)
	if err == nil {
		t.Fatal("expected error for unknown engine")
	}
}

func TestEngine_ComposeCmd_podman(t *testing.T) {
	eng := Engine{Kind: KindPodman, Binary: "/usr/bin/podman"}
	cmd := eng.ComposeCmd()
	if len(cmd) != 2 || cmd[0] != "/usr/bin/podman" || cmd[1] != "compose" {
		t.Errorf("unexpected ComposeCmd: %v", cmd)
	}
}

func TestEngine_ComposeCmd_docker(t *testing.T) {
	eng := Engine{Kind: KindDocker, Binary: "/usr/bin/docker", composeInline: true}
	cmd := eng.ComposeCmd()
	if len(cmd) != 2 || cmd[0] != "/usr/bin/docker" || cmd[1] != "compose" {
		t.Errorf("unexpected ComposeCmd: %v", cmd)
	}
}

func TestEngine_ProjectCmd(t *testing.T) {
	eng := Engine{Kind: KindPodman, Binary: "podman"}
	got := eng.ProjectCmd("stack-rag", ".stack/rag.env", []string{"rag-mcp-server/compose.yaml"})
	want := []string{"podman", "compose", "-p", "stack-rag", "--env-file", ".stack/rag.env", "-f", "rag-mcp-server/compose.yaml"}
	if len(got) != len(want) {
		t.Fatalf("ProjectCmd len: got %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ProjectCmd[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEngine_RunCmd(t *testing.T) {
	eng := Engine{Kind: KindPodman, Binary: "podman"}
	opts := RunOptions{
		Image:   "myimage:latest",
		Remove:  true,
		Network: "stack-net",
		EnvVars: map[string]string{"FOO": "bar"},
		Volumes: []string{"/host:/container:ro"},
		Args:    []string{"--flag"},
	}
	got := eng.RunCmd(opts)
	// Check structure: podman run --rm --network stack-net -e FOO=bar -v /host:/container:ro myimage:latest --flag
	if got[0] != "podman" || got[1] != "run" {
		t.Errorf("expected 'podman run' prefix, got %v", got[:2])
	}
	if got[len(got)-1] != "--flag" {
		t.Errorf("expected last arg to be --flag, got %q", got[len(got)-1])
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]string{"ZZZ": "1", "AAA": "2", "MMM": "3"}
	got := sortedKeys(m)
	if got[0] != "AAA" || got[1] != "MMM" || got[2] != "ZZZ" {
		t.Errorf("sortedKeys: got %v", got)
	}
}
