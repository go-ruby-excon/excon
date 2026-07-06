// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

// Handler is one step of the resolved request pipeline: it receives the
// [Request], does its work (typically calling the next handler), and returns the
// [Response] or an error. The innermost handler is the connection core (retry +
// transport + :expects check); each [Middleware] wraps it.
type Handler func(req *Request) (*Response, error)

// Middleware wraps a downstream [Handler] and returns a new one, mirroring an
// entry of Excon's :middlewares stack. The stack is composed outermost-first: a
// request-phase middleware acts before calling next, a response-phase middleware
// acts on the value next returns. Supply a stack via [Options.Middlewares]; the
// built-in retry (:idempotent) and status assertion (:expects) run inside it.
type Middleware func(next Handler) Handler

// RequestMiddleware builds a request-phase [Middleware]: fn runs before the
// downstream handler and may rewrite the request or abort by returning an error.
func RequestMiddleware(fn func(req *Request) error) Middleware {
	return func(next Handler) Handler {
		return func(req *Request) (*Response, error) {
			if err := fn(req); err != nil {
				return nil, err
			}
			return next(req)
		}
	}
}

// ResponseMiddleware builds a response-phase [Middleware]: fn runs on the
// [Response] the downstream handler returned (skipped when it errored) and may
// rewrite it or abort by returning an error.
func ResponseMiddleware(fn func(resp *Response) error) Middleware {
	return func(next Handler) Handler {
		return func(req *Request) (*Response, error) {
			resp, err := next(req)
			if err != nil {
				return nil, err
			}
			if err := fn(resp); err != nil {
				return nil, err
			}
			return resp, nil
		}
	}
}
