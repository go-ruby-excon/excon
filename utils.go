// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import (
	"encoding/base64"
	"strings"
)

// Utils groups the deterministic string helpers Excon relies on around the
// wire: the percent-escape/unescape pair used to build query strings and the
// Basic-auth header builder. They are package-level functions here.
//
// The escape codec is byte-faithful to Ruby's CGI.escape, which Excon's
// Excon::Utils.query_string uses to encode query keys and values: the set
// [A-Za-z0-9 _.-] is left literal except that a space is rewritten to '+', and
// every other byte becomes %XX with upper-case hex.

// Escape percent-encodes s the way Ruby's CGI.escape does (the escaper Excon
// applies to query keys and values): [A-Za-z0-9 _.-~] stays literal, a space
// becomes '+', and any other byte becomes %XX with upper-case hex.
func Escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			b.WriteByte('+')
		case escapeUnreserved(c):
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hexDigit(c >> 4))
			b.WriteByte(hexDigit(c & 0xf))
		}
	}
	return b.String()
}

// Unescape reverses [Escape] the way Ruby's CGI.unescape does: '+' becomes a
// space and %XX becomes its byte. A truncated or invalid %XX is left literal.
func Unescape(s string) string {
	if !strings.ContainsAny(s, "%+") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '+':
			b.WriteByte(' ')
		case s[i] == '%' && i+2 < len(s):
			hi, ok1 := fromHex(s[i+1])
			lo, ok2 := fromHex(s[i+2])
			if ok1 && ok2 {
				b.WriteByte(hi<<4 | lo)
				i += 2
				continue
			}
			b.WriteByte('%')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// BasicHeaderFrom returns the Basic Authorization header value for a user and
// password: "Basic " followed by the newline-free base64 of "user:password",
// matching what Excon sets when :user/:password are supplied.
func BasicHeaderFrom(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

// escapeUnreserved reports whether c is left literal by [Escape] before the
// space-to-'+' rewrite: CGI.escape's unreserved set [A-Za-z0-9_.-~].
func escapeUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == '~':
		return true
	}
	return false
}

// hexDigit maps a nibble (0..15) to its upper-case hexadecimal ASCII digit.
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

// fromHex parses a single hexadecimal ASCII digit.
func fromHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
