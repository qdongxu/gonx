package core

import (
	"bytes"
	"net/http"
	"sync"
)

var responsePool = sync.Pool{
	New: func() interface{} {
		return &Response{Headers: make(http.Header)}
	},
}

// AcquireResponse gets a Response from the pool.
func AcquireResponse() *Response {
	return responsePool.Get().(*Response)
}

// ReleaseResponse returns a Response to the pool.
func ReleaseResponse(rsp *Response) {
	rsp.reset()
	responsePool.Put(rsp)
}

// Response wraps an HTTP response with enhanced header control.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewResponse creates a Response with 200 OK defaults.
func NewResponse() *Response {
	rsp := AcquireResponse()
	rsp.StatusCode = 200
	return rsp
}

// SetHeader sets a response header.
func (rsp *Response) SetHeader(key, value string) {
	rsp.Headers.Set(key, value)
}

// WriteTo writes the response to an http.ResponseWriter.
func (rsp *Response) WriteTo(w http.ResponseWriter) {
	if rsp.StatusCode == 0 {
		rsp.StatusCode = 200
	}
	w.WriteHeader(rsp.StatusCode)
	for k, vv := range rsp.Headers {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	if len(rsp.Body) > 0 {
		w.Write(rsp.Body)
	}
}

// ContentType returns the Content-Type header.
func (rsp *Response) ContentType() string {
	return rsp.Headers.Get("Content-Type")
}

func (rsp *Response) reset() {
	rsp.StatusCode = 0
	for k := range rsp.Headers {
		delete(rsp.Headers, k)
	}
	rsp.Body = nil
}

var responseWriterPool = sync.Pool{
	New: func() interface{} {
		return &ResponseWriter{header: make(http.Header)}
	},
}

// AcquireResponseWriter gets a ResponseWriter from the pool.
func AcquireResponseWriter() *ResponseWriter {
	return responseWriterPool.Get().(*ResponseWriter)
}

// ReleaseResponseWriter returns a ResponseWriter to the pool.
func ReleaseResponseWriter(rw *ResponseWriter) {
	rw.reset()
	responseWriterPool.Put(rw)
}

// ResponseWriter adapts http.ResponseWriter to capture the response.
type ResponseWriter struct {
	statusCode int
	header     http.Header
	body       bytes.Buffer
	written    bool
}

// Header implements http.ResponseWriter.
func (rw *ResponseWriter) Header() http.Header { return rw.header }

// WriteHeader implements http.ResponseWriter.
func (rw *ResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
}

// Write implements http.ResponseWriter.
func (rw *ResponseWriter) Write(p []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.body.Write(p)
}

// ToResponse converts captured response to a gonx Response.
func (rw *ResponseWriter) ToResponse() *Response {
	rsp := AcquireResponse()
	rsp.StatusCode = rw.statusCode
	if rsp.StatusCode == 0 {
		rsp.StatusCode = 200
	}
	for k, vv := range rw.header {
		for _, v := range vv {
			rsp.Headers.Add(k, v)
		}
	}
	rsp.Body = rw.body.Bytes()
	return rsp
}

func (rw *ResponseWriter) reset() {
	rw.statusCode = 0
	for k := range rw.header {
		delete(rw.header, k)
	}
	rw.body.Reset()
	rw.written = false
}
