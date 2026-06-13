// Package core defines the HTTP server core interfaces.
// Phase 0: skeleton only; implementations deferred to Phase 1–2.
package core

import (
	"io"
	"net"
)

// Handler processes HTTP requests and returns responses.
// Implementations may be static file handlers, proxy handlers, or Lua/WASM
// extensions (not in scope for Phase 0).
type Handler interface {
	Handle(req *Request) *Response
}

// Request represents an HTTP request. Phase 0: minimal placeholder.
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    io.Reader
	Conn    net.Conn // underlying connection (for close, keepalive, etc.)
}

// Response represents an HTTP response. Phase 0: minimal placeholder.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       io.Reader
}

// NewResponse creates a Response with sensible defaults.
func NewResponse() *Response {
	return &Response{
		StatusCode: 200,
		Headers:    make(map[string]string),
	}
}

// PlaceholderHandler implements Handler with a fixed 200 OK response.
// Used for interface wiring tests in Phase 0.
type PlaceholderHandler struct{}

// Handle returns a plain-text placeholder response.
func (h *PlaceholderHandler) Handle(req *Request) *Response {
	rsp := NewResponse()
	rsp.Headers["Content-Type"] = "text/plain; charset=utf-8"
	return rsp
}

// NewPlaceholderHandler creates a no-op handler for testing.
func NewPlaceholderHandler() Handler {
	return &PlaceholderHandler{}
}
