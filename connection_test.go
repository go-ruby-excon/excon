// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewURLParsing(t *testing.T) {
	// Full URL with path, query and userinfo seeds the connection defaults.
	var rec *Request
	c := New("https://alice:secret@h/base?seed=1&flag").Transport(stub(okResp(200), &rec))
	if c.base.RetryLimit != defaultRetryLimit {
		t.Fatalf("default retry limit = %d", c.base.RetryLimit)
	}
	if _, err := c.Get(); err != nil {
		t.Fatal(err)
	}
	// Path and query came from the URL; userinfo produced a Basic auth header.
	if rec.URL != "https://h/base?seed=1&flag" {
		t.Fatalf("url = %q", rec.URL)
	}
	if v, _ := rec.Headers.Get("Authorization"); v != BasicHeaderFrom("alice", "secret") {
		t.Fatalf("userinfo auth = %q", v)
	}

	// Options override URL-derived path/query/user, and set an explicit retry limit.
	c = New("https://bob@h/base?seed=1", Options{
		Path:       "/override",
		Query:      QueryOf([2]string{"k", "v"}),
		User:       "carol",
		Password:   "pw",
		RetryLimit: 9,
	}).Transport(stub(okResp(200), &rec))
	if c.base.RetryLimit != 9 {
		t.Fatalf("explicit retry limit = %d", c.base.RetryLimit)
	}
	if _, err := c.Get(); err != nil {
		t.Fatal(err)
	}
	if rec.URL != "https://h/override?k=v" {
		t.Fatalf("overridden url = %q", rec.URL)
	}
	if v, _ := rec.Headers.Get("Authorization"); v != BasicHeaderFrom("carol", "pw") {
		t.Fatalf("overridden auth = %q", v)
	}

	// URL with userinfo user only (no password), base.User empty.
	c = New("http://dave@h/").Transport(stub(okResp(200), &rec))
	if c.base.User != "dave" || c.base.Password != "" {
		t.Fatalf("userinfo w/o password: user=%q pass=%q", c.base.User, c.base.Password)
	}

	// Unparsable URL: the parse block is skipped, defaults still applied.
	c = New("http://a\x7fb")
	if c.scheme != "" || c.hostPort != "" || c.base.RetryLimit != defaultRetryLimit {
		t.Fatalf("bad url should leave scheme/host empty: %q %q", c.scheme, c.hostPort)
	}
}

func TestParseRawQuery(t *testing.T) {
	// Empty segment skipped; valued and bare keys both parsed and unescaped.
	q := parseRawQuery("a=hello%20world&&flag&b=x%26y")
	if q.Len() != 3 {
		t.Fatalf("len = %d", q.Len())
	}
	if q.Pairs()[0].Val != "hello world" || !q.Pairs()[0].HasVal {
		t.Fatalf("valued pair = %+v", q.Pairs()[0])
	}
	if q.Pairs()[1].Key != "flag" || q.Pairs()[1].HasVal {
		t.Fatalf("bare key pair = %+v", q.Pairs()[1])
	}
	if q.Pairs()[2].Val != "x&y" {
		t.Fatalf("escaped value = %q", q.Pairs()[2].Val)
	}
}

func TestMergeInheritsAllDefaults(t *testing.T) {
	// A connection with every field set; an empty per-request Options inherits all.
	base := Options{
		Method:         "PUT",
		Path:           "/p",
		Query:          QueryOf([2]string{"a", "1"}),
		Headers:        HeadersOf([2]string{"X-Base", "1"}),
		Body:           "b",
		Expects:        []int{200},
		Idempotent:     true,
		RetryLimit:     7,
		RetryInterval:  50,
		ReadTimeout:    1,
		WriteTimeout:   2,
		ConnectTimeout: 3,
		User:           "u",
		Password:       "p",
		Middlewares:    []Middleware{RequestMiddleware(func(*Request) error { return nil })},
	}
	got := base.merge(Options{})
	if got.Method != "PUT" || got.Path != "/p" || got.Body != "b" ||
		got.RetryLimit != 7 || got.RetryInterval != 50 ||
		got.ReadTimeout != 1 || got.WriteTimeout != 2 || got.ConnectTimeout != 3 ||
		got.User != "u" || got.Password != "p" || !got.Idempotent ||
		got.Query == nil || len(got.Expects) != 1 || got.Middlewares == nil {
		t.Fatalf("empty override should inherit everything: %+v", got)
	}

	// A per-request Options with every field set overrides all of them.
	over := Options{
		Method:         "PATCH",
		Path:           "/q",
		Query:          QueryOf([2]string{"b", "2"}),
		Headers:        HeadersOf([2]string{"X-Over", "9"}),
		Body:           "c",
		Expects:        []int{201},
		Idempotent:     true,
		RetryLimit:     3,
		RetryInterval:  10,
		ReadTimeout:    4,
		WriteTimeout:   5,
		ConnectTimeout: 6,
		User:           "x",
		Password:       "y",
		Middlewares:    []Middleware{},
	}
	g2 := base.merge(over)
	if g2.Method != "PATCH" || g2.Path != "/q" || g2.Body != "c" ||
		g2.RetryLimit != 3 || g2.RetryInterval != 10 ||
		g2.ReadTimeout != 4 || g2.WriteTimeout != 5 || g2.ConnectTimeout != 6 ||
		g2.User != "x" || g2.Password != "y" || g2.Expects[0] != 201 {
		t.Fatalf("override should replace everything: %+v", g2)
	}
	// Headers deep-merge keeps both base and override entries.
	if v, _ := g2.Headers.Get("X-Base"); v != "1" {
		t.Fatalf("base header lost in merge: %q", v)
	}
	if v, _ := g2.Headers.Get("X-Over"); v != "9" {
		t.Fatalf("override header missing: %q", v)
	}
	// Query replaced (not merged).
	if v := g2.Query.Pairs()[0].Key; v != "b" {
		t.Fatalf("query should be replaced: %q", v)
	}
}

func TestOneShotVerbs(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(200)
	}))
	defer srv.Close()

	verbs := []struct {
		call func(string, ...Options) (*Response, error)
		want string
	}{
		{Get, "GET"}, {Head, "HEAD"}, {Delete, "DELETE"},
		{Post, "POST"}, {Put, "PUT"}, {Patch, "PATCH"},
	}
	for _, v := range verbs {
		resp, err := v.call(srv.URL, Options{Expects: []int{200}})
		if err != nil {
			t.Fatalf("%s: %v", v.want, err)
		}
		if resp.Status() != 200 {
			t.Fatalf("%s status = %d", v.want, resp.Status())
		}
		if method != v.want {
			t.Fatalf("server saw %q, want %q", method, v.want)
		}
	}
}
