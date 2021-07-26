package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Manager redis管理器：client连接池(cluster、sentinel、standalone部署模式，可配置读写分离)，lua脚本管理
type Manager struct {
	client redis.UniversalClient
	script map[interface{}]*redis.Script
}

// NewManager 根据基础配置 初始化redis管理器
func NewManager(config *Config) (*Manager, error) {
	var client redis.UniversalClient
	if len(config.ClusterAddrs) > 0 {
		client = newCluster(config)
	} else if len(config.SentinelAddrs) > 0 && config.MasterName != "" {
		client = newSentinel(config)
	} else if len(config.StandaloneAddrs) > 0 {
		client = newStandalone(config)
	} else {
		return nil, errors.New("invalid deploy mode confiy")
	}

	return &Manager{
		client: client,
		script: make(map[interface{}]*redis.Script),
	}, client.Ping(context.Background()).Err()
}

// DB 返回连接池客户端
func (m *Manager) DB() redis.UniversalClient {
	return m.client
}

// RegisterScript 注册redis lua script：若服务用到某个脚本，需在服务初始化时注册脚本，每增加一个脚本均需注册
// 若redis服务实例的脚本缓存未缓存该脚本则Load，以便后续调用只需传脚本的SHA1校验和：节省带宽、且redis无需再次编译脚本，
//   主要为方便script与pipeline等结合使用时可直接调用EvalSha（单独使用script时driver已做处理）
// kvs必须偶数个: key1 value1 key2 value2 ...。key为各自服务包内定义，可参考type scriptKeyXXX struct{} 空结构体。
//   value是相应脚本*redisdb.Script=redisdb.NewScript()
func (m *Manager) RegisterScript(kvs ...interface{}) error {
	if len(kvs)%2 != 0 {
		return fmt.Errorf("len(kvs)(%d) not even", len(kvs))
	}
	for i := 0; i < len(kvs)-1; i = i + 2 {
		if kvs[i] == nil || kvs[i+1] == nil {
			return errors.New("nil key or value")
		}
		v, ok := kvs[i+1].(*redis.Script)
		if !ok {
			return fmt.Errorf("key(%T), value type(%T) not *redisdb.Script", kvs[i], kvs[i+1])
		}
		if _, ok := m.script[kvs[i]]; ok {
			return fmt.Errorf("duplicate registered key(%T)", kvs[i])
		}
		if err := loadScript(m.client, v); err != nil {
			return err
		}
		m.script[kvs[i]] = v
	}
	return nil
}

// Script 根据脚本key取脚本结构
// 调用方保证：需在服务初始化时 调用RegisterScript()注册相应的键值对，对未注册的key 本函数将返回nil
func (m *Manager) Script(key interface{}) *redis.Script {
	return m.script[key]
}

// Close 释放连接池使用的资源。该函数应当很少用到
func (m *Manager) Close() {
	if err := m.client.Close(); err != nil {
		log.Fatal("close redisdb client error", zap.Error(err))
	}
}

func newCluster(config *Config) redis.UniversalClient {
	return redis.NewClusterClient(clusterOptions(config))
}

func newSentinel(config *Config) redis.UniversalClient {
	//未配置主从读写分离：采用被验证较多的NewFailoverClient()函数 返回Client，redisdb sentinel automatic failover
	if !config.RouteByLatency && !config.RouteRandomly {
		return redis.NewFailoverClient(failoverOptions(config))
	}
	//配置主从读写分离：采用v8 experimental NewFailoverClusterClient()函数 返回ClusterClient，其自动故障转移、主从读写分离、多key命令等最好仔细验证
	//其底层借助ClusterSlots配置，一套哨兵的主从节点 作为1个ClusterSlot 0-16383：使用ClusterClient的读写分离能力，但会导致DB配置项失效
	return redis.NewFailoverClusterClient(failoverOptions(config))
}

