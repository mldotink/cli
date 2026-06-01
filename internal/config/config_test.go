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

	got := Resolve("", "", "", "", "", "")
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

	got := Resolve("", "", "", "", "", "")
	if got.Workspace != "child-workspace" {
		t.Fatalf("workspace = %q, want %q", got.Workspace, "child-workspace")
	}
}

func TestResolveEnvOverridesWorkspaceProjectAndHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("INK_API_KEY", "")
	t.Setenv("INK_WORKSPACE", "team-env")
	t.Setenv("INK_PROJECT", "api-env")
	t.Setenv("INK_API_URL", "https://enterprise.example.com/graphql")
	t.Setenv("INK_OAUTH_URL", "https://mcp.enterprise.example.com/")
	t.Setenv("INK_WEB_URL", "https://enterprise.example.com/")
	chdir(t, t.TempDir())

	if err := os.MkdirAll(filepath.Join(home, ".config", "ink"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(home, ".config", "ink", "config"),
		[]byte(`{"workspace":"team-global","project":"api-global","api_url":"https://api.ml.ink/graphql","oauth_url":"https://mcp.ml.ink"}`+"\n"),
		0o600,
	); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	got := Resolve("", "", "", "", "", "")
	if got.Workspace != "team-env" {
		t.Fatalf("workspace = %q, want %q", got.Workspace, "team-env")
	}
	if got.Project != "api-env" {
		t.Fatalf("project = %q, want %q", got.Project, "api-env")
	}
	if got.APIURL != "https://enterprise.example.com/graphql" {
		t.Fatalf("api url = %q, want enterprise api", got.APIURL)
	}
	if got.OAuthURL != "https://mcp.enterprise.example.com" {
		t.Fatalf("oauth url = %q, want trimmed enterprise oauth", got.OAuthURL)
	}
	if got.WebURL != "https://enterprise.example.com" {
		t.Fatalf("web url = %q, want trimmed enterprise web", got.WebURL)
	}
	if got.Sources["workspace"] != "env" || got.Sources["api_url"] != "env" {
		t.Fatalf("sources = %#v, want env for workspace/api_url", got.Sources)
	}
}

func TestSaveGlobalPersistsHostURLs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := SaveGlobal(&Config{
		APIURL:   "https://api.enterprise.example.com/graphql",
		OAuthURL: "https://mcp.enterprise.example.com/",
		WebURL:   "https://enterprise.example.com/",
	})
	if err != nil {
		t.Fatalf("save global: %v", err)
	}

	got := loadFile(GlobalPath())
	if got == nil {
		t.Fatal("global config was not written")
	}
	if got.APIURL != "https://api.enterprise.example.com/graphql" {
		t.Fatalf("api url = %q, want enterprise api", got.APIURL)
	}
	if got.OAuthURL != "https://mcp.enterprise.example.com" {
		t.Fatalf("oauth url = %q, want trimmed enterprise oauth", got.OAuthURL)
	}
	if got.WebURL != "https://enterprise.example.com" {
		t.Fatalf("web url = %q, want trimmed enterprise web", got.WebURL)
	}
}

func TestResolveFlagsOverrideEnvHosts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("INK_API_URL", "https://env.example.com/graphql")
	t.Setenv("INK_OAUTH_URL", "https://mcp.env.example.com")
	t.Setenv("INK_WEB_URL", "https://env.example.com")
	chdir(t, t.TempDir())

	got := Resolve(
		"",
		"",
		"",
		"https://flag.example.com/graphql",
		"https://mcp.flag.example.com/",
		"https://flag.example.com/",
	)
	if got.APIURL != "https://flag.example.com/graphql" {
		t.Fatalf("api url = %q, want flag api", got.APIURL)
	}
	if got.OAuthURL != "https://mcp.flag.example.com" {
		t.Fatalf("oauth url = %q, want trimmed flag oauth", got.OAuthURL)
	}
	if got.WebURL != "https://flag.example.com" {
		t.Fatalf("web url = %q, want trimmed flag web", got.WebURL)
	}
	if got.Sources["api_url"] != "flag" {
		t.Fatalf("api_url source = %q, want flag", got.Sources["api_url"])
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
