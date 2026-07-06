// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// stub returns a DoerFunc that records the request it saw and replies with a
// fixed response, so the connection logic runs without touching the network.
func stub(resp *Response, rec **Request) Doer {
	return DoerFunc(func(req *Request) (*Response, error) {
		if rec != nil {
			*rec = req
		}
		return resp, nil
	})
}

func okResp(status int) *Response {
	return NewResponse(status, "body", HeadersOf([2]string{"Content-Type", "text/plain"}), "OK", "203.0.113.7")
}

// ---- utils ----

func TestEscapeUnescape(t *testing.T) {
	cases := map[string]string{
		"a b":         "a+b",
		"plain.text-": "plain.text-",
		"a/b&c=d":     "a%2Fb%26c%3Dd",
		"~tilde":      "~tilde", // CGI.escape leaves '~' literal
		"_under":      "_under",
	}
	for in, want := range cases {
		if got := Escape(in); got != want {
			t.Fatalf("Escape(%q) = %q, want %q", in, got, want)
		}
		if got := Unescape(want); got != in {
			t.Fatalf("Unescape(%q) = %q, want %q", want, got, in)
		}
	}
	// Fast path: no % or + returns input unchanged.
	if got := Unescape("plain"); got != "plain" {
		t.Fatalf("Unescape fast path = %q", got)
	}
	// Truncated %X (no room for two hex digits) and invalid hex are left literal.
	if got := Unescape("a%2"); got != "a%2" {
		t.Fatalf("Unescape truncated = %q", got)
	}
	if got := Unescape("a%zzb"); got != "a%zzb" {
		t.Fatalf("Unescape invalid hex = %q", got)
	}
	// Non-ASCII byte encodes to %XX upper-case (default branch of Escape).
	if got := Escape("é"); got != "%C3%A9" {
		t.Fatalf("Escape non-ascii = %q", got)
	}
}

func TestFromHexAllForms(t *testing.T) {
	if got := Unescape("%6a%2F%3d"); got != "j/=" {
		t.Fatalf("lower/upper hex decode = %q", got)
	}
}

func TestBasicHeaderFrom(t *testing.T) {
	if got := BasicHeaderFrom("aladdin", "opensesame"); got != "Basic YWxhZGRpbjpvcGVuc2VzYW1l" {
		t.Fatalf("BasicHeaderFrom = %q", got)
	}
}

// ---- headers ----

func TestHeaders(t *testing.T) {
	h := HeadersOf([2]string{"Content-Type", "text/plain"})
	h.Set("content-type", "application/json") // case-insensitive overwrite
	if v, ok := h.Get("CONTENT-TYPE"); !ok || v != "application/json" {
		t.Fatalf("Get = %q,%v", v, ok)
	}
	if h.Len() != 1 || len(h.Pairs()) != 1 {
		t.Fatalf("len = %d", h.Len())
	}
	if h.Pairs()[0].Key != "Content-Type" {
		t.Fatalf("original casing lost: %q", h.Pairs()[0].Key)
	}
	if !h.Has("content-type") || h.Has("nope") {
		t.Fatal("Has wrong")
	}
	if _, ok := h.Get("absent"); ok {
		t.Fatal("Get absent should be false")
	}
	// Delete with reindex.
	h.Set("X-A", "1")
	h.Set("X-B", "2")
	h.Delete("Content-Type")
	h.Delete("absent") // no-op branch
	if h.Len() != 2 || h.Pairs()[0].Key != "X-A" {
		t.Fatalf("after delete: %+v", h.Pairs())
	}
	// Clone independence + Merge nil and non-nil.
	c := h.Clone()
	c.Set("X-A", "changed")
	if v, _ := h.Get("X-A"); v != "1" {
		t.Fatalf("clone shared state: %q", v)
	}
	m := h.Merge(nil)
	if m.Len() != 2 {
		t.Fatalf("merge nil = %d", m.Len())
	}
	m = h.Merge(HeadersOf([2]string{"X-A", "9"}, [2]string{"X-C", "3"}))
	if v, _ := m.Get("X-A"); v != "9" {
		t.Fatalf("merge overwrite = %q", v)
	}
	if v, _ := m.Get("X-C"); v != "3" {
		t.Fatalf("merge append = %q", v)
	}
	// Set on a zero-value Headers (nil index).
	var z Headers
	z.Set("k", "v")
	if v, _ := z.Get("k"); v != "v" {
		t.Fatalf("zero headers set = %q", v)
	}
}

