package mysqldb

import (
	"testing"

	"github.com/gorillazer/ginny/confiy"
)

// 连接测试、配置文件测试
func TestNewManager(t *testing.T) {
	cfg := &Config{}
	confiy.Init().LoadConf(cfg, "./db_config.yaml")

	t.Log(cfg)

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := mgr.RDB().QueryRow("SELECT count(1) FROM user").Scan(&count); err != nil {
		t.Fatal(err)
	}
	t.Log(count)
}
