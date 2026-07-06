// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

// Request is the fully-resolved HTTP request handed to the transport [Doer] and
// threaded through the middleware stack. A [Connection] builds it from the merged
// options: the method, the absolute URL (scheme/host/port + path + encoded
// query), the outgoing headers (including any Basic-auth header), the body, and
// the per-request timeouts a host transport may honour.
type Request struct {
	// Method is the upper-case HTTP method ("GET", "POST", …).
	Method string
	// URL is the absolute request URL, query string included.
	URL string
	// Headers are the outgoing request headers.
	Headers *Headers
	// Body is the outgoing request body ("" for none).
	Body string
	// ReadTimeout is the response-read timeout in seconds (0 = unset).
	ReadTimeout int
	// WriteTimeout is the request-write timeout in seconds (0 = unset).
	WriteTimeout int
	// ConnectTimeout is the connection-open timeout in seconds (0 = unset).
	ConnectTimeout int
}
