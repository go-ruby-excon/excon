// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

// The one-shot verb helpers mirror Excon.get/post/put/delete/head/patch(url,
// opts): each builds a throwaway [Connection] for the URL and issues a single
// request with the given options and method. For repeated calls to the same
// host, build a persistent [Connection] with [New] and reuse it.

// Get issues a one-shot GET request (Excon.get).
func Get(rawurl string, opts ...Options) (*Response, error) { return oneShot("GET", rawurl, opts) }

// Head issues a one-shot HEAD request (Excon.head).
func Head(rawurl string, opts ...Options) (*Response, error) { return oneShot("HEAD", rawurl, opts) }

// Delete issues a one-shot DELETE request (Excon.delete).
func Delete(rawurl string, opts ...Options) (*Response, error) {
	return oneShot("DELETE", rawurl, opts)
}

// Post issues a one-shot POST request (Excon.post).
func Post(rawurl string, opts ...Options) (*Response, error) { return oneShot("POST", rawurl, opts) }

// Put issues a one-shot PUT request (Excon.put).
func Put(rawurl string, opts ...Options) (*Response, error) { return oneShot("PUT", rawurl, opts) }

// Patch issues a one-shot PATCH request (Excon.patch).
func Patch(rawurl string, opts ...Options) (*Response, error) { return oneShot("PATCH", rawurl, opts) }

// oneShot builds a connection for rawurl and issues a single request with the
// given method and options.
func oneShot(method, rawurl string, opts []Options) (*Response, error) {
	o := firstOpt(opts)
	o.Method = method
	return New(rawurl).Request(o)
}
