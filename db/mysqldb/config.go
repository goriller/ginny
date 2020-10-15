package mysqldb

import (
	"fmt"
	"strings"
)

// Config DB基础配置
type Config struct {
	WDB         Source   `yaml:"wdb"`
	RDBs        []Source `yaml:"rdbs"`
	DBName      string   `yaml:"dbname"`
	MaxOpenConn int      `yaml:"max_open_conn"`
	MaxIdleConn int      `yaml:"max_idle_conn"`
	MaxLifetime int      `yaml:"max_lifetime"`
	Keepalive   int      `yaml:"keepalive"`
}

// Source DB部署实例数据源配置
type Source struct {
	Host string `yaml:"host"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// String 打印可输出的配置
func (s *Source) String() string {
	return fmt.Sprintf("host:%s user:%s", s.Host, s.User)
}

// String 打印可输出的配置
func (c *Config) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "mysql confiy:")
	fmt.Fprintln(&str, "wdb:", c.WDB)
	fmt.Fprintln(&str, "rdbs:", c.RDBs)
	fmt.Fprintln(&str, "dbname:", c.DBName)
	fmt.Fprintln(&str, "max_open_conn:", c.MaxOpenConn)
	fmt.Fprintln(&str, "max_idle_conn:", c.MaxIdleConn)
	fmt.Fprintln(&str, "max_lifetime:", c.MaxLifetime)
	fmt.Fprintln(&str, "keepalive:", c.Keepalive)
	return str.String()
}
