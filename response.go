// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

// Response is the result of a request, mirroring Excon::Response: the status
// code, the body, the response headers, the reason phrase, and the remote IP the
// connection resolved to. It is produced by the transport [Doer] and returned
// unchanged (on a successful, expectation-satisfying request) to the caller.
type Response struct {
	status   int
	body     string
	headers  *Headers
	reason   string
	remoteIP string
}

// NewResponse builds a [Response] from its parts. Transports (including a test
// stub) use it to report a completed round-trip.
func NewResponse(status int, body string, headers *Headers, reason, remoteIP string) *Response {
	if headers == nil {
		headers = NewHeaders()
	}
	return &Response{status: status, body: body, headers: headers, reason: reason, remoteIP: remoteIP}
}

// Status returns the HTTP status code (Excon::Response#status).
func (r *Response) Status() int { return r.status }

// Body returns the response body (Excon::Response#body).
func (r *Response) Body() string { return r.body }

// Headers returns the response headers (Excon::Response#headers).
func (r *Response) Headers() *Headers { return r.headers }

// ReasonPhrase returns the response reason phrase (Excon::Response#reason_phrase).
func (r *Response) ReasonPhrase() string { return r.reason }

// RemoteIp returns the resolved remote IP address (Excon::Response#remote_ip).
func (r *Response) RemoteIp() string { return r.remoteIP }

// Success reports whether the status is a 2xx (Excon treats 2xx as success).
func (r *Response) Success() bool { return r.status >= 200 && r.status < 300 }
