package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey    string `json:"api_key,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Project   string `json:"project,omitempty"`
	APIURL    string `json:"api_url,omitempty"`
	OAuthURL  string `json:"oauth_url,omitempty"`
	WebURL    string `json:"web_url,omitempty"`
}

type Resolved struct {
	APIKey    string
	Workspace string
	Project   string
	APIURL    string
	OAuthURL  string
	WebURL    string
	Sources   map[string]string // field -> "global", "local", "env", "flag"
}

func Resolve(flagAPIKey, flagWorkspace, flagProject, flagAPIURL, flagOAuthURL, flagWebURL string) *Resolved {
	r := &Resolved{Sources: make(map[string]string)}

	// 1. Global config
	g := loadFile(GlobalPath())
	if g != nil {
		set(r, "api_key", g.APIKey, "global")
		set(r, "workspace", g.Workspace, "global")
		set(r, "project", g.Project, "global")
		set(r, "api_url", g.APIURL, "global")
		set(r, "oauth_url", g.OAuthURL, "global")
		set(r, "web_url", g.WebURL, "global")
	}

	// 2. Local .ink from current directory or nearest ancestor (overrides global)
	if l := loadFile(localPath()); l != nil {
		set(r, "api_key", l.APIKey, "local")
		set(r, "workspace", l.Workspace, "local")
		set(r, "project", l.Project, "local")
		set(r, "api_url", l.APIURL, "local")
		set(r, "oauth_url", l.OAuthURL, "local")
		set(r, "web_url", l.WebURL, "local")
	}

	// 3. Env vars
	if key := os.Getenv("INK_API_KEY"); key != "" {
		set(r, "api_key", key, "env")
	}
	if workspace := os.Getenv("INK_WORKSPACE"); workspace != "" {
		set(r, "workspace", workspace, "env")
	}
	if project := os.Getenv("INK_PROJECT"); project != "" {
		set(r, "project", project, "env")
	}
	if apiURL := os.Getenv("INK_API_URL"); apiURL != "" {
		set(r, "api_url", apiURL, "env")
	}
	if oauthURL := firstEnv("INK_OAUTH_URL", "INK_MCP_URL"); oauthURL != "" {
		set(r, "oauth_url", oauthURL, "env")
	}
	if webURL := os.Getenv("INK_WEB_URL"); webURL != "" {
		set(r, "web_url", webURL, "env")
	}

	// 4. CLI flags (highest priority)
	if flagAPIKey != "" {
		set(r, "api_key", flagAPIKey, "flag")
	}
	if flagWorkspace != "" {
		set(r, "workspace", flagWorkspace, "flag")
	}
	if flagProject != "" {
		set(r, "project", flagProject, "flag")
	}
	if flagAPIURL != "" {
		set(r, "api_url", flagAPIURL, "flag")
	}
	if flagOAuthURL != "" {
		set(r, "oauth_url", flagOAuthURL, "flag")
	}
	if flagWebURL != "" {
		set(r, "web_url", flagWebURL, "flag")
	}

	return r
}

func localPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, ".ink")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func set(r *Resolved, field, value, source string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	switch field {
	case "api_key":
		r.APIKey = value
	case "workspace":
		r.Workspace = value
	case "project":
		r.Project = value
	case "api_url":
		r.APIURL = value
	case "oauth_url":
		r.OAuthURL = strings.TrimRight(value, "/")
	case "web_url":
		r.WebURL = strings.TrimRight(value, "/")
	}
	r.Sources[field] = source
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func loadFile(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func GlobalPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ink", "config")
}

func SaveGlobal(cfg *Config) error {
	path := GlobalPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeConfig(path, cfg)
}

func SaveLocal(cfg *Config) error {
	if err := writeConfig(".ink", cfg); err != nil {
		return err
	}
	addToGitignore(".ink")
	return nil
}

func writeConfig(path string, update *Config) error {
	// Read existing, merge, write back
	existing := loadFile(path)
	if existing == nil {
		existing = &Config{}
	}
	if update.APIKey != "" {
		existing.APIKey = update.APIKey
	}
	if update.Workspace != "" {
		existing.Workspace = update.Workspace
	}
	if update.Project != "" {
		existing.Project = update.Project
	}
	if update.APIURL != "" {
		existing.APIURL = strings.TrimSpace(update.APIURL)
	}
	if update.OAuthURL != "" {
		existing.OAuthURL = strings.TrimRight(strings.TrimSpace(update.OAuthURL), "/")
	}
	if update.WebURL != "" {
		existing.WebURL = strings.TrimRight(strings.TrimSpace(update.WebURL), "/")
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func addToGitignore(entry string) {
	data, _ := os.ReadFile(".gitignore")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			return
		}
	}
	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}
