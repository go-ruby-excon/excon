// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"net/url"
	"strings"
	"time"
)

// defaultRetryLimit is Excon's default :retry_limit for idempotent requests.
const defaultRetryLimit = 4

// sleep is the delay used between idempotent retries; a package variable so tests
// can replace it with a no-op instead of really waiting.
var sleep = time.Sleep

// Options configures a request (and, on [New], the connection defaults),
// mirroring the option hash Excon.new / Excon.get accept. A per-request Options
// is merged over the connection's defaults: :headers deep-merge, every other
// present field overrides. A zero field is treated as unset and inherits the
// connection default (so, e.g., an empty Body keeps the connection's).
type Options struct {
	// Method is the HTTP method; defaults to "GET" when unset.
	Method string
	// Path overrides the request path (else the connection/URL path is used).
	Path string
	// Query is the URL query (:query); when non-nil it replaces the default.
	Query *Query
	// Headers are request headers (:headers); deep-merged over the defaults.
	Headers *Headers
	// Body is the request body (:body).
	Body string
	// Expects asserts the response status: a status not listed raises the
	// matching status [Error]. An empty/nil Expects performs no assertion.
	Expects []int
	// Idempotent enables automatic retry of transport failures (:idempotent).
	Idempotent bool
	// RetryLimit is the maximum number of attempts for an idempotent request
	// (:retry_limit); defaults to 4 on a connection.
	RetryLimit int
	// RetryInterval is the delay between idempotent retries, in milliseconds
	// (:retry_interval).
	RetryInterval int
	// ReadTimeout/WriteTimeout/ConnectTimeout are per-request timeouts, in
	// seconds, that a host transport may honour.
	ReadTimeout    int
	WriteTimeout   int
	ConnectTimeout int
	// User/Password set HTTP Basic credentials (:user/:password): a Basic
	// Authorization header is added unless one is already present.
	User     string
	Password string
	// Middlewares is the outer middleware stack (:middlewares) wrapped around the
	// built-in retry/expects core; when non-nil it replaces the default (none).
	Middlewares []Middleware
}

// Connection is a persistent, reusable HTTP client bound to a base URL and a set
// of default options, mirroring Excon::Connection. Build one with [New]; issue
// requests with [Connection.Request] or the verb helpers.
type Connection struct {
	scheme    string
	hostPort  string
	base      Options
	transport Doer
}

// New builds a [Connection] for a base URL, mirroring Excon.new(url, opts). The
// URL's scheme, host, port, path, query and userinfo seed the connection
// defaults; the optional opts override them. The connection defaults to the
// net/http transport ([NetHTTP]); tests override it with [Connection.Transport].
func New(rawurl string, opts ...Options) *Connection {
	base := firstOpt(opts)
	c := &Connection{transport: NetHTTP()}
	if u, err := url.Parse(rawurl); err == nil {
		c.scheme = u.Scheme
		c.hostPort = u.Host
		if base.Path == "" {
			base.Path = u.Path
		}
		if base.Query == nil && u.RawQuery != "" {
			base.Query = parseRawQuery(u.RawQuery)
		}
		if u.User != nil {
			if base.User == "" {
				base.User = u.User.Username()
			}
			if base.Password == "" {
				pw, _ := u.User.Password()
				base.Password = pw
			}
		}
	}
	if base.RetryLimit == 0 {
		base.RetryLimit = defaultRetryLimit
	}
	c.base = base
	return c
}

// Transport sets the connection's transport [Doer] (the host seam) and returns
// the connection for chaining. Tests inject a [DoerFunc] stub; rbgo wires the
// real transport.
func (c *Connection) Transport(d Doer) *Connection {
	c.transport = d
	return c
}

// Request issues a request, merging the given per-request options over the
// connection defaults, mirroring Excon::Connection#request. It builds the
// [Request], runs it through the middleware stack around the retry/transport/
// :expects core, and returns the [Response] (or an [Error]).
func (c *Connection) Request(opts ...Options) (*Response, error) {
	o := c.base.merge(firstOpt(opts))
	if o.Method == "" {
		o.Method = "GET"
	}

	headers := NewHeaders()
	if o.Headers != nil {
		headers = o.Headers.Clone()
	}
	if (o.User != "" || o.Password != "") && !headers.Has("Authorization") {
		headers.Set("Authorization", BasicHeaderFrom(o.User, o.Password))
	}

	req := &Request{
		Method:         strings.ToUpper(o.Method),
		URL:            c.buildURL(o),
		Headers:        headers,
		Body:           o.Body,
		ReadTimeout:    o.ReadTimeout,
		WriteTimeout:   o.WriteTimeout,
		ConnectTimeout: o.ConnectTimeout,
	}

	handler := c.core(o)
	for i := len(o.Middlewares) - 1; i >= 0; i-- {
		handler = o.Middlewares[i](handler)
	}
	return handler(req)
}

// Get issues a GET request (Excon::Connection#get).
func (c *Connection) Get(opts ...Options) (*Response, error) { return c.verb("GET", opts) }

