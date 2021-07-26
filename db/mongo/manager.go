package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	mongoD "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/tag"
)

var globalManager *Manager

// Manager 客户端连接池管理器 对可用同1套认证凭据访问的1个deployment(副本集/分片集群实例)
type Manager struct {
	Client   *mongoD.Client
	Database *mongoD.Database
}

// NewManager 根据基础配置 初始化连接池管理器
// Manager所有方法 均不对dbname进行trim、大小写等处理，由调用方检查 Manager保持入参原样
func NewManager(config *Config) (*Manager, error) {
	if globalManager != nil {
		return globalManager, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ConnectTimeout)*time.Second)
	defer cancel()
	client, err := mongoD.Connect(ctx, clientOptions(config))
	if err != nil {
		return nil, err
	}
	//首先ping主节点，主节点若up则无错返回。主节点down时，寻找其他可ping节点，若有1个节点up则无错返回。或直到ServerSelectionTimeout返回错误（driver默认30s）
	//使用ping会降低应用弹性，因为有可能节点是短暂down或正在自动故障转移。所以此处保证集群里有一个节点up 则可启动
	if err := client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, err
	}

	mgr := &Manager{
		Client:   client,
		Database: client.Database(config.DefaultDBName),
	}

	return mgr, nil
}

// NewManagerFromClient 从客户端连接池实例client 初始化管理器
// 不同业务库部署在同一实例集群 可用同一套认证凭据访问，且业务方需指定不同的默认DB来使用时，可使用本初始化方法 由业务方控制多个Manager共享client
func NewManagerFromClient(client *mongo.Client, defaultDBName string) *Manager {
	defaultDB := client.Database(defaultDBName)
	return &Manager{
		Client:   client,
		Database: defaultDB,
	}
}

// Close 释放所有连接池使用的资源。该函数应当很少用到
func (m *Manager) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := m.Client.Disconnect(ctx); err != nil {
		log.Fatal(fmt.Sprintf("disconnect mongodb client error: %s", err.Error()))
	}
	globalManager = nil
}

func clientOptions(config *Config) *options.ClientOptions {
	opt := options.Client()

	//集群实例设置
	opt.SetHosts(config.Hosts)
	opt.SetReplicaSet(config.ReplicaSet)

	//读首选项设置
	var readPrefOpts []readpref.Option
	if config.ReadPreference.MaxStaleness > 0 {
		readPrefOpts = append(readPrefOpts, readpref.WithMaxStaleness(
			time.Duration(config.ReadPreference.MaxStaleness)*time.Second))
	}
	if len(config.ReadPreference.Tags) > 0 {
		tags := make([]tag.Tag, 0, len(config.ReadPreference.Tags))
		for name, value := range config.ReadPreference.Tags {
			tags = append(tags, tag.Tag{Name: name, Value: value})
		}
		readPrefOpts = append(readPrefOpts, readpref.WithTagSets(tags))
	}
	switch config.ReadPreference.Mode {
	case int(readpref.PrimaryPreferredMode):
		opt.SetReadPreference(readpref.PrimaryPreferred(readPrefOpts...))
	case int(readpref.SecondaryMode):
		opt.SetReadPreference(readpref.Secondary(readPrefOpts...))
	case int(readpref.SecondaryPreferredMode):
		opt.SetReadPreference(readpref.SecondaryPreferred(readPrefOpts...))
	case int(readpref.NearestMode):
		opt.SetReadPreference(readpref.Nearest(readPrefOpts...))
		//未配置、非法值等 均与PrimaryMode配置一致处理：不显式设置read preference，采用driver默认的Primary模式 无其他首选项配置
	}

	//身份认证设置
	if config.Auth.Mechanism != "" || len(config.Auth.MechanismProperties) > 0 || config.Auth.Source != "" ||
		config.Auth.Username != "" || config.Auth.Password != "" || config.Auth.PasswordSet {
		//无Credential时 需要不调用SetAuth()才可正常连接
		opt.SetAuth(options.Credential{
			AuthMechanism:           config.Auth.Mechanism,
			AuthMechanismProperties: config.Auth.MechanismProperties,
			AuthSource:              config.Auth.Source,
			Username:                config.Auth.Username,
			Password:                config.Auth.Password,
			PasswordSet:             config.Auth.PasswordSet,
		})
	}

	//连接与连接池设置
	opt.SetConnectTimeout(time.Duration(config.ConnectTimeout) * time.Second)
	opt.SetMaxConnIdleTime(time.Duration(config.MaxConnIdleTime) * time.Second)
	opt.SetMaxPoolSize(uint64(config.MaxPoolSize))
	opt.SetMinPoolSize(uint64(config.MinPoolSize))

	return opt
}
