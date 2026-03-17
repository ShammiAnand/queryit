package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type BastionConfig struct {
	User string `yaml:"user"`
	Host string `yaml:"host"`
	PEM  string `yaml:"pem"`
}

type Profile struct {
	Driver   string         `yaml:"driver,omitempty"` // postgres (default) | mysql | sqlite
	Host     string         `yaml:"host,omitempty"`
	Port     int            `yaml:"port,omitempty"`
	Database string         `yaml:"database"`
	User     string         `yaml:"user,omitempty"`
	Password string         `yaml:"password,omitempty"`
	SSLMode  string         `yaml:"sslmode,omitempty"`
	Bastion  *BastionConfig `yaml:"bastion,omitempty"`
}

func (p *Profile) DriverName() string {
	if p.Driver == "" {
		return "postgres"
	}
	return p.Driver
}

// ResolvedPassword returns the actual password, expanding $ENV_VAR references.
func (p *Profile) ResolvedPassword() (string, error) {
	if strings.HasPrefix(p.Password, "$") {
		envKey := p.Password[1:]
		val := os.Getenv(envKey)
		if val == "" {
			return "", fmt.Errorf("environment variable %q is not set or empty", envKey)
		}
		return val, nil
	}
	return p.Password, nil
}

type Settings struct {
	PageSize     int    `yaml:"page_size"`
	DefaultView  string `yaml:"default_view"`
	HistorySize  int    `yaml:"history_size"`
	QueryTimeout int    `yaml:"query_timeout"`
}

type Config struct {
	Profiles map[string]*Profile `yaml:"profiles"`
	Settings Settings            `yaml:"settings"`
}

func DefaultConfig() *Config {
	return &Config{
		Profiles: make(map[string]*Profile),
		Settings: Settings{
			PageSize:     20,
			DefaultView:  "table",
			HistorySize:  1000,
			QueryTimeout: 30,
		},
	}
}

func ConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(base, "queryit", "config.yaml")
}

func CachePath(profile string) string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(base, "queryit", profile, "schema.json")
}

func DataPath(profile string) string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(base, "queryit", profile, "history")
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := Save(cfg); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Settings.PageSize == 0 {
		cfg.Settings.PageSize = 20
	}
	if cfg.Settings.DefaultView == "" {
		cfg.Settings.DefaultView = "table"
	}
	if cfg.Settings.HistorySize == 0 {
		cfg.Settings.HistorySize = 1000
	}
	if cfg.Settings.QueryTimeout == 0 {
		cfg.Settings.QueryTimeout = 30
	}
}

func Save(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func AddProfile(name string, p *Profile) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Profiles[name] = p
	return Save(cfg)
}

func RemoveProfile(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(cfg.Profiles, name)
	return Save(cfg)
}