// Head issues a HEAD request (Excon::Connection#head).
func (c *Connection) Head(opts ...Options) (*Response, error) { return c.verb("HEAD", opts) }

// Delete issues a DELETE request (Excon::Connection#delete).
func (c *Connection) Delete(opts ...Options) (*Response, error) { return c.verb("DELETE", opts) }

// Post issues a POST request (Excon::Connection#post).
func (c *Connection) Post(opts ...Options) (*Response, error) { return c.verb("POST", opts) }

// Put issues a PUT request (Excon::Connection#put).
func (c *Connection) Put(opts ...Options) (*Response, error) { return c.verb("PUT", opts) }

// Patch issues a PATCH request (Excon::Connection#patch).
func (c *Connection) Patch(opts ...Options) (*Response, error) { return c.verb("PATCH", opts) }

// verb sets the method on the (optional) options and issues the request.
func (c *Connection) verb(method string, opts []Options) (*Response, error) {
	o := firstOpt(opts)
	o.Method = method
	return c.Request(o)
}

// core is the innermost [Handler]: it runs the transport with idempotent retry,
// then asserts the :expects status, mirroring the roles of Excon's Idempotent and
// Expects middleware.
func (c *Connection) core(o Options) Handler {
	return func(req *Request) (*Response, error) {
		resp, err := c.withRetry(o, req)
		if err != nil {
			return nil, err
		}
		if err := checkExpects(o, req, resp); err != nil {
			return nil, err
		}
		return resp, nil
	}
}

// withRetry calls the transport, retrying a retryable transport failure up to
// RetryLimit total attempts when the request is idempotent, sleeping
// RetryInterval between attempts.
func (c *Connection) withRetry(o Options, req *Request) (*Response, error) {
	attempt := 0
	for {
		resp, err := c.transport.Call(req)
		if err == nil {
			return resp, nil
		}
		attempt++
		if !o.Idempotent || !retryable(err) || attempt >= o.RetryLimit {
			return nil, err
		}
		if o.RetryInterval > 0 {
			sleep(time.Duration(o.RetryInterval) * time.Millisecond)
		}
	}
}

// buildURL assembles the absolute request URL from the connection's scheme/host
// and the merged path and query.
func (c *Connection) buildURL(o Options) string {
	path := o.Path
	if path == "" {
		path = "/"
	}
	u := c.scheme + "://" + c.hostPort + path
	if o.Query != nil {
		u += o.Query.Encode()
	}
	return u
}

// merge overlays the per-request options o onto the connection defaults c and
// returns the effective options: :headers deep-merge, :idempotent OR-combines,
// and every other present (non-zero) field of o overrides the default.
func (c Options) merge(o Options) Options {
	out := c
	if o.Method != "" {
		out.Method = o.Method
	}
	if o.Path != "" {
		out.Path = o.Path
	}
	if o.Query != nil {
		out.Query = o.Query
	}
	out.Headers = mergeHeaders(c.Headers, o.Headers)
	if o.Body != "" {
		out.Body = o.Body
	}
	if o.Expects != nil {
		out.Expects = o.Expects
	}
	out.Idempotent = c.Idempotent || o.Idempotent
	if o.RetryLimit > 0 {
		out.RetryLimit = o.RetryLimit
	}
	if o.RetryInterval > 0 {
		out.RetryInterval = o.RetryInterval
	}
	if o.ReadTimeout > 0 {
		out.ReadTimeout = o.ReadTimeout
	}
	if o.WriteTimeout > 0 {
		out.WriteTimeout = o.WriteTimeout
	}
	if o.ConnectTimeout > 0 {
		out.ConnectTimeout = o.ConnectTimeout
	}
	if o.User != "" {
		out.User = o.User
	}
	if o.Password != "" {
		out.Password = o.Password
	}
	if o.Middlewares != nil {
		out.Middlewares = o.Middlewares
	}
	return out
}

// checkExpects asserts that resp's status is one of o.Expects, returning the
// matching status [Error] otherwise. An empty Expects performs no assertion.
func checkExpects(o Options, req *Request, resp *Response) error {
	if len(o.Expects) == 0 {
		return nil
	}
	for _, s := range o.Expects {
		if s == resp.status {
			return nil
		}
	}
	return newStatusError(o.Expects, req, resp)
}

// mergeHeaders returns base's headers overlaid by extra's (both may be nil).
func mergeHeaders(base, extra *Headers) *Headers {
	if base == nil {
		base = NewHeaders()
	}
	return base.Merge(extra)
}

// parseRawQuery decodes a raw URL query (from the base URL) into an ordered
// [Query], unescaping each key and value and preserving order.
func parseRawQuery(raw string) *Query {
	q := NewQuery()
	for _, seg := range strings.Split(raw, "&") {
		if seg == "" {
			continue
		}
		k, v, hasVal := strings.Cut(seg, "=")
		if hasVal {
			q.Add(Unescape(k), Unescape(v))
		} else {
			q.AddKey(Unescape(k))
		}
	}
	return q
}

// firstOpt returns the first optional Options or a zero value.
func firstOpt(opts []Options) Options {
	if len(opts) > 0 {
		return opts[0]
	}
	return Options{}
}