func newStandalone(config *Config) redis.UniversalClient {
	//未配置主从读写分离，且仅单机一主(有无从随意)：采用被验证较多的NewClient()函数 返回Client
	if !config.RouteByLatency && !config.RouteRandomly && len(config.StandaloneAddrs) == 1 {
		return redis.NewClient(options(config))
	}
	//配置主从读写分离，或standalone独立部署了多主分片节点：采用NewClusterClient()函数 ClusterSlots配置 返回ClusterClient
	//用ClusterClient的读写分离能力，但会导致DB配置项失效，且若部署了多主分片节点需注意多key命令(哈希标签)
	return redis.NewClusterClient(clusterOptions(config))
}

//sentinel部署模式配置
func failoverOptions(config *Config) *redis.FailoverOptions {
	return &redis.FailoverOptions{
		MasterName:       config.MasterName,
		SentinelAddrs:    config.SentinelAddrs,
		SentinelPassword: config.SentinelPassword,

		//以下2个字段若有为true的，只能用于NewFailoverClusterClient()，若用于NewFailoverClient()会引起panic
		RouteByLatency: config.RouteByLatency,
		RouteRandomly:  config.RouteRandomly,

		Username: config.Username,
		Password: config.Password,
		DB:       config.DB, //使用NewFailoverClusterClient()时 DB配置无效

		DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,

		PoolSize:           config.PoolSize,
		MinIdleConns:       config.MinIdleConns,
		MaxConnAge:         time.Duration(config.MaxConnAge) * time.Second,
		PoolTimeout:        time.Duration(config.PoolTimeout) * time.Second,
		IdleTimeout:        time.Duration(config.IdleTimeout) * time.Second,
		IdleCheckFrequency: time.Duration(config.IdleCheckFrequency) * time.Second,
	}
}

//cluster部署模式配置，或standalone部署模式（自建多主分片集群，或有主从读写分离配置）
func clusterOptions(config *Config) *redis.ClusterOptions {
	opt := &redis.ClusterOptions{
		//sentinel NewFailoverClusterClient()使用的FailoverOptions未考虑单独ReadOnly配置项，此处统一令cluster部署模式也不考虑单独ReadOnly配置项
		RouteByLatency: config.RouteByLatency,
		RouteRandomly:  config.RouteRandomly,

		Username: config.Username,
		Password: config.Password,
		//ClusterClient DB配置项无效

		DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,

		PoolSize:           config.PoolSize,
		MinIdleConns:       config.MinIdleConns,
		MaxConnAge:         time.Duration(config.MaxConnAge) * time.Second,
		PoolTimeout:        time.Duration(config.PoolTimeout) * time.Second,
		IdleTimeout:        time.Duration(config.IdleTimeout) * time.Second,
		IdleCheckFrequency: time.Duration(config.IdleCheckFrequency) * time.Second,
	}

	if len(config.ClusterAddrs) > 0 { //cluster部署模式
		opt.Addrs = config.ClusterAddrs
		opt.MaxRedirects = config.MaxRedirects
		return opt
	}

	//standalone部署模式（自建多主分片集群，或有主从读写分离配置），采用ClusterSlots配置 返回ClusterClient
	opt.ClusterSlots = func(ctx context.Context) ([]redis.ClusterSlot, error) {
		if len(config.StandaloneAddrs) <= 0 {
			return nil, errors.New("empty cluster slots")
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
		return slots, nil
	}
	return opt
}

//standalone 单机/主从部署模式(无自建多主分片集群) 且不要求主从读写分离时，可调用options配置 仅配置master地址(不在意实际部署是否有从库)，来初始化Client对象
func options(config *Config) *redis.Options {
	return &redis.Options{
		Addr: config.StandaloneAddrs[0].Master,

		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,

		DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,

		PoolSize:           config.PoolSize,
		MinIdleConns:       config.MinIdleConns,
		MaxConnAge:         time.Duration(config.MaxConnAge) * time.Second,
		PoolTimeout:        time.Duration(config.PoolTimeout) * time.Second,
		IdleTimeout:        time.Duration(config.IdleTimeout) * time.Second,
		IdleCheckFrequency: time.Duration(config.IdleCheckFrequency) * time.Second,
	}
}

func loadScript(client redis.UniversalClient, script *redis.Script) error {
	exists, err := script.Exists(context.Background(), client).Result()
	if err != nil {
		return err
	}
	if len(exists) <= 0 || !exists[0] {
		return script.Load(context.Background(), client).Err()
	}
	return nil
}
