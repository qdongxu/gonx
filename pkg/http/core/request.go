package core

import (
	"net"
	"net/http"
	"net/url"
	"sync"
)

var requestPool = sync.Pool{
	New: func() interface{} {
		return &Request{Headers: make(http.Header), Vars: make(map[string]string)}
	},
}

// AcquireRequest gets a Request from the pool.
func AcquireRequest() *Request {
	return requestPool.Get().(*Request)
}

// ReleaseRequest returns a Request to the pool.
func ReleaseRequest(req *Request) {
	req.reset()
	requestPool.Put(req)
}

// Request wraps an HTTP request with nginx-style variable access.
type Request struct {
	Method        string
	URL           *url.URL
	Proto         string
	Headers       http.Header
	Body          []byte
	ContentLength int64
	Host          string
	RemoteAddr    string
	Conn          net.Conn
	Vars          map[string]string
	Raw           *http.Request
}

// FromHTTP populates the Request from a standard http.Request.
func (req *Request) FromHTTP(r *http.Request) {
	req.Method = r.Method
	req.URL = r.URL
	req.Proto = r.Proto
	req.Host = r.Host
	req.RemoteAddr = r.RemoteAddr
	req.Raw = r
	for k, v := range r.Header {
		req.Headers[k] = v
	}
	req.ContentLength = r.ContentLength
	if req.Vars == nil {
		req.Vars = make(map[string]string)
	}
	req.Vars["$uri"] = r.URL.Path
	req.Vars["$request_uri"] = r.URL.RequestURI()
	req.Vars["$args"] = r.URL.RawQuery
	req.Vars["$host"] = r.Host
	req.Vars["$remote_addr"] = r.RemoteAddr
	req.Vars["$request_method"] = r.Method
	req.Vars["$scheme"] = r.URL.Scheme
	if req.Vars["$scheme"] == "" {
		req.Vars["$scheme"] = "http"
	}
}

// ToHTTP converts back to a standard http.Request.
func (req *Request) ToHTTP() *http.Request {
	if req.Raw != nil {
		return req.Raw
	}
	r, _ := http.NewRequest(req.Method, req.URL.String(), nil)
	r.Header = req.Headers
	r.Host = req.Host
	r.RemoteAddr = req.RemoteAddr
	return r
}

// Var returns a nginx-style variable.
func (req *Request) Var(name string) string {
	if req.Vars == nil {
		return ""
	}
	return req.Vars[name]
}

// SetVar sets a nginx-style variable.
func (req *Request) SetVar(name, value string) {
	if req.Vars == nil {
		req.Vars = make(map[string]string)
	}
	req.Vars[name] = value
}

// Path returns the request path.
func (req *Request) Path() string {
	if req.URL != nil {
		return req.URL.Path
	}
	return req.Var("$uri")
}

// Query returns the raw query string.
func (req *Request) Query() string {
	if req.URL != nil {
		return req.URL.RawQuery
	}
	return req.Var("$args")
}

func (req *Request) reset() {
	req.Method = ""
	req.URL = nil
	req.Proto = ""
	for k := range req.Headers {
		delete(req.Headers, k)
	}
	req.Body = nil
	req.ContentLength = 0
	req.Host = ""
	req.RemoteAddr = ""
	req.Conn = nil
	for k := range req.Vars {
		delete(req.Vars, k)
	}
	req.Raw = nil
}