// ---- query ----

func TestQuery(t *testing.T) {
	if NewQuery().Encode() != "" {
		t.Fatal("empty query should encode to empty string")
	}
	q := QueryOf([2]string{"a", "hello world"}, [2]string{"b", "x&y"})
	q.AddKey("flag")
	if got := q.Encode(); got != "?a=hello+world&b=x%26y&flag" {
		t.Fatalf("Encode = %q", got)
	}
	if q.Len() != 3 || len(q.Pairs()) != 3 {
		t.Fatalf("len = %d", q.Len())
	}
	if q.Pairs()[2].HasVal {
		t.Fatal("bare key should have HasVal false")
	}
}

// ---- errors ----

func TestErrorTree(t *testing.T) {
	e := newStatusError([]int{200}, &Request{}, okResp(404))
	if e.Kind != KindNotFound {
		t.Fatalf("404 kind = %s", e.Kind)
	}
	if !errors.Is(e, ErrNotFound) || !errors.Is(e, ErrClient) ||
		!errors.Is(e, ErrHTTPStatus) || !errors.Is(e, ErrError) {
		t.Fatal("404 should match NotFound/Client/HTTPStatus/Error")
	}
	if errors.Is(e, ErrServer) || errors.Is(e, ErrTimeout) {
		t.Fatal("404 should not match Server/Timeout")
	}
	if IsHTTPStatusError(e) != true || IsClientError(e) != true || IsServerError(e) {
		t.Fatal("predicate mismatch on 404")
	}
	if !strings.Contains(e.Error(), "Expected([200]) <=> Actual(404 OK)") {
		t.Fatalf("message = %q", e.Error())
	}
	// Non-*Error target and non-*Error argument.
	if e.Is(errors.New("x")) {
		t.Fatal("Is against non-*Error should be false")
	}
	if IsClientError(errors.New("x")) {
		t.Fatal("isKind against non-*Error should be false")
	}
	// Certificate < Socket < Error.
	cert := &Error{Kind: KindCertificate}
	if !cert.Is(ErrSocket) || !IsSocketError(cert) {
		t.Fatal("Certificate should be a Socket error")
	}
}

func TestStatusErrorKindRanges(t *testing.T) {
	cases := map[int]ErrorKind{
		100: KindInformational,
		302: KindRedirection,
		418: KindClient, // unmapped 4xx
		599: KindServer, // unmapped 5xx
		700: KindHTTPStatus,
		422: KindUnprocessableEntity,
		500: KindInternalServerError,
	}
	for status, want := range cases {
		if got := statusErrorKind(status); got != want {
			t.Fatalf("statusErrorKind(%d) = %s, want %s", status, got, want)
		}
	}
}

func TestTransportErrorAndRetryable(t *testing.T) {
	if e := newTransportError(KindSocket, nil); e.Message != string(KindSocket) {
		t.Fatalf("nil-cause message = %q", e.Message)
	}
	cause := errors.New("boom")
	e := newTransportError(KindTimeout, cause)
	if e.Message != "boom" || !errors.Is(e, cause) {
		t.Fatalf("cause not wrapped: %v", e)
	}
	if !IsTimeout(e) || !retryable(e) {
		t.Fatal("timeout should be retryable")
	}
	if !retryable(newTransportError(KindSocket, nil)) {
		t.Fatal("socket should be retryable")
	}
	if retryable(&Error{Kind: KindNotFound}) || retryable(errors.New("x")) {
		t.Fatal("non-transport errors should not be retryable")
	}
}

// ---- middleware ----

