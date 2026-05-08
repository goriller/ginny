// Package config provides configuration management using koanf.
//
// Supports multiple configuration sources: files (YAML, JSON, TOML), environment
// variables, and optional remote providers.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/v2"
)

// Config holds all Ginny framework configuration.
type Config struct {
	App     AppConfig     `koanf:"app"`
	Server  ServerConfig  `koanf:"server"`
	Admin   AdminConfig   `koanf:"admin"`
	Logging LoggingConfig `koanf:"logging"`
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Name    string `koanf:"name"`
	Version string `koanf:"version"`
	Env     string `koanf:"env"` // dev, staging, prod
}

// ServerConfig holds the app server (RPC) configuration.
type ServerConfig struct {
	Addr string `koanf:"addr"` // e.g., ":8080"
}

// AdminConfig holds the admin server configuration.
type AdminConfig struct {
	Addr string `koanf:"addr"` // e.g., ":8081"
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn, error
	Format string `koanf:"format"` // json, text
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "ginny-app",
			Version: "0.1.0",
			Env:     "dev",
		},
		Server: ServerConfig{
			Addr: ":8080",
		},
		Admin: AdminConfig{
			Addr: ":8081",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Loader provides methods to load configuration from various sources.
type Loader struct {
	k    *koanf.Koanf
	opts loaderOptions
}

type loaderOptions struct {
	configPath string
	envPrefix  string
}

// LoaderOption configures the config loader.
type LoaderOption func(*loaderOptions)

// WithConfigPath sets the path to the config file.
func WithConfigPath(path string) LoaderOption {
	return func(o *loaderOptions) {
		o.configPath = path
	}
}

// WithEnvPrefix sets the prefix for environment variable overrides.
func WithEnvPrefix(prefix string) LoaderOption {
	return func(o *loaderOptions) {
		o.envPrefix = prefix
	}
}

// NewLoader creates a new config loader.
func NewLoader(opts ...LoaderOption) *Loader {
	o := loaderOptions{
		configPath: "./configs/config.yaml",
		envPrefix:  "GINNY",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Loader{
		k:    koanf.New("."),
		opts: o,
	}
}

// Load reads configuration from file and environment variables.
func (l *Loader) Load() (*Config, error) {
	// Load from file if it exists
	path := l.opts.configPath
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			provider := l.fileProvider(path)
			if err := l.k.Load(provider, nil); err != nil {
				return nil, fmt.Errorf("config: failed to load file %s: %w", path, err)
			}
		}
	}

	// Override from environment variables (GINNY_APP_NAME, etc.)
	envVars := os.Environ()
	envMap := make(map[string]string)
	prefix := strings.ToUpper(l.opts.envPrefix) + "_"
	for _, ev := range envVars {
		if strings.HasPrefix(strings.ToUpper(ev), prefix) {
			parts := strings.SplitN(ev, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimPrefix(parts[0], prefix))
				key = strings.ReplaceAll(key, "_", ".")
				envMap[key] = parts[1]
			}
		}
	}
	for k, v := range envMap {
		if err := l.k.Set(k, v); err != nil {
			return nil, fmt.Errorf("config: failed to set env var %s: %w", k, err)
		}
	}

	cfg := DefaultConfig()
	if err := l.k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal: %w", err)
	}

	return cfg, nil
}

// MustLoad loads config or panics.
func MustLoad(opts ...LoaderOption) *Config {
	cfg, err := NewLoader(opts...).Load()
	if err != nil {
		panic(fmt.Sprintf("config: failed to load: %v", err))
	}
	return cfg
}

// fileProvider creates a koanf provider for the given file path.
func (l *Loader) fileProvider(path string) *fileProviderImpl {
	return &fileProviderImpl{path: path}
}

// fileProviderImpl reads config files for koanf.
type fileProviderImpl struct {
	path string
}

func (p *fileProviderImpl) ReadBytes() ([]byte, error) {
	return os.ReadFile(p.path)
}

func (p *fileProviderImpl) Read() (map[string]interface{}, error) {
	return nil, fmt.Errorf("config: use ReadBytes for file provider")
}
