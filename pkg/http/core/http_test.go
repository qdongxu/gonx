package core

import (
	"strings"
	"testing"
)

func TestPlaceholderHandlerHandle(t *testing.T) {
	h := NewPlaceholderHandler()
	req := &Request{Method: "GET", Path: "/"}
	rsp := h.Handle(req)
	if rsp == nil {
		t.Fatal("expected non-nil Response")
	}
	if rsp.StatusCode != 200 {
		t.Fatalf("unexpected status: %d", rsp.StatusCode)
	}
	if rsp.Headers["Content-Type"] != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", rsp.Headers["Content-Type"])
	}
}

func TestNewResponseDefaults(t *testing.T) {
	rsp := NewResponse()
	if rsp.StatusCode != 200 {
		t.Fatalf("expected default status 200, got %d", rsp.StatusCode)
	}
	if rsp.Headers == nil {
		t.Fatal("expected non-nil Headers map")
	}
}

func TestRequestStructure(t *testing.T) {
	req := &Request{
		Method:  "POST",
		Path:    "/api/v1/test",
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	if req.Method != "POST" {
		t.Fatalf("unexpected method: %s", req.Method)
	}
	if req.Path != "/api/v1/test" {
		t.Fatalf("unexpected path: %s", req.Path)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Fatalf("unexpected header value: %s", req.Headers["Content-Type"])
	}
}

func TestResponseWithBody(t *testing.T) {
	body := strings.NewReader("hello")
	rsp := &Response{
		StatusCode: 201,
		Headers:    map[string]string{"X-Custom": "yes"},
		Body:       body,
	}
	if rsp.StatusCode != 201 {
		t.Fatalf("unexpected status: %d", rsp.StatusCode)
	}
	if rsp.Headers["X-Custom"] != "yes" {
		t.Fatalf("unexpected header: %s", rsp.Headers["X-Custom"])
	}
	if rsp.Body == nil {
		t.Fatal("expected non-nil Body")
	}
}
