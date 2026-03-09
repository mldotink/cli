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
}

type Resolved struct {
	APIKey    string
	Workspace string
	Project   string
	Sources   map[string]string // field -> "global", "local", "env", "flag"
}

func Resolve(flagAPIKey, flagWorkspace, flagProject string) *Resolved {
	r := &Resolved{Sources: make(map[string]string)}

	// 1. Global config (try new path, fall back to legacy)
	g := loadFile(GlobalPath())
	if g == nil {
		g = loadFile(legacyGlobalPath())
	}
	if g != nil {
		set(r, "api_key", g.APIKey, "global")
		set(r, "workspace", g.Workspace, "global")
		set(r, "project", g.Project, "global")
	}

	// 2. Local .ink (overrides global)
	if l := loadFile(".ink"); l != nil {
		set(r, "api_key", l.APIKey, "local")
		set(r, "workspace", l.Workspace, "local")
		set(r, "project", l.Project, "local")
	}

	// 3. Env var (API key only)
	if key := os.Getenv("INK_API_KEY"); key != "" {
		r.APIKey = strings.TrimSpace(key)
		r.Sources["api_key"] = "env"
	}

	// 4. CLI flags (highest priority)
	if flagAPIKey != "" {
		r.APIKey = flagAPIKey
		r.Sources["api_key"] = "flag"
	}
	if flagWorkspace != "" {
		r.Workspace = flagWorkspace
		r.Sources["workspace"] = "flag"
	}
	if flagProject != "" {
		r.Project = flagProject
		r.Sources["project"] = "flag"
	}

	return r
}

func set(r *Resolved, field, value, source string) {
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
	}
	r.Sources[field] = source
}

func loadFile(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Fallback: plain text API key (backward compat with skill format)
		key := strings.TrimSpace(string(data))
		if strings.HasPrefix(key, "dk_") {
			return &Config{APIKey: key}
		}
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

func legacyGlobalPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ink", "credentials")
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