func TestMiddleware(t *testing.T) {
	var rec *Request
	c := New("http://h/").Transport(stub(okResp(200), &rec))
	tag := RequestMiddleware(func(req *Request) error {
		req.Headers.Set("X-Tag", "yes")
		return nil
	})
	seen := ""
	watch := ResponseMiddleware(func(resp *Response) error {
		seen = resp.Body()
		return nil
	})
	if _, err := c.Get(Options{Middlewares: []Middleware{tag, watch}}); err != nil {
		t.Fatal(err)
	}
	if v, _ := rec.Headers.Get("X-Tag"); v != "yes" {
		t.Fatalf("request middleware did not run: %q", v)
	}
	if seen != "body" {
		t.Fatalf("response middleware did not run: %q", seen)
	}
	// Aborting request middleware.
	boom := errors.New("abort-req")
	c = New("http://h/").Transport(stub(okResp(200), nil))
	_, err := c.Get(Options{Middlewares: []Middleware{RequestMiddleware(func(*Request) error { return boom })}})
	if !errors.Is(err, boom) {
		t.Fatalf("request abort = %v", err)
	}
	// Aborting response middleware.
	_, err = c.Get(Options{Middlewares: []Middleware{ResponseMiddleware(func(*Response) error { return boom })}})
	if !errors.Is(err, boom) {
		t.Fatalf("response abort = %v", err)
	}
	// Response middleware skipped when downstream errors.
	c = New("http://h/").Transport(DoerFunc(func(*Request) (*Response, error) {
		return nil, newTransportError(KindSocket, boom)
	}))
	ran := false
	_, err = c.Get(Options{Middlewares: []Middleware{ResponseMiddleware(func(*Response) error { ran = true; return nil })}})
	if ran || !IsSocketError(err) {
		t.Fatalf("response mw should be skipped on error: ran=%v err=%v", ran, err)
	}
}

// ---- response ----

func TestResponse(t *testing.T) {
	r := NewResponse(204, "", nil, "No Content", "")
	if r.Status() != 204 || r.Body() != "" || r.Headers().Len() != 0 ||
		r.ReasonPhrase() != "No Content" || r.RemoteIp() != "" {
		t.Fatalf("response fields wrong: %+v", r)
	}
	if !r.Success() {
		t.Fatal("204 should be success")
	}
	if okResp(404).Success() {
		t.Fatal("404 should not be success")
	}
}

// ---- connection & options ----

