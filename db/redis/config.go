package redis

import (
	"fmt"
	"strings"
)

// Config redis基础配置
type Config struct {
	// 6.0及以上版本的redis采用ACL系统，使用Username鉴定连接
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	//业务库DB，使用ClusterClient时该配置失效（包括redis cluster部署、读写分离配置下的sentinel、读写分离或分片多主standalone）
	DB int `json:"db" mapstructure:"db"`

	//连接池可持有分配的最大连接数。driver默认 Client:10*NumCPU ClusterClient:5*NumCPU
	PoolSize int `json:"pool_size" mapstructure:"pool_size"`
	//最小空闲连接数
	MinIdleConns int `json:"min_idle_conns" mapstructure:"min_idle_conns"`
	//空闲连接超时时间(单位s)，保持空闲超过该持续时长的连接会由client主动关闭。driver默认5min，配置-1 则不检查空闲超时
	//该设置需小于redis服务端的timeout设置（建议redis服务端timeout设置为0，即服务端不主动断开连接）
	IdleTimeout int `json:"idle_timeout" mapstructure:"idle_timeout"`
	//idle connection reaper对空闲连接的检查频率(单位s)。driver默认1min，配置-1 则reaper无效 但若配置了IdleTimeout 空闲连接仍会被连接池丢弃
	IdleCheckFrequency int `json:"idle_check_frequency" mapstructure:"idle_check_frequency"`
	//连接龄期(单位s)，过旧连接会被client关闭。driver默认不关闭过旧连接
	MaxConnAge int `json:"max_conn_age" mapstructure:"max_conn_age"`
	//连接池超时时间(单位s)，连接池所有连接都忙时 client等待PoolTimeout后返回错误。driver默认ReadTimeout+1s
	PoolTimeout int `json:"pool_timeout" mapstructure:"pool_timeout"`

	//连接超时(单位s)，driver默认5s
	DialTimeout int `json:"dial_timeout" mapstructure:"dial_timeout"`
	//读超时(单位s)，driver默认3s，配置-1 无超时
	ReadTimeout int `json:"read_timeout" mapstructure:"read_timeout"`
	//写超时(单位s)，driver默认同ReadTimeout
	WriteTimeout int `json:"write_timeout" mapstructure:"write_timeout"`

	//读写分离配置，均为false时 不读从库 所有读写都连主库，二者均为true时driver优先选择RouteByLatency策略
	//允许路由只读命令到最近的主或从节点
	RouteByLatency bool `json:"route_by_latency" mapstructure:"route_by_latency"`
	//允许路由只读命令到任意主或从节点
	RouteRandomly bool `json:"route_randomly" mapstructure:"route_randomly"`

	//通过3种部署的必要配置有无 来区分部署模式：MasterName与SentinelAddrs、ClusterAddrs、StandaloneAddrs
	Sentinel   `json:"sentinel,inline" mapstructure:"sentinel,inline"`
	Cluster    `json:"cluster,inline" mapstructure:"cluster,inline"`
	Standalone `json:"standalone,inline" mapstructure:"standalone,inline"`
}

// String 打印可输出的配置
func (c *Config) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "redisdb confiy:")
	fmt.Fprintln(&str, "username:", c.Username)
	fmt.Fprintln(&str, "db:", c.DB)

	fmt.Fprintln(&str, "pool_size:", c.PoolSize)
	fmt.Fprintln(&str, "min_idle_conns:", c.MinIdleConns)
	fmt.Fprintln(&str, "idle_timeout:", c.IdleTimeout)
	fmt.Fprintln(&str, "idle_check_frequency:", c.IdleCheckFrequency)
	fmt.Fprintln(&str, "max_conn_age:", c.MaxConnAge)
	fmt.Fprintln(&str, "pool_timeout:", c.PoolTimeout)

	fmt.Fprintln(&str, "dial_timeout:", c.DialTimeout)
	fmt.Fprintln(&str, "read_timeout:", c.ReadTimeout)
	fmt.Fprintln(&str, "write_timeout:", c.WriteTimeout)

	fmt.Fprintln(&str, "route_by_latency:", c.RouteByLatency)
	fmt.Fprintln(&str, "route_randomly:", c.RouteRandomly)

	fmt.Fprintln(&str, c.Sentinel.String())
	fmt.Fprintln(&str, c.Cluster.String())
	fmt.Fprintln(&str, c.Standalone.String())

	return str.String()
}

// Sentinel 哨兵部署特性配置
type Sentinel struct {
	//主节点别名
	MasterName string `json:"master_name" mapstructure:"master_name"`
	//哨兵节点的地址、requirepass
	SentinelAddrs    []string `json:"sentinel_addrs" mapstructure:"sentinel_addrs"`
	SentinelPassword string   `json:"sentinel_password" mapstructure:"sentinel_password"`
}

// String 打印可输出的配置
func (s *Sentinel) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "sentinel confiy:")
	fmt.Fprintln(&str, "master_name:", s.MasterName)
	fmt.Fprintln(&str, "sentinel_addrs:", s.SentinelAddrs)
	return str.String()
}

// Cluster redisdb cluster部署特性配置
type Cluster struct {
	//集群节点地址
	ClusterAddrs []string `json:"cluster_addrs" mapstructure:"cluster_addrs"`
	//最大的重定向重试次数，网络错误或找错数据分片时 client会得到重定向错误和集群的最新情况 进行重定向。-1表示不限制，driver默认3次
	MaxRedirects int `json:"max_redirects" mapstructure:"max_redirects"`
}

// String 打印可输出的配置
func (c *Cluster) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "cluster confiy:")
	fmt.Fprintln(&str, "cluster_addrs:", c.ClusterAddrs)
	fmt.Fprintln(&str, "max_redirects:", c.MaxRedirects)
	return str.String()
}

// Standalone 单机、一主多从、或非redis哨兵/集群模式的分片集群部署(多主多从 此种情况应当少见) 特性配置
type Standalone struct {
	//需要读写分离时 需配出部署中的所有主从节点地址，1套主从为1个ClusterSlot，仅单机模式 只有1主没有从时 不写slaves配置
	StandaloneAddrs []struct {
		Master string   `json:"master" mapstructure:"master"`
		Slaves []string `json:"slaves" mapstructure:"slaves"`
	} `json:"standalone_addrs" mapstructure:"standalone_addrs"`
}

// String 打印可输出的配置
func (s *Standalone) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "standalone confiy:")
	for i := 0; i < len(s.StandaloneAddrs); i++ {
		fmt.Fprintln(&str, "master:", s.StandaloneAddrs[i].Master)
		fmt.Fprintln(&str, "slaves:", s.StandaloneAddrs[i].Slaves)
	}
	return str.String()
}
