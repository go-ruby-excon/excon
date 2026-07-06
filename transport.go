// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
)

// Doer is the transport host seam: given a fully-resolved [Request], it performs
// the HTTP round-trip and returns the [Response], or a transport [Error]
// (Socket/Timeout). The core opens no socket itself — every request runs through
// whatever Doer the connection is set to, so tests inject a stub (see [DoerFunc]
// or drive [NetHTTP] over an httptest server) and rbgo wires the real transport.
type Doer interface {
	Call(req *Request) (*Response, error)
}

// DoerFunc adapts a function to the [Doer] interface, the convenient way to
// inject a stub transport in tests or a custom one in a host.
type DoerFunc func(req *Request) (*Response, error)

// Call invokes f(req).
func (f DoerFunc) Call(req *Request) (*Response, error) { return f(req) }

// httpClient is the minimal net/http surface [NetHTTPDoer] depends on, indirected
// so tests can drive the adapter's request-building and error mapping without a
// live server.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NetHTTPDoer is the default [Doer]: it turns a [Request] into a net/http
// request, executes it with its Client, captures the remote IP via httptrace,
// and maps the response (or a transport failure) into a [Response] or a transport
// [Error]. A timeout maps to [KindTimeout]; any other transport failure to
// [KindSocket].
type NetHTTPDoer struct {
	// Client performs the request; defaults to http.DefaultClient.
	Client httpClient
}

// NetHTTP returns the default net/http-backed [Doer].
func NetHTTP() *NetHTTPDoer { return &NetHTTPDoer{Client: http.DefaultClient} }

// Call performs the HTTP round-trip for req with net/http.
func (d *NetHTTPDoer) Call(req *Request) (*Response, error) {
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}
	hr, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return nil, newTransportError(KindSocket, err)
	}
	for _, p := range req.Headers.Pairs() {
		hr.Header.Set(p.Key, p.Val)
	}

	var remoteAddr string
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			remoteAddr = info.Conn.RemoteAddr().String()
		},
	}
	hr = hr.WithContext(httptrace.WithClientTrace(hr.Context(), trace))

	resp, err := d.Client.Do(hr)
	if err != nil {
		return nil, newTransportError(classifyTransport(err), err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newTransportError(KindSocket, err)
	}

	return &Response{
		status:   resp.StatusCode,
		body:     string(raw),
		headers:  headersFromHTTP(resp.Header),
		reason:   reasonPhrase(resp.Status),
		remoteIP: hostOnly(remoteAddr),
	}, nil
}

// timeoutError is the net-error surface used to detect a timeout.
type timeoutError interface{ Timeout() bool }

// classifyTransport maps a net/http client error to the Excon transport kind: a
// timeout becomes [KindTimeout], anything else [KindSocket].
func classifyTransport(err error) ErrorKind {
	if te, ok := err.(timeoutError); ok && te.Timeout() {
		return KindTimeout
	}
	return KindSocket
}

// headersFromHTTP converts an http.Header into an Excon [Headers], joining
// multi-valued headers with ", ".
func headersFromHTTP(h http.Header) *Headers {
	out := NewHeaders()
	for k, vs := range h {
		out.Set(k, strings.Join(vs, ", "))
	}
	return out
}

// reasonPhrase extracts the reason phrase from an http.Response.Status like
// "200 OK" → "OK". A status with no phrase yields "".
func reasonPhrase(status string) string {
	if _, phrase, found := strings.Cut(status, " "); found {
		return phrase
	}
	return ""
}

// hostOnly returns the host portion of a "host:port" remote address, the whole
// string when it has no port, or "" when the address is empty (the connection's
// remote IP was never observed).
func hostOnly(addr string) string {
	if addr == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}
