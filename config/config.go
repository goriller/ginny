package config

import (
	"bytes"
	"flag"
	"io/ioutil"
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
	v.OnConfigChange(func(in fsnotify.Event) {
		logger.Default().Info("Config file updated.")
		err := loadConfig(v)
		if err != nil {
			logger.Default().Error("Config file reload error." + err.Error())
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
	logger.Default().Info("Loading config...")
	// load config from remote
	p := os.Getenv("REMOTE_CONFIG")
	if remoteConfig == "" {
		remoteConfig = p
	}
	if remoteConfig != "" {
		return loadConfigFromRemote(v, remoteConfig)
	}
	data, err := ioutil.ReadFile(v.ConfigFileUsed())
	if err != nil {
		return err
	}
	logger.Default().Info("Getting environment variables...")
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
