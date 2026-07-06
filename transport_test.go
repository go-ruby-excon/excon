// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeHTTP is a stub net/http client that returns a canned response or error,
// exercising NetHTTPDoer's request-building and error mapping without a server.
type fakeHTTP struct {
	resp    *http.Response
	err     error
	gotReq  *http.Request
	gotBody string
}

func (f *fakeHTTP) Do(req *http.Request) (*http.Response, error) {
	f.gotReq = req
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		f.gotBody = string(b)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func mkResp(status int, statusLine, body string, header http.Header) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     statusLine,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNetHTTPOverHTTPTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Req") != "1" {
			t.Errorf("request header not sent: %q", r.Header.Get("X-Req"))
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != "payload" {
			t.Errorf("body = %q", b)
		}
		w.Header().Set("X-Multi", "a")
		w.Header().Add("X-Multi", "b")
		w.WriteHeader(200)
		io.WriteString(w, "hello")
	}))
	defer srv.Close()

	c := New(srv.URL) // default NetHTTP transport, real loopback round-trip
	resp, err := c.Post(Options{
		Body:    "payload",
		Headers: HeadersOf([2]string{"X-Req", "1"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status() != 200 || resp.Body() != "hello" || resp.ReasonPhrase() != "OK" {
		t.Fatalf("response = %+v", resp)
	}
	if v, _ := resp.Headers().Get("X-Multi"); v != "a, b" {
		t.Fatalf("multi-header join = %q", v)
	}
	// httptrace captured the loopback peer IP.
	if ip := resp.RemoteIp(); ip != "127.0.0.1" && ip != "::1" {
		t.Fatalf("remote ip = %q", ip)
	}
}

func TestNetHTTPEmptyBodyNoPhrase(t *testing.T) {
	fh := &fakeHTTP{resp: mkResp(200, "200", "", http.Header{})}
	d := NetHTTP()
	d.Client = fh
	req := &Request{Method: "GET", URL: "https://h/x", Headers: NewHeaders()}
	resp, err := d.Call(req)
	if err != nil {
		t.Fatal(err)
	}
	if fh.gotReq.Body != nil {
		t.Fatal("empty body should produce a nil request body")
	}
	if resp.ReasonPhrase() != "" {
		t.Fatalf("reason should be empty, got %q", resp.ReasonPhrase())
	}
	if resp.RemoteIp() != "" {
		t.Fatalf("no trace fired: remote ip should be empty, got %q", resp.RemoteIp())
	}
}

func TestNetHTTPNewRequestError(t *testing.T) {
	d := &NetHTTPDoer{Client: &fakeHTTP{}}
	req := &Request{Method: "BAD METHOD", URL: "https://h/x", Headers: NewHeaders()}
	if _, err := d.Call(req); !IsSocketError(err) {
		t.Fatalf("expected Socket error, got %v", err)
	}
}

// netTimeout is a net-style timeout error.
type netTimeout struct{ to bool }

func (e netTimeout) Error() string { return "i/o timeout" }
func (e netTimeout) Timeout() bool { return e.to }

func TestNetHTTPTransportErrors(t *testing.T) {
	req := &Request{Method: "GET", URL: "https://h", Headers: NewHeaders()}
	// Timeout → Timeout error.
	d := &NetHTTPDoer{Client: &fakeHTTP{err: netTimeout{to: true}}}
	if _, err := d.Call(req); !IsTimeout(err) {
		t.Fatalf("expected Timeout, got %v", err)
	}
	// net.Error with Timeout()==false → Socket.
	d = &NetHTTPDoer{Client: &fakeHTTP{err: netTimeout{to: false}}}
	if _, err := d.Call(req); !IsSocketError(err) {
		t.Fatalf("expected Socket for non-timeout net.Error, got %v", err)
	}
	// Plain error (no Timeout method) → Socket.
	d = &NetHTTPDoer{Client: &fakeHTTP{err: errors.New("boom")}}
	if _, err := d.Call(req); !IsSocketError(err) {
		t.Fatalf("expected Socket for plain error, got %v", err)
	}
}

// errBody is a response body whose Read fails, exercising the ReadAll error path.
type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errBody) Close() error             { return nil }

func TestNetHTTPReadError(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{}, Body: errBody{}}
	d := &NetHTTPDoer{Client: &fakeHTTP{resp: resp}}
	req := &Request{Method: "GET", URL: "https://h", Headers: NewHeaders()}
	if _, err := d.Call(req); !IsSocketError(err) {
		t.Fatalf("expected Socket on read error, got %v", err)
	}
}

func TestHostOnly(t *testing.T) {
	if got := hostOnly(""); got != "" {
		t.Fatalf("hostOnly empty = %q", got)
	}
	if got := hostOnly("10.0.0.1:443"); got != "10.0.0.1" {
		t.Fatalf("hostOnly host:port = %q", got)
	}
	if got := hostOnly("bare-address"); got != "bare-address" {
		t.Fatalf("hostOnly bare = %q", got)
	}
}
