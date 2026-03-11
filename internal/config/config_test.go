package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFindsLocalConfigInParentDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("INK_API_KEY", "")

	repo := filepath.Join(t.TempDir(), "repo")
	nested := filepath.Join(repo, "apps", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".ink"), []byte("{\"workspace\":\"team-local\",\"project\":\"backend\"}\n"), 0o600); err != nil {
		t.Fatalf("write .ink: %v", err)
	}

	chdir(t, nested)

	got := Resolve("", "", "")
	if got.Workspace != "team-local" {
		t.Fatalf("workspace = %q, want %q", got.Workspace, "team-local")
	}
	if got.Project != "backend" {
		t.Fatalf("project = %q, want %q", got.Project, "backend")
	}
	if got.Sources["workspace"] != "local" {
		t.Fatalf("workspace source = %q, want %q", got.Sources["workspace"], "local")
	}
}

func TestResolvePrefersNearestAncestorLocalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("INK_API_KEY", "")

	repo := filepath.Join(t.TempDir(), "repo")
	child := filepath.Join(repo, "apps")
	nested := filepath.Join(child, "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".ink"), []byte("{\"workspace\":\"root-workspace\"}\n"), 0o600); err != nil {
		t.Fatalf("write root .ink: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, ".ink"), []byte("{\"workspace\":\"child-workspace\"}\n"), 0o600); err != nil {
		t.Fatalf("write child .ink: %v", err)
	}

	chdir(t, nested)

	got := Resolve("", "", "")
	if got.Workspace != "child-workspace" {
		t.Fatalf("workspace = %q, want %q", got.Workspace, "child-workspace")
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}
