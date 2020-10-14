package redisdb

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v2"
)

// TestNewManager 连接测试
// 测试standalone 一主多从：非读写分离模式(Client)、读写分离模式(ClusterClient)：对DB配置、多Key命令等的影响
// 测试脚本注册与获取
func TestNewManager(t *testing.T) {
	config, err := testLoadConfig(t)
	if err != nil {
		t.Fatal(err)
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	//对DB配置的影响：Client DB配置生效，ClusterClient DB配置无效 默认选择DB 0
	setRes, err := manager.DB().Set(context.Background(), "key1", "value1", time.Duration(1)*time.Minute).Result()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(setRes)

	//对多key命令的影响：因为仅一主多从 是1套cluster slot 0-16383，暂未看到对set交集并集等多key命令的影响
	t.Log(manager.DB().SAdd(context.Background(), "set1", "a", "b", "c").Result())
	t.Log(manager.DB().SAdd(context.Background(), "set2", "a", "e", "f").Result())
	t.Log(manager.DB().SUnion(context.Background(), "set1", "set2").Result())
	t.Log(manager.DB().SInter(context.Background(), "set1", "set2").Result())

	//测试脚本
	type scriptKeyTest struct{}
	testScript := `redisdb.call('SET',KEYS[1],ARGV[1])`
	if err := manager.RegisterScript(scriptKeyTest{}, redis.NewScript(testScript)); err != nil {
		t.Fatal(err)
	}
	t.Log(manager.Script(scriptKeyTest{}).Run(context.Background(), manager.DB(), []string{"sKey1"}, "sValue1").Result())
	scriptRes, err := manager.DB().Get(context.Background(), "sKey1").Result()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(scriptRes)
	if scriptRes != "sValue1" {
		t.Fatal("assert fail")
	}
}

// TestClusterSlot 测试ClusterSlot函数 slot分配：如0、1、2、3、4份……多主分片部署时 start与end
func TestClusterSlot(t *testing.T) {
	config, err := testLoadConfig(t)
	if err != nil {
		t.Fatal(err)
	}

	if len(config.StandaloneAddrs) <= 0 {
		t.Fatal("empty cluster slots")
	}
	slots := make([]redis.ClusterSlot, 0, len(config.StandaloneAddrs))
	slotInterval := 16384 / len(config.StandaloneAddrs)
	for i := 0; i < len(config.StandaloneAddrs); i++ {
		nodes := make([]redis.ClusterNode, 0, len(config.StandaloneAddrs[i].Slaves)+1)
		nodes = append(nodes, redis.ClusterNode{Addr: config.StandaloneAddrs[i].Master})
		for _, slave := range config.StandaloneAddrs[i].Slaves {
			nodes = append(nodes, redis.ClusterNode{Addr: slave})
		}
		slot := redis.ClusterSlot{
			Start: i * slotInterval,
			End:   (i+1)*slotInterval - 1,
			Nodes: nodes,
		}
		if i == len(config.StandaloneAddrs)-1 {
			slot.End = 16383
		}
		slots = append(slots, slot)
	}
	t.Log(slots)
}

func testLoadConfig(t *testing.T) (*Config, error) {
	data, err := ioutil.ReadFile("./redisdb_config.yaml")
	if err != nil {
		return nil, err
	}
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	t.Log(config)
	return config, nil
}
