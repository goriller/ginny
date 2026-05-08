<<<<<<< HEAD
package config

import (
	"bytes"
	"flag"
	"net/url"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/google/wire"
	"github.com/goriller/ginny/logger"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

var (
	remoteConfig      string
	defaultConfigPath string
	// ConfigProviderSet
	ConfigProviderSet = wire.NewSet(NewConfig)
)

func init() {
	// 远程配置  etcd、consul
	flag.StringVar(&remoteConfig, "remote", "", "remote config provider: etcd://127.0.0.1:8500/test or consul://127.0.0.1:6577/test ")
	// 配置文件路径
	flag.StringVar(&defaultConfigPath, "conf", "./configs/config.yaml", "uri to load config")
}

// NewConfig
func NewConfig() (*viper.Viper, error) {
	var (
		err error
		v   = viper.New()
	)

	flag.Parse()

	v.AddConfigPath(".")
	v.SetConfigFile(defaultConfigPath)

	v.AutomaticEnv()
	v.SetEnvPrefix("ginny")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// 监听配置文件变更
	v.WatchConfig()
	v.OnConfigChange(func(_ fsnotify.Event) {
		log := logger.Default()
		log.Info("Config file updated.")
		err := loadConfig(v)
		if err != nil {
			log.Error("Config file reload error." + err.Error())
		}
	})

	// if err := v.ReadInConfig(); err == nil {
	// 	log.Printf("Config %s loaded successfully...", v.ConfigFileUsed())
	// } else {
	// 	return nil, err
	// }
	if err := loadConfig(v); err != nil {
		return nil, err
	}

	return v, err
}

// loadConfig
func loadConfig(v *viper.Viper) error {
	log := logger.Default()
	log.Info("Loading config...")
	// load config from remote
	p := os.Getenv("REMOTE_CONFIG")
	if remoteConfig == "" {
		remoteConfig = p
	}
	if remoteConfig != "" {
		return loadConfigFromRemote(v, remoteConfig)
	}
	data, err := os.ReadFile(v.ConfigFileUsed())
	if err != nil {
		return err
	}
	log.Info("Getting environment variables...")
	conf := expandEnv(string(data))
	err = v.ReadConfig(bytes.NewReader([]byte(conf)))
	if err != nil {
		return err
	}

	return nil
}

// loadConfigFromRemote
func loadConfigFromRemote(v *viper.Viper, uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	t := u.Query().Get("type")
	if t == "" {
		t = "json"
	}
	if err := v.AddRemoteProvider(u.Scheme, u.Host, u.Path); err != nil {
		return err
	}
	v.SetConfigType(t)

	if err := v.ReadRemoteConfig(); err != nil {
		return err
	}
	return nil
}

// expandEnv 寻找s中的 ${var} 并替换为环境变量的值，没有则替换为空，不解析 $var
func expandEnv(s string) string {
	var buf []byte
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' { // 只匹配${var} 不匹配$var
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getShellName(s[j+1:])
			if name == "" && w > 0 {
				// 非法匹配，去掉$
			} else if name == "" {
				buf = append(buf, s[j]) // 保留$
			} else {
				buf = append(buf, os.Getenv(name)...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return string(buf) + s[i:]
}

// getShellName 获取占位符的key，即${var}里面的var内容
// 返回 key内容 和 key长度
func getShellName(s string) (string, int) {
	// 匹配右括号 }
	// 输入已经保证第一个字符是{，并且至少两个字符以上
	for i := 1; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '"' { // "xx${xxx"
			return "", 0 // 遇到上面这些字符认为没有匹配中，保留$
		}
		if s[i] == '}' {
			if i == 1 { // ${}
				return "", 2 // 去掉${}
			}
			return s[1:i], i + 1
		}
	}
	return "", 0 // 没有右括号，保留$
}
=======
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
>>>>>>> feat/new
