package core

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// === Server ===

func TestServerLifecycle(t *testing.T) {
	s := NewServer(&ServerConfig{Addr: "127.0.0.1:0"})
	s.SetHandler(NewPlaceholderHandler())

	if s.IsActive() {
		t.Fatal("expected inactive")
	}
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !s.IsActive() {
		t.Fatal("expected active")
	}
	if s.Addr() == nil {
		t.Fatal("expected addr")
	}

	// Dial to verify listener works.
	conn, err := net.Dial("tcp", s.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if s.IsActive() {
		t.Fatal("expected inactive after shutdown")
	}
}

func TestServerStartNoHandler(t *testing.T) {
	s := NewServer(nil)
	if err := s.Start(); err == nil {
		t.Fatal("expected error for missing handler")
	}
}

func TestServerShutdownInactive(t *testing.T) {
	s := NewServer(nil)
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// === Handler ===

func TestHandlerFunc(t *testing.T) {
	called := false
	f := HandlerFunc(func(req *Request) *Response {
		called = true
		return NewResponse()
	})
	f.Handle(AcquireRequest())
	if !called {
		t.Fatal("expected called")
	}
}

func TestWrapUnwrapHTTPHandler(t *testing.T) {
	std := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	h := WrapHTTPHandler(std)
	req := AcquireRequest()
	req.URL, _ = url.Parse("http://example.com/test")
	defer ReleaseRequest(req)

	rsp := h.Handle(req)
	if rsp == nil || rsp.StatusCode != 200 {
		t.Fatal("expected 200")
	}

	// Unwrap back to std.
	std2 := UnwrapHTTPHandler(NewPlaceholderHandler())
	rec := httptest.NewRecorder()
	std2.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestChain(t *testing.T) {
	var order []int
	h1 := HandlerFunc(func(req *Request) *Response {
		order = append(order, 1)
		return nil
	})
	h2 := HandlerFunc(func(req *Request) *Response {
		order = append(order, 2)
		return NewResponse()
	})
	Chain(h1, h2).Handle(AcquireRequest())
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestChainShortCircuit(t *testing.T) {
	h1 := HandlerFunc(func(req *Request) *Response {
		rsp := NewResponse()
		rsp.StatusCode = 403
		return rsp
	})
	h2 := HandlerFunc(func(req *Request) *Response {
		t.Fatal("should not reach second handler")
		return nil
	})
	rsp := Chain(h1, h2).Handle(AcquireRequest())
	if rsp == nil || rsp.StatusCode != 403 {
		t.Fatal("expected 403")
	}
}

func TestPlaceholderHandler(t *testing.T) {
	rsp := NewPlaceholderHandler().Handle(AcquireRequest())
	if rsp == nil || rsp.StatusCode != 200 {
		t.Fatal("expected 200")
	}
	if rsp.ContentType() != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", rsp.ContentType())
	}
}

// === Request ===

func TestRequestFromHTTP(t *testing.T) {
	r, _ := http.NewRequest("POST", "http://example.com/path?q=1", nil)
	r.Header.Set("X-Test", "value")
	r.RemoteAddr = "192.168.1.1:1234"

	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.FromHTTP(r)

	if req.Method != "POST" || req.Host != "example.com" {
		t.Fatal("basic fields mismatch")
	}
	if req.Var("$uri") != "/path" || req.Var("$args") != "q=1" {
		t.Fatal("nginx vars mismatch")
	}
	if req.Var("$scheme") != "http" {
		t.Fatalf("unexpected scheme: %s", req.Var("$scheme"))
	}
}

func TestRequestVars(t *testing.T) {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetVar("$custom", "hello")
	if req.Var("$custom") != "hello" || req.Var("$missing") != "" {
		t.Fatal("var mismatch")
	}
}

func TestRequestPathQuery(t *testing.T) {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.URL = &url.URL{Path: "/api", RawQuery: "x=1"}
	if req.Path() != "/api" || req.Query() != "x=1" {
		t.Fatal("path/query mismatch")
	}
}

func TestRequestReset(t *testing.T) {
	req := AcquireRequest()
	req.Method = "GET"
	req.SetVar("$test", "v")
	req.reset()
	if req.Method != "" || req.Var("$test") != "" {
		t.Fatal("reset failed")
	}
	ReleaseRequest(req)
}

// === Response ===

func TestResponseDefaults(t *testing.T) {
	rsp := NewResponse()
	if rsp.StatusCode != 200 {
		t.Fatal("expected 200")
	}
	ReleaseResponse(rsp)
}

func TestResponseWriteTo(t *testing.T) {
	rsp := AcquireResponse()
	defer ReleaseResponse(rsp)
	rsp.StatusCode = 201
	rsp.SetHeader("X-Test", "value")
	rsp.Body = []byte("hello")

	rec := httptest.NewRecorder()
	rsp.WriteTo(rec)
	if rec.Code != 201 || rec.Body.String() != "hello" {
		t.Fatal("write mismatch")
	}
}

func TestResponseReset(t *testing.T) {
	rsp := AcquireResponse()
	rsp.StatusCode = 404
	rsp.SetHeader("X-Test", "v")
	rsp.Body = []byte("data")
	rsp.reset()
	if rsp.StatusCode != 0 || rsp.Headers.Get("X-Test") != "" || len(rsp.Body) != 0 {
		t.Fatal("reset failed")
	}
	ReleaseResponse(rsp)
}

func TestResponseWriter(t *testing.T) {
	rw := AcquireResponseWriter()
	defer ReleaseResponseWriter(rw)
	rw.WriteHeader(202)
	rw.Header().Set("X-RW", "yes")
	rw.Write([]byte("body"))

	rsp := rw.ToResponse()
	if rsp.StatusCode != 202 || rsp.Headers.Get("X-RW") != "yes" || string(rsp.Body) != "body" {
		t.Fatal("response writer mismatch")
	}
}

// === Phase ===

func TestPhaseString(t *testing.T) {
	if PostReadPhase.String() != "post_read" || ContentPhase.String() != "content" || LogPhase.String() != "log" {
		t.Fatal("phase string mismatch")
	}
	if Phase(999).String() != "unknown" {
		t.Fatal("expected unknown")
	}
}

func TestDefaultPhaseEngine(t *testing.T) {
	pe := NewDefaultPhaseEngine()
	ph := &testPhaseHandler{phase: ContentPhase}
	if err := pe.Register(ph); err != nil {
		t.Fatalf("register: %v", err)
	}

	req := AcquireRequest()
	defer ReleaseRequest(req)
	ctx := &PhaseContext{Handler: NewPlaceholderHandler(), Request: req}
	rsp := pe.Run(ctx)
	if rsp == nil || rsp.StatusCode != 200 {
		t.Fatal("expected 200")
	}
}

func TestPhaseContextVars(t *testing.T) {
	ctx := &PhaseContext{}
	ctx.SetVar("$k", "v")
	if ctx.Var("$k") != "v" || ctx.Var("$missing") != "" {
		t.Fatal("phase context vars mismatch")
	}
}

// === Connection ===

func TestConnState(t *testing.T) {
	c := &Conn{}
	c.SetKeepAlive(false)
	if c.KeepAlive() {
		t.Fatal("expected keep-alive off")
	}
	c.IncrementRequestCount()
	if c.RequestCount() != 1 {
		t.Fatal("expected 1 request")
	}
	c.SetIdleTimeout(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	if !c.IsIdle() {
		t.Fatal("expected idle")
	}
	c.Touch()
	if c.IsIdle() {
		t.Fatal("expected not idle after touch")
	}
}

func TestSimpleConnPool(t *testing.T) {
	pool := NewSimpleConnPool(2)
	c, err := pool.Get()
	if err != nil || c == nil {
		t.Fatal("expected non-nil conn")
	}
	pool.Put(c)
	pool.Put(&Conn{})
	pool.Put(&Conn{}) // overflow
	if err := pool.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// testPhaseHandler is a test implementation.
type testPhaseHandler struct {
	phase Phase
}

func (h *testPhaseHandler) Phase() Phase { return h.phase }
func (h *testPhaseHandler) Handle(req *Request) *Response {
	return NewResponse()
}
