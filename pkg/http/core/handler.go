package core

import (
	"net/http"
)

// Handler processes HTTP requests and produces responses.
type Handler interface {
	Handle(req *Request) *Response
}

// HandlerFunc is an adapter for ordinary functions.
type HandlerFunc func(*Request) *Response

// Handle calls f(req).
func (f HandlerFunc) Handle(req *Request) *Response { return f(req) }

// WrapHTTPHandler adapts a standard net/http.Handler.
func WrapHTTPHandler(h http.Handler) Handler {
	return HandlerFunc(func(req *Request) *Response {
		rw := AcquireResponseWriter()
		defer ReleaseResponseWriter(rw)
		h.ServeHTTP(rw, req.ToHTTP())
		return rw.ToResponse()
	})
}

// UnwrapHTTPHandler adapts a gonx Handler to net/http.Handler.
func UnwrapHTTPHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := AcquireRequest()
		req.FromHTTP(r)
		defer ReleaseRequest(req)
		rsp := h.Handle(req)
		if rsp == nil {
			http.NotFound(w, r)
			return
		}
		rsp.WriteTo(w)
	})
}

// Chain chains multiple handlers. Short-circuits on non-2xx.
func Chain(handlers ...Handler) Handler {
	return HandlerFunc(func(req *Request) *Response {
		for _, h := range handlers {
			rsp := h.Handle(req)
			if rsp != nil && (rsp.StatusCode < 200 || rsp.StatusCode >= 300) {
				return rsp
			}
		}
		return nil
	})
}

// PlaceholderHandler returns a fixed 200 OK response.
type PlaceholderHandler struct{}

// Handle returns a plain-text placeholder response.
func (h *PlaceholderHandler) Handle(req *Request) *Response {
	rsp := NewResponse()
	rsp.SetHeader("Content-Type", "text/plain; charset=utf-8")
	rsp.Body = []byte("gonx placeholder\n")
	return rsp
}

// NewPlaceholderHandler creates a no-op handler for testing.
func NewPlaceholderHandler() Handler { return &PlaceholderHandler{} }
