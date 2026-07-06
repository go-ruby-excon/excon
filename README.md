<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-excon/brand/main/social/go-ruby-excon-excon.png" alt="go-ruby-excon/excon" width="720"></p>

# excon — go-ruby-excon

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-excon.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[`excon`](https://github.com/excon/excon) gem** — the fast, persistent HTTP
client. It reproduces the reusable connection, the one-shot verb helpers, the
option handling Excon performs around the wire (path / query building, header
merging, Basic auth, the `:expects` status assertion, `:idempotent` retry), the
`Response` (`Status`/`Body`/`Headers`/`RemoteIp`/`ReasonPhrase`), and the full
`Excon::Error` tree — **without any Ruby runtime**.

It is the Excon client for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-faraday](https://github.com/go-ruby-faraday/faraday),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb).

> **What it is — and isn't.** Everything Excon does *around* the socket is
> deterministic and needs **no interpreter**, so it lives here as pure Go:
> merging per-request options over the connection defaults, building the absolute
> URL (path plus an order-preserving, CGI-escaped query string), adding the Basic
> Authorization header, asserting the response status against `:expects` (raising
> the matching status error), and retrying an `:idempotent` request on a transport
> failure. The **HTTP round-trip itself is a host seam**: the `Doer` transport
> performs it. The default is `NetHTTP`, backed by `net/http` (it also captures
> the remote IP via `httptrace`); **tests inject a `DoerFunc` stub or drive
> `NetHTTP` over an `httptest` server**, and a host wires the real transport.

## Features

Faithful port of the `excon` gem's client core, validated against Ruby on every
platform where it is installed:

- **Persistent connection** — `excon.New(url, Options{...})` bound to a base URL,
  reused across requests, with per-request overrides merged over the defaults.
- **One-shot verbs** — `excon.Get`/`Head`/`Delete`/`Post`/`Put`/`Patch(url, opts)`
  and the connection methods `conn.Get`/`Head`/`Delete`/`Post`/`Put`/`Patch(opts)`
  plus the general `conn.Request(opts)`.
- **Options** — `Headers`, `Query`, `Body`, `Path`, `Expects` (status assertion),
  `Idempotent`/`RetryLimit`/`RetryInterval`, `ReadTimeout`/`WriteTimeout`/
  `ConnectTimeout`, `User`/`Password` (Basic auth), and `Middlewares`.
- **Response** — `Status`, `Body`, `Headers`, `RemoteIp`, `ReasonPhrase`,
  `Success` (mirrors `Excon::Response`).
- **`:expects`** — a status not in the expected set raises the matching
  `Excon::Error` subclass by code (404 → `NotFound`, 422 → `UnprocessableEntity`,
  an unmapped 4xx → `Client`, 5xx → `Server`, …).
- **`:idempotent` retry** — retries a socket / timeout transport failure up to
  `RetryLimit` attempts, sleeping `RetryInterval` between them.
- **Transport seam** — `conn.Transport(Doer)`; `NetHTTP()` is the default
  net/http transport (capturing the peer IP), a `DoerFunc` a test stub. **The
  core never opens a socket itself.**
- **Error tree** — the full `Excon::Error` hierarchy: `Socket` (`Certificate`),
  `Timeout`, `ResponseParse`, `ProxyConnectionError`, `ProxyParse`,
  `TooManyRedirects`, and the `HTTPStatus` subtree (`Informational`,
  `Redirection`, `Client` → `BadRequest`/`Unauthorized`/`Forbidden`/`NotFound`/…,
  `Server` → `InternalServerError`/`BadGateway`/`GatewayTimeout`/…), matched with
  `errors.Is` against the `Err*` sentinels (a superclass matches its subclasses).
- **`Utils`** — `Escape`/`Unescape` (CGI-faithful, `+`-for-space), the ordered
  `Query` codec, `BasicHeaderFrom`, and a case-insensitive `Headers`.

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian).

## Install

```sh
go get github.com/go-ruby-excon/excon
```

## Usage

```go
package main

import (
	"errors"
	"fmt"

	"github.com/go-ruby-excon/excon"
)

func main() {
	conn := excon.New("https://api.example.com", excon.Options{
		Headers: excon.HeadersOf([2]string{"Accept", "application/json"}),
	})

	resp, err := conn.Get(excon.Options{
		Path:    "/widgets",
		Query:   excon.QueryOf([2]string{"q", "gadget"}),
		Expects: []int{200},
	})
	if err != nil {
		// a *excon.Error: use errors.Is(err, excon.ErrNotFound), IsServerError(err), …
		if errors.Is(err, excon.ErrNotFound) {
			fmt.Println("no such widget")
		}
		return
	}
	fmt.Println(resp.Status(), resp.Success(), resp.RemoteIp(), resp.Body())

	// one-shot form
	_, _ = excon.Post("https://api.example.com/widgets",
		excon.Options{Body: `{"name":"gadget"}`, Expects: []int{201}})
}
```

### Injecting a transport (tests / hosts)

```go
conn := excon.New("https://api.example.com").Transport(
	excon.DoerFunc(func(req *excon.Request) (*excon.Response, error) {
		return excon.NewResponse(200, `{"ok":true}`,
			excon.HeadersOf([2]string{"Content-Type", "application/json"}),
			"OK", "203.0.113.7"), nil
	}))

resp, _ := conn.Get(excon.Options{Path: "/ping", Expects: []int{200}})
// resp.Status() == 200, resp.RemoteIp() == "203.0.113.7"
```

## Value model

| gem                                       | this package                                   |
| ----------------------------------------- | ---------------------------------------------- |
| `Excon.new(url, opts)`                    | `New(url, Options{...})`                        |
| `Excon.get/post/...(url, opts)`           | `Get/Post/...(url, opts)`                       |
| `conn.request(method:, ...)`              | `conn.Request(Options{Method, ...})`           |
| `conn.get/post/...(opts)`                 | `conn.Get/Post/...(opts)`                       |
| `:headers` / `:query` / `:body`           | `Options.Headers` / `.Query` / `.Body`         |
| `:expects` / `:idempotent`                | `Options.Expects` / `.Idempotent`              |
| `:user` / `:password`                     | `Options.User` / `.Password` (Basic auth)      |
| `:middlewares`                            | `Options.Middlewares` (`[]Middleware`)         |
| `Excon::Response#status/body/remote_ip`   | `(*Response).Status()/Body()/RemoteIp()`       |
| `Excon::Error` subtree                    | `*Error` + `Err*` sentinels (`errors.Is`)      |
| `Excon::Utils.query_string`               | `(*Query).Encode()` / `Escape` / `Unescape`    |

## Tests & coverage

The suite pairs deterministic tests (which alone hold coverage at **100%**, so
the qemu cross-arch and Windows lanes pass the gate) with a **differential
oracle** against Ruby: the escape codec, ordered query building, and the
Basic-auth header are diffed against the stdlib (`CGI` / `Base64`, exactly what
Excon uses), and the error hierarchy is diffed against the real `Excon::Error`
classes when the gem is installed. The oracle scripts skip themselves where ruby
(or the gem) is absent. **No test opens a remote socket** — the transport is
stubbed or driven over loopback `httptest`.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-excon/excon authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
