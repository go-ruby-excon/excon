// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"os/exec"
	"strings"
	"testing"
)

// The oracle tests diff this package against a reference Ruby. The escape codec
// and Basic-auth header are diffed against Ruby's stdlib (CGI / Base64), which
// is exactly what Excon uses to build query strings and auth headers — a
// rock-solid reference available on every non-Windows lane. The error tree is
// diffed against the `excon` gem's real Excon::Error class hierarchy when the gem
// is installed. All of these skip themselves where ruby (or the gem) is absent —
// the qemu cross-arch and Windows lanes — so the deterministic, ruby-free suite
// alone holds the 100% coverage gate there.

// stdlibRuby returns a ruby on PATH or skips (no gem required).
func stdlibRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping stdlib oracle")
	}
	return bin
}

// gemRuby returns a ruby whose `excon` gem loads, or skips.
func gemRuby(t *testing.T) string {
	t.Helper()
	bin := stdlibRuby(t)
	if err := exec.Command(bin, "-e", `require "excon"`).Run(); err != nil {
		t.Skip("excon gem absent; skipping gem oracle")
	}
	return bin
}

// rubyEval runs a ruby one-liner (binary stdout) and returns its trimmed output.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	out, err := exec.Command(bin, "-e", "$stdout.binmode\n"+script).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

func rubyString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s) + `"`
}

func TestOracleEscape(t *testing.T) {
	bin := stdlibRuby(t)
	for _, s := range []string{"a b/c&d=é", "hello world", "plain.text-_~", "100%x", "/?#[]@:"} {
		want := rubyEval(t, bin, `require "cgi"; print CGI.escape(`+rubyString(s)+`)`)
		if got := Escape(s); got != want {
			t.Fatalf("Escape(%q) = %q, CGI.escape = %q", s, got, want)
		}
	}
}

func TestOracleUnescape(t *testing.T) {
	bin := stdlibRuby(t)
	for _, s := range []string{"a+b%2Fc%26d%3D", "hello%20world", "%C3%A9", "plain.text-_~"} {
		want := rubyEval(t, bin, `require "cgi"; print CGI.unescape(`+rubyString(s)+`)`)
		if got := Unescape(s); got != want {
			t.Fatalf("Unescape(%q) = %q, CGI.unescape = %q", s, got, want)
		}
	}
}

func TestOracleQueryString(t *testing.T) {
	bin := stdlibRuby(t)
	// Query.Encode composes the same escaped pairs Excon builds via CGI.escape.
	q := QueryOf([2]string{"na me", "a/b"}, [2]string{"x", "y&z"})
	want := rubyEval(t, bin, `require "cgi"
pairs = [["na me","a/b"],["x","y&z"]]
print "?" + pairs.map { |k, v| "#{CGI.escape(k)}=#{CGI.escape(v)}" }.join("&")`)
	if got := q.Encode(); got != want {
		t.Fatalf("Query.Encode = %q, ruby = %q", got, want)
	}
}

func TestOracleBasicAuth(t *testing.T) {
	bin := stdlibRuby(t)
	want := rubyEval(t, bin, `require "base64"; print "Basic " + Base64.strict_encode64("aladdin:opensesame")`)
	if got := BasicHeaderFrom("aladdin", "opensesame"); got != want {
		t.Fatalf("BasicHeaderFrom = %q, ruby = %q", got, want)
	}
}

// TestOracleErrorTree diffs our Excon error hierarchy against the gem's real
// Excon::Error classes: each named subclass must exist and descend from the same
// ancestors we model. Skips if the gem is too old to expose a class we assert.
func TestOracleErrorTree(t *testing.T) {
	bin := gemRuby(t)
	// A representative slice of the tree: kind -> the ancestor chain we expect.
	checks := []struct {
		kind    ErrorKind
		parents []ErrorKind
	}{
		{KindNotFound, []ErrorKind{KindClient, KindHTTPStatus, KindError}},
		{KindUnprocessableEntity, []ErrorKind{KindClient, KindHTTPStatus, KindError}},
		{KindInternalServerError, []ErrorKind{KindServer, KindHTTPStatus, KindError}},
		{KindGatewayTimeout, []ErrorKind{KindServer, KindHTTPStatus, KindError}},
		{KindSocket, []ErrorKind{KindError}},
		{KindTimeout, []ErrorKind{KindError}},
	}
	simple := func(k ErrorKind) string { return strings.TrimPrefix(string(k), "Excon::Error::") }
	for _, c := range checks {
		name := simple(c.kind)
		script := `require "excon"
exit 2 unless Excon::Error.const_defined?(` + rubyString(name) + `, false)
print Excon::Error.const_get(` + rubyString(name) + `).ancestors.map(&:to_s).join("\n")`
		out, err := exec.Command(bin, "-e", script).CombinedOutput()
		if err != nil {
			t.Skipf("excon gem lacks Excon::Error::%s (%v); skipping error-tree oracle", name, err)
		}
		anc := string(out)
		// The class itself descends from each modelled parent (and from Error).
		for _, p := range c.parents {
			if !strings.Contains(anc, string(p)) {
				t.Fatalf("Excon::Error::%s ancestry %q missing parent %s", name, anc, p)
			}
		}
	}
}
