package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config represents the full fuego configuration loaded from config.yaml.
type Config struct {
	Site     SiteConfig            `yaml:"site"`
	Prebuild string                `yaml:"prebuild"`
	Dev      DevConfig             `yaml:"dev"`
	Dirs     DirsConfig            `yaml:"dirs"`
	Ignore   []string              `yaml:"ignore"`
	Routes   map[string]string     `yaml:"routes"`
	Collections map[string]CollectionConfig `yaml:"collections"`
	Taxonomies  map[string]TaxonomyConfig  `yaml:"taxonomies"`
	Parsers     map[string]ParserConfig    `yaml:"parsers"`
}

type SiteConfig struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
}

type DevConfig struct {
	Command   string `yaml:"command"`
	ProxyPort int    `yaml:"proxy_port"`
	Port      int    `yaml:"port"`
}

type DirsConfig struct {
	Content string `yaml:"content"`
	Theme   string `yaml:"theme"`
	Output  string `yaml:"output"`
	Static  string `yaml:"static"`
}

type CollectionConfig struct {
	Match  string `yaml:"match"`
	SortBy string `yaml:"sort_by"`
	Layout string `yaml:"layout"`
	Path   string `yaml:"path"`
}

type TaxonomyConfig struct {
	Path        string `yaml:"path"`
	Layout      string `yaml:"layout"`
	IndexPath   string `yaml:"index_path"`
	IndexLayout string `yaml:"index_layout"`
}

type ParserConfig struct {
	Rules []RuleConfig `yaml:"rules"`
}

type RuleConfig struct {
	Match string     `yaml:"match"`
	Emit  EmitConfig `yaml:"emit"`
}

type EmitConfig struct {
	Type       string         `yaml:"type"`
	Content    string         `yaml:"content"`
	Attributes map[string]any `yaml:"attributes"`
}

// Load reads a config file from disk and returns a Config with defaults applied.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyDefaults(cfg)

	if err := validateParsers(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateParsers checks that all regex patterns in declarative parser configs compile.
func validateParsers(cfg *Config) error {
	for name, pc := range cfg.Parsers {
		for i, rule := range pc.Rules {
			if _, err := regexp.Compile(rule.Match); err != nil {
				return fmt.Errorf("parser %q rule %d: invalid regex %q: %w", name, i, rule.Match, err)
			}
		}
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Dirs.Content == "" {
		cfg.Dirs.Content = "content"
	}
	if cfg.Dirs.Theme == "" {
		cfg.Dirs.Theme = "theme"
	}
	if cfg.Dirs.Output == "" {
		cfg.Dirs.Output = "build"
	}
	if cfg.Dirs.Static == "" {
		cfg.Dirs.Static = "public"
	}
	if cfg.Dev.Port == 0 {
		cfg.Dev.Port = 8080
	}
	// ProxyPort is only used when explicitly set — no default.
	// A value of 0 means no proxy (serve assets from build output).
	if cfg.Routes == nil {
		cfg.Routes = make(map[string]string)
	}
	if cfg.Collections == nil {
		cfg.Collections = make(map[string]CollectionConfig)
	}
	if cfg.Taxonomies == nil {
		cfg.Taxonomies = make(map[string]TaxonomyConfig)
	}
	if cfg.Parsers == nil {
		cfg.Parsers = make(map[string]ParserConfig)
	}
}
