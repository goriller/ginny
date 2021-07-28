package config

import (
	"log"
	"os"
	"strings"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

// Init 初始化viper
func New(path string) (*viper.Viper, error) {
	var (
		err error
		v   = viper.New()
	)
	log.Println("Loading config...")
	v.AddConfigPath(".")
	v.SetConfigFile(string(path))

	if err := v.ReadInConfig(); err == nil {
		log.Printf("Config %s loaded successfully...", v.ConfigFileUsed())
	} else {
		return nil, err
	}
	log.Println("Getting environment variables...")
	for _, k := range viper.AllKeys() {
		value := viper.GetString(k)
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			viper.Set(k, getEnv(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")))
		}
	}

	return v, err
}

func getEnv(env string) string {
	res := os.Getenv(env)
	if len(env) == 0 {
		return ""
	}
	return res
}

var ProviderSet = wire.NewSet(New)
