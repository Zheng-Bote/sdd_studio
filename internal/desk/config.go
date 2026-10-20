// Package desk is the optional AI desktop sdd-desk (ADR 0002): it talks to the selected language model and runs sdd
// tools only through a `sdd mcp` child process, each call with the user's permission. sdd itself never imports it.
package desk

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Protocols spoken by the providers.
const (
	ProtocolOllama = "ollama"
	ProtocolOpenAI = "openai"
	ProtocolClaude = "claude"
)

// LocalOllamaID is the provider id of the local Ollama instance.
const LocalOllamaID = "ollama-local"

// Provider is one configured language model provider.
type Provider struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol string   `json:"protocol"`
	Endpoint string   `json:"endpoint"`
	Cloud    bool     `json:"cloud"`
	Models   []string `json:"models"`
	APIKey   string   `json:"-"`
}

type providerSpec struct {
	prefix, id, name, protocol, endpoint string
	cloud                                bool
}

// providerSpecs mirrors md_indexer's .env format, plus OLLAMA_LOCAL_ for the local instance.
var providerSpecs = []providerSpec{
	{"OLLAMA_LOCAL", LocalOllamaID, "Ollama (lokal)", ProtocolOllama, "http://localhost:11434", false},
	{"OLLAMA", "ollama-cloud", "Ollama Cloud", ProtocolOllama, "https://ollama.com", true},
	{"OPENROUTER", "openrouter", "OpenRouter", ProtocolOpenAI, "https://openrouter.ai/api/v1", true},
	{"NVIDIA", "nvidia", "NVIDIA NIM", ProtocolOpenAI, "https://integrate.api.nvidia.com/v1", true},
	{"CLAUDE", "claude", "Claude", ProtocolClaude, "https://api.anthropic.com", true},
	{"GEMINI", "gemini", "Gemini", ProtocolOpenAI, "https://generativelanguage.googleapis.com/v1beta/openai", true},
}

// Config is the loaded provider configuration.
type Config struct {
	Path      string     `json:"path"`
	Providers []Provider `json:"providers"`
	Warnings  []string   `json:"warnings"`
}

// ErrInsecureConfig is returned when llm.env is readable by group or others.
var ErrInsecureConfig = errors.New("config/llm.env ist für Gruppe oder andere lesbar")

// DefaultConfigPath is config/llm.env next to the sdd-desk executable (ADR 0002): keys belong to the program, not to
// a project.
func DefaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), "config", "llm.env")
}

// LoadConfig reads the provider configuration. A missing file at the default location means "no cloud providers";
// a missing explicit file is an error. Values from the process environment (getenv) take precedence.
func LoadConfig(path string, explicit bool, getenv func(string) string) (*Config, error) {
	cfg := &Config{Path: path, Providers: []Provider{}, Warnings: []string{}}
	vars := map[string]string{}
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("%w (%s, Rechte %04o) — chmod 600 %s", ErrInsecureConfig, path, info.Mode().Perm(), path)
		}
		if vars, err = parseEnvFile(path); err != nil {
			return nil, err
		}
	case errors.Is(err, os.ErrNotExist) && !explicit:
		cfg.Warnings = append(cfg.Warnings, "keine Konfiguration unter "+path+"; nur lokales Ollama verfügbar")
	default:
		return nil, fmt.Errorf("Konfiguration %s nicht lesbar: %w", path, err)
	}
	value := func(key string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return strings.TrimSpace(vars[key])
	}

	for _, spec := range providerSpecs {
		p := Provider{ID: spec.id, Name: spec.name, Protocol: spec.protocol, Endpoint: spec.endpoint, Cloud: spec.cloud, Models: []string{}}
		if ep := value(spec.prefix + "_ENDPOINT"); ep != "" {
			p.Endpoint = strings.TrimRight(ep, "/")
		}
		for i := 1; i <= 99; i++ {
			if m := value(fmt.Sprintf("%s_LLM_%02d", spec.prefix, i)); m != "" {
				p.Models = append(p.Models, m)
			}
		}
		if spec.cloud {
			key := value(spec.prefix + "_API_KEY")
			if key == "" || strings.EqualFold(key, "none") {
				continue
			}
			p.APIKey = key
			if len(p.Models) == 0 {
				cfg.Warnings = append(cfg.Warnings, spec.name+": kein Modell eingetragen ("+spec.prefix+"_LLM_01)")
			}
		}
		cfg.Providers = append(cfg.Providers, p)
	}
	return cfg, nil
}

// parseEnvFile reads KEY=VALUE lines; # comments, blank lines, an optional `export ` prefix and surrounding quotes are
// handled.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	vars := map[string]string{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: KEY=VALUE erwartet", path, n)
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		vars[strings.TrimSpace(key)] = val
	}
	return vars, sc.Err()
}
