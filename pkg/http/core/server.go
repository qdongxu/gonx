package core

import (
	"context"
	"net"
	"sync"
	"time"
)

// Server manages listeners, handlers, and the request lifecycle.
type Server struct {
	config   *ServerConfig
	listener net.Listener
	handler  Handler
	phase    PhaseEngine
	mu       sync.RWMutex
	active   bool
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// ServerConfig holds runtime configuration.
type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewServer creates a Server with the given configuration.
func NewServer(cfg *ServerConfig) *Server {
	if cfg == nil {
		cfg = &ServerConfig{Addr: ":8080",
			ReadTimeout: 30 * time.Second,
			IdleTimeout: 120 * time.Second}
	}
	return &Server{config: cfg, shutdown: make(chan struct{})}
}

// SetHandler sets the root handler. Must be called before Start.
func (s *Server) SetHandler(h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

// SetPhaseEngine sets the phase engine.
func (s *Server) SetPhaseEngine(p PhaseEngine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = p
}

// Handler returns the current root handler.
func (s *Server) Handler() Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.handler
}

// Start begins listening on the configured address.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return nil
	}
	if s.handler == nil {
		s.mu.Unlock()
		return errNoHandler
	}
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.listener = ln
	s.active = true
	s.mu.Unlock()
	go s.serve()
	return nil
}

// Stop immediately closes the listener.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return nil
	}
	s.active = false
	close(s.shutdown)
	return s.listener.Close()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return nil
	}
	s.active = false
	close(s.shutdown)
	ln := s.listener
	s.mu.Unlock()
	if err := ln.Close(); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr returns the listener address.
func (s *Server) Addr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// IsActive reports whether the server is running.
func (s *Server) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

func (s *Server) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return
			default:
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(raw net.Conn) {
	defer s.wg.Done()
	c := &Conn{raw: raw, createdAt: time.Now(),
		lastActivity: time.Now(), keepAlive: true,
		idleTimeout: s.config.IdleTimeout}
	defer c.close()
	if s.phase != nil {
		ctx := &PhaseContext{Conn: c, Server: s,
			Handler: s.handler}
		_ = s.phase.Run(ctx)
	}
}

type serverError string

func (e serverError) Error() string { return string(e) }

const errNoHandler serverError = "server: no handler configured"
