package core

import (
	"net"
	"sync"
	"time"
)

// Conn wraps a net.Conn with gonx-specific state.
type Conn struct {
	raw          net.Conn
	createdAt    time.Time
	lastActivity time.Time
	requestCount int
	keepAlive    bool
	idleTimeout  time.Duration
	mu           sync.RWMutex
}

// Raw returns the underlying net.Conn.
func (c *Conn) Raw() net.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw
}

// SetKeepAlive enables or disables keep-alive.
func (c *Conn) SetKeepAlive(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keepAlive = v
}

// KeepAlive reports whether keep-alive is enabled.
func (c *Conn) KeepAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.keepAlive
}

// IncrementRequestCount increments the request counter.
func (c *Conn) IncrementRequestCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestCount++
	c.lastActivity = time.Now()
}

// RequestCount returns the number of requests processed.
func (c *Conn) RequestCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.requestCount
}

// SetIdleTimeout sets the idle timeout.
func (c *Conn) SetIdleTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idleTimeout = d
}

// IsIdle reports whether the connection has been idle too long.
func (c *Conn) IsIdle() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastActivity) > c.idleTimeout
}

// Touch updates the last activity timestamp.
func (c *Conn) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActivity = time.Now()
}

func (c *Conn) close() error {
	c.mu.RLock()
	raw := c.raw
	c.mu.RUnlock()
	if raw != nil {
		return raw.Close()
	}
	return nil
}

// ConnPool is a pool of reusable connections.
type ConnPool interface {
	Get() (*Conn, error)
	Put(c *Conn)
	Close() error
}

// SimpleConnPool is a basic connection pool.
type SimpleConnPool struct {
	pool chan *Conn
}

// NewSimpleConnPool creates a pool with the given capacity.
func NewSimpleConnPool(capacity int) *SimpleConnPool {
	return &SimpleConnPool{pool: make(chan *Conn, capacity)}
}

// Get retrieves a connection from the pool.
func (p *SimpleConnPool) Get() (*Conn, error) {
	select {
	case c := <-p.pool:
		return c, nil
	default:
		return &Conn{}, nil
	}
}

// Put returns a connection to the pool.
func (p *SimpleConnPool) Put(c *Conn) {
	select {
	case p.pool <- c:
	default:
		c.close()
	}
}

// Close closes all connections in the pool.
func (p *SimpleConnPool) Close() error {
	close(p.pool)
	for c := range p.pool {
		c.close()
	}
	return nil
}
