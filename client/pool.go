package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var (
	ErrPoolClosed = errors.New("connection pool is closed")
	ErrNoConn     = errors.New("no available connections")
)

// ConnPool gRPC连接池
type ConnPool struct {
	mu          sync.RWMutex
	target      string
	options     []grpc.DialOption
	conns       []*poolConn
	maxSize     int
	minSize     int
	current     int
	closed      bool
	healthCheck time.Duration

	// 性能统计
	stats ConnPoolStats
}

// poolConn 池化连接包装
type poolConn struct {
	*grpc.ClientConn
	created  time.Time
	lastUsed time.Time
	useCount int64
	inUse    bool
}

// ConnPoolStats 连接池统计信息
type ConnPoolStats struct {
	TotalConns    int   `json:"total_conns"`
	ActiveConns   int   `json:"active_conns"`
	IdleConns     int   `json:"idle_conns"`
	TotalRequests int64 `json:"total_requests"`
	TotalErrors   int64 `json:"total_errors"`
}

// ConnPoolConfig 连接池配置
type ConnPoolConfig struct {
	Target              string
	MaxSize             int
	MinSize             int
	HealthCheckInterval time.Duration
	Options             []grpc.DialOption
}

// NewConnPool 创建新的连接池
func NewConnPool(config ConnPoolConfig) (*ConnPool, error) {
	if config.MaxSize <= 0 {
		config.MaxSize = 10
	}
	if config.MinSize < 0 {
		config.MinSize = 1
	}
	if config.MinSize > config.MaxSize {
		config.MinSize = config.MaxSize
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	pool := &ConnPool{
		target:      config.Target,
		options:     config.Options,
		conns:       make([]*poolConn, 0, config.MaxSize),
		maxSize:     config.MaxSize,
		minSize:     config.MinSize,
		healthCheck: config.HealthCheckInterval,
	}

	// 初始化最小连接数
	for i := 0; i < config.MinSize; i++ {
		conn, err := pool.createConn()
		if err != nil {
			pool.Close()
			return nil, err
		}
		pool.conns = append(pool.conns, conn)
		pool.current++
	}

	// 启动健康检查
	go pool.healthChecker()

	return pool, nil
}

// Get 获取连接
func (p *ConnPool) Get(ctx context.Context) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	p.stats.TotalRequests++

	// 查找可用连接
	for _, conn := range p.conns {
		if !conn.inUse && p.isHealthy(conn) {
			conn.inUse = true
			conn.lastUsed = time.Now()
			conn.useCount++
			p.stats.ActiveConns++
			p.stats.IdleConns--
			return conn.ClientConn, nil
		}
	}

	// 如果没有可用连接且未达到最大连接数，创建新连接
	if p.current < p.maxSize {
		conn, err := p.createConn()
		if err != nil {
			p.stats.TotalErrors++
			return nil, err
		}

		conn.inUse = true
		conn.lastUsed = time.Now()
		conn.useCount++

		p.conns = append(p.conns, conn)
		p.current++
		p.stats.ActiveConns++

		return conn.ClientConn, nil
	}

	p.stats.TotalErrors++
	return nil, ErrNoConn
}

// Put 归还连接
func (p *ConnPool) Put(conn *grpc.ClientConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	// 查找对应的池化连接
	for _, poolConn := range p.conns {
		if poolConn.ClientConn == conn {
			if poolConn.inUse {
				poolConn.inUse = false
				poolConn.lastUsed = time.Now()
				p.stats.ActiveConns--
				p.stats.IdleConns++
			}
			break
		}
	}
}

// Stats 获取连接池统计信息
func (p *ConnPool) Stats() ConnPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.TotalConns = p.current
	return stats
}

// Close 关闭连接池
func (p *ConnPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// 关闭所有连接
	for _, conn := range p.conns {
		conn.ClientConn.Close()
	}

	p.conns = nil
	p.current = 0

	return nil
}

// createConn 创建新连接
func (p *ConnPool) createConn() (*poolConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientConn, err := grpc.DialContext(ctx, p.target, p.options...)
	if err != nil {
		return nil, err
	}

	return &poolConn{
		ClientConn: clientConn,
		created:    time.Now(),
		lastUsed:   time.Now(),
	}, nil
}

// isHealthy 检查连接健康状态
func (p *ConnPool) isHealthy(conn *poolConn) bool {
	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// healthChecker 健康检查协程
func (p *ConnPool) healthChecker() {
	ticker := time.NewTicker(p.healthCheck)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}

		// 检查并清理不健康的连接
		var healthyConns []*poolConn
		for _, conn := range p.conns {
			if p.isHealthy(conn) {
				healthyConns = append(healthyConns, conn)
			} else {
				// 关闭不健康的连接
				if !conn.inUse {
					conn.ClientConn.Close()
					p.current--
					if conn.inUse {
						p.stats.ActiveConns--
					} else {
						p.stats.IdleConns--
					}
				}
			}
		}
		p.conns = healthyConns

		// 如果连接数低于最小值，创建新连接
		for p.current < p.minSize {
			conn, err := p.createConn()
			if err != nil {
				break
			}
			p.conns = append(p.conns, conn)
			p.current++
			p.stats.IdleConns++
		}

		p.mu.Unlock()
	}
}

// PooledClientConn 池化连接包装器，实现自动归还
type PooledClientConn struct {
	*grpc.ClientConn
	pool   *ConnPool
	closed bool
	mu     sync.Mutex
}

// Close 重写Close方法，实现连接归还而非关闭
func (p *PooledClientConn) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		p.pool.Put(p.ClientConn)
		p.closed = true
	}
	return nil
}

// GetPooled 获取池化连接（带自动归还功能）
func (p *ConnPool) GetPooled(ctx context.Context) (*PooledClientConn, error) {
	conn, err := p.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &PooledClientConn{
		ClientConn: conn,
		pool:       p,
	}, nil
}
