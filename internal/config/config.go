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
	// Packs holds per-pack config subtrees (packs.{name}:). Each registered
	// pack receives its own subtree to validate in Go; the engine does not
	// interpret the contents.
	Packs map[string]map[string]any `yaml:"packs"`
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
	Match    string `yaml:"match"`
	SortBy   string `yaml:"sort_by"`
	Layout   string `yaml:"layout"`
	Path     string `yaml:"path"`
	PageSize int    `yaml:"page_size"` // 0 = no pagination
}

type TaxonomyConfig struct {
	Path        string `yaml:"path"`
	Layout      string `yaml:"layout"`
	IndexPath   string `yaml:"index_path"`
	IndexLayout string `yaml:"index_layout"`
	PageSize    int    `yaml:"page_size"` // 0 = no pagination (term pages)
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
	cfg, _, err := LoadLayered(path, nil)
	return cfg, err
}

// LoadLayered reads the user config at path and deep-merges the given pack
// config layers beneath it (packs lowest, in registration order; the user
// config always wins). It returns the resolved config and the provenance of
// every key. Validation runs on the merged result.
func LoadLayered(path string, packLayers []Layer) (*Config, *Provenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading config: %w", err)
	}

	userMap := map[string]any{}
	if err := yaml.Unmarshal(data, &userMap); err != nil {
		return nil, nil, fmt.Errorf("parsing config: %w", err)
	}

	layers := make([]Layer, 0, len(packLayers)+1)
	layers = append(layers, packLayers...)
	layers = append(layers, Layer{Source: "user", Data: userMap})
	merged, prov := mergeLayers(layers)

	// Round-trip the merged map into the typed Config.
	mergedYAML, err := yaml.Marshal(merged)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding merged config: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(mergedYAML, cfg); err != nil {
		return nil, nil, fmt.Errorf("decoding merged config: %w", err)
	}

	applyDefaults(cfg)

	if err := validateParsers(cfg, prov); err != nil {
		return nil, nil, err
	}
	if err := validatePageSizes(cfg, prov); err != nil {
		return nil, nil, err
	}

	return cfg, prov, nil
}

// ParsePackLayer builds a config Layer from a pack's name and its YAML
// config-defaults fragment (which may be empty).
func ParsePackLayer(name string, yamlBytes []byte) (Layer, error) {
	m := map[string]any{}
	if len(yamlBytes) > 0 {
		if err := yaml.Unmarshal(yamlBytes, &m); err != nil {
			return Layer{}, fmt.Errorf("pack %q config defaults: %w", name, err)
		}
	}
	return Layer{Source: name, Data: m}, nil
}

// sourceSuffix returns " (from pack \"X\")" when the value at path came from a
// pack, and "" when it came from the user config or an engine default.
func sourceSuffix(prov *Provenance, path string) string {
	src := prov.Source(path)
	if src == "" || src == "user" {
		return ""
	}
	return fmt.Sprintf(" (from pack %q)", src)
}

// validatePageSizes rejects negative page_size values.
func validatePageSizes(cfg *Config, prov *Provenance) error {
	for name, c := range cfg.Collections {
		if c.PageSize < 0 {
			return fmt.Errorf("collection %q: page_size must be >= 0, got %d%s",
				name, c.PageSize, sourceSuffix(prov, "collections."+name))
		}
	}
	for name, t := range cfg.Taxonomies {
		if t.PageSize < 0 {
			return fmt.Errorf("taxonomy %q: page_size must be >= 0, got %d%s",
				name, t.PageSize, sourceSuffix(prov, "taxonomies."+name))
		}
	}
	return nil
}

// validateParsers checks that all regex patterns in declarative parser configs compile.
func validateParsers(cfg *Config, prov *Provenance) error {
	for name, pc := range cfg.Parsers {
		for i, rule := range pc.Rules {
			if _, err := regexp.Compile(rule.Match); err != nil {
				return fmt.Errorf("parser %q rule %d: invalid regex %q%s: %w",
					name, i, rule.Match, sourceSuffix(prov, "parsers."+name), err)
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
