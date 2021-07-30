package mysql

import (
	"fmt"
	"strings"
)

// Config DB基础配置
type Config struct {
	WDB         Source `json:"wdb" yaml:"wdb"`
	RDB         Source `json:"rdb" yaml:"rdb"` // 多个RDB host使用,分隔
	DBName      string `json:"dbname" yaml:"dbname"`
	MaxOpenConn int    `json:"max_open_conn" yaml:"max_open_conn"`
	MaxIdleConn int    `json:"max_idle_conn" yaml:"max_idle_conn"`
	MaxLifetime int    `json:"max_lifetime" yaml:"max_lifetime"`
	Keepalive   int    `json:"keepalive" yaml:"keepalive"`
}

// Source DB部署实例数据源配置
type Source struct {
	Host string `json:"host" yaml:"host"`
	User string `json:"user" yaml:"user"`
	Pass string `json:"pass" yaml:"pass"`
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
	fmt.Fprintln(&str, "rdb:", c.RDB)
	fmt.Fprintln(&str, "dbname:", c.DBName)
	fmt.Fprintln(&str, "max_open_conn:", c.MaxOpenConn)
	fmt.Fprintln(&str, "max_idle_conn:", c.MaxIdleConn)
	fmt.Fprintln(&str, "max_lifetime:", c.MaxLifetime)
	fmt.Fprintln(&str, "keepalive:", c.Keepalive)
	return str.String()
}
