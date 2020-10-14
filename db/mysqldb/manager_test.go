package mysqldb

import (
	"io/ioutil"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

// 连接测试、配置文件测试
func TestNewManager(t *testing.T) {
	data, err := ioutil.ReadFile("./db_config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		t.Fatal(err)
	}
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