func TestConnectionRequestAndAuth(t *testing.T) {
	var rec *Request
	c := New("https://api.example.com/base?seed=1",
		Options{Headers: HeadersOf([2]string{"Accept", "application/json"})}).
		Transport(stub(okResp(200), &rec))

	resp, err := c.Get(Options{
		Path:  "/widgets",
		Query: QueryOf([2]string{"q", "gadget"}),
		User:  "user", Password: "pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status() != 200 {
		t.Fatalf("status = %d", resp.Status())
	}
	if rec.URL != "https://api.example.com/widgets?q=gadget" {
		t.Fatalf("url = %q", rec.URL)
	}
	if rec.Method != "GET" {
		t.Fatalf("method = %q", rec.Method)
	}
	if v, _ := rec.Headers.Get("Authorization"); v != BasicHeaderFrom("user", "pass") {
		t.Fatalf("basic auth = %q", v)
	}
	if v, _ := rec.Headers.Get("Accept"); v != "application/json" {
		t.Fatalf("default header lost: %q", v)
	}

	// Explicit Authorization is not overwritten by :user/:password.
	c.Transport(stub(okResp(200), &rec))
	if _, err := c.Get(Options{
		Headers: HeadersOf([2]string{"Authorization", "Bearer tok"}),
		User:    "u", Password: "p",
	}); err != nil {
		t.Fatal(err)
	}
	if v, _ := rec.Headers.Get("Authorization"); v != "Bearer tok" {
		t.Fatalf("explicit auth overwritten: %q", v)
	}

	// No path and no query: buildURL defaults path to "/", no query appended.
	c2 := New("http://h").Transport(stub(okResp(200), &rec))
	if _, err := c2.Request(); err != nil {
		t.Fatal(err)
	}
	if rec.URL != "http://h/" {
		t.Fatalf("default url = %q", rec.URL)
	}
	if v, _ := rec.Headers.Get("Authorization"); v != "" {
		t.Fatalf("no-cred request should have no auth header")
	}
}

func TestConnectionVerbs(t *testing.T) {
	var rec *Request
	c := New("http://h/").Transport(stub(okResp(200), &rec))
	verbs := []struct {
		call func(...Options) (*Response, error)
		want string
	}{
		{c.Get, "GET"}, {c.Head, "HEAD"}, {c.Delete, "DELETE"},
		{c.Post, "POST"}, {c.Put, "PUT"}, {c.Patch, "PATCH"},
	}
	for _, v := range verbs {
		if _, err := v.call(); err != nil {
			t.Fatalf("%s: %v", v.want, err)
		}
		if rec.Method != v.want {
			t.Fatalf("method = %q, want %q", rec.Method, v.want)
		}
	}
	// Body override + base body inheritance via merge.
	c = New("http://h/", Options{Body: "base-body"}).Transport(stub(okResp(200), &rec))
	if _, err := c.Post(); err != nil {
		t.Fatal(err)
	}
	if rec.Body != "base-body" {
		t.Fatalf("base body not inherited: %q", rec.Body)
	}
	if _, err := c.Post(Options{Body: "override"}); err != nil {
		t.Fatal(err)
	}
	if rec.Body != "override" {
		t.Fatalf("body override failed: %q", rec.Body)
	}
}

func TestExpects(t *testing.T) {
	c := New("http://h/").Transport(stub(okResp(200), nil))
	// Match: no error.
	if _, err := c.Get(Options{Expects: []int{200, 204}}); err != nil {
		t.Fatalf("expects match should not error: %v", err)
	}
	// Mismatch: status error carrying request+response.
	c.Transport(stub(okResp(500), nil))
	_, err := c.Get(Options{Expects: []int{200}})
	if !IsServerError(err) || !IsHTTPStatusError(err) {
		t.Fatalf("expects mismatch = %v", err)
	}
	var ee *Error
	if !errors.As(err, &ee) || ee.Response == nil || ee.Request == nil {
		t.Fatalf("status error missing context: %+v", ee)
	}
}

func TestIdempotentRetry(t *testing.T) {
	orig := sleep
	slept := 0
	sleep = func(time.Duration) { slept++ }
	defer func() { sleep = orig }()

	// Retryable transport failures, retried until one succeeds.
	calls := 0
	flaky := DoerFunc(func(*Request) (*Response, error) {
		calls++
		if calls < 3 {
			return nil, newTransportError(KindTimeout, errors.New("i/o timeout"))
		}
		return okResp(200), nil
	})
	c := New("http://h/").Transport(flaky)
	resp, err := c.Get(Options{Idempotent: true, RetryLimit: 5, RetryInterval: 10})
	if err != nil || resp.Status() != 200 {
		t.Fatalf("retry-then-succeed: resp=%v err=%v", resp, err)
	}
	if calls != 3 || slept != 2 {
		t.Fatalf("calls=%d slept=%d, want 3 and 2", calls, slept)
	}

	// Retryable failure that never recovers: exhausts RetryLimit attempts.
	calls = 0
	c.Transport(DoerFunc(func(*Request) (*Response, error) {
		calls++
		return nil, newTransportError(KindSocket, errors.New("refused"))
	}))
	if _, err := c.Get(Options{Idempotent: true, RetryLimit: 2}); !IsSocketError(err) {
		t.Fatalf("exhausted retry = %v", err)
	}
	if calls != 2 {
		t.Fatalf("exhausted attempts = %d, want 2", calls)
	}

	// Not idempotent: a single attempt, no retry.
	calls = 0
	c.Transport(DoerFunc(func(*Request) (*Response, error) {
		calls++
		return nil, newTransportError(KindTimeout, errors.New("timeout"))
	}))
	if _, err := c.Get(); !IsTimeout(err) {
		t.Fatalf("non-idempotent = %v", err)
	}
	if calls != 1 {
		t.Fatalf("non-idempotent attempts = %d, want 1", calls)
	}

	// Idempotent but a non-retryable error: not retried.
	calls = 0
	c.Transport(DoerFunc(func(*Request) (*Response, error) {
		calls++
		return nil, &Error{Kind: KindResponseParse, Message: "bad"}
	}))
	if _, err := c.Get(Options{Idempotent: true, RetryLimit: 4}); !errors.Is(err, ErrResponseParse) {
		t.Fatalf("non-retryable = %v", err)
	}
	if calls != 1 {
		t.Fatalf("non-retryable attempts = %d, want 1", calls)
	}
}
