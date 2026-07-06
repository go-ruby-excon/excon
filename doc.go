// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package excon is a pure-Go (CGO-free) reimplementation of the deterministic
// core of Ruby's `excon` gem — the fast, persistent HTTP client. It models a
// reusable [Connection] bound to a base URL and default options, the one-shot
// verb helpers ([Get], [Post], …), the option handling Excon performs around the
// wire (path/query building, header merging, Basic auth, the :expects status
// assertion, and :idempotent retry), the [Response] (Status/Body/Headers/
// RemoteIp/ReasonPhrase), and the full Excon::Error tree.
//
// # What it is — and isn't
//
// Everything Excon does around the socket is deterministic and needs no
// interpreter, so it lives here as pure Go: merging per-request options over the
// connection defaults, building the absolute URL (path plus an order-preserving,
// CGI-escaped query string), adding the Basic Authorization header, asserting the
// response status against :expects (raising the matching status error), and
// retrying an :idempotent request on a transport failure. The HTTP round-trip
// itself is a host seam: the [Doer] transport performs it. The default is
// [NetHTTP], backed by net/http (it also captures the remote IP via httptrace);
// tests inject a [DoerFunc] stub or drive [NetHTTP] over an httptest server, and
// a host (go-embedded-ruby / rbgo) wires the real transport.
//
// # Flow
//
//	conn := excon.New("https://api.example.com", excon.Options{
//		Headers: excon.HeadersOf([2]string{"Accept", "application/json"}),
//	})
//
//	resp, err := conn.Get(excon.Options{
//		Path:    "/widgets",
//		Query:   excon.QueryOf([2]string{"q", "gadget"}),
//		Expects: []int{200},
//	})
//	if err != nil { /* an excon.Error: NotFound, InternalServerError, Timeout, … */ }
//	_ = resp.Status()       // 200
//	_ = resp.Body()         // response body
//	_ = resp.RemoteIp()     // resolved peer IP
//	_ = resp.Success()      // true for 2xx
//
//	// one-shot form
//	resp, err = excon.Post("https://api.example.com/widgets",
//		excon.Options{Body: `{"name":"gadget"}`, Expects: []int{201}})
//
// # Errors
//
// An :expects mismatch raises a status [Error] whose Kind names the Excon
// subclass for the code (404 → [KindNotFound], 500 → [KindInternalServerError],
// an unmapped 4xx → [KindClient], …); a transport failure raises [KindSocket] or
// [KindTimeout]. The whole tree matches with errors.Is via the sentinels: e.g.
// errors.Is(err, excon.ErrClient) matches any 4xx, errors.Is(err,
// excon.ErrHTTPStatus) any status error, errors.Is(err, excon.ErrError) any excon
// error — mirroring Ruby's rescue of a superclass. Helpers [IsClientError],
// [IsServerError], [IsHTTPStatusError], [IsSocketError] and [IsTimeout] wrap the
// common checks.
//
// # Value model
//
// Query params are an ordered [Query] (CGI-escaped like Excon::Utils.query_string);
// headers a case-insensitive ordered [Headers]; the body a string. A host maps
// its Ruby Excon::Connection / Response objects to and from these shapes.
package excon
