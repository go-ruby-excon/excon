// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import "strings"

// QueryPair is one entry of a [Query]: a key, its value, and whether a value is
// present. A key with no value (HasVal false) is emitted bare — matching Excon,
// which encodes a nil-valued query key as just the escaped key.
type QueryPair struct {
	Key    string
	Val    string
	HasVal bool
}

// Query is the ordered set of URL query parameters passed as the :query option,
// mirroring the Hash (or String) Excon accepts. Keys are emitted in insertion
// order and may repeat (an array value in Excon becomes several pairs sharing a
// key). Encoding is byte-faithful to Excon::Utils.query_string.
type Query struct {
	pairs []QueryPair
}

// NewQuery returns an empty [Query].
func NewQuery() *Query { return &Query{} }

// QueryOf builds a [Query] from ordered key/value pairs (each with a value).
func QueryOf(kv ...[2]string) *Query {
	q := NewQuery()
	for _, e := range kv {
		q.Add(e[0], e[1])
	}
	return q
}

// Add appends a key=value pair.
func (q *Query) Add(key, val string) {
	q.pairs = append(q.pairs, QueryPair{Key: key, Val: val, HasVal: true})
}

// AddKey appends a bare key with no value (Excon's nil-valued query entry).
func (q *Query) AddKey(key string) {
	q.pairs = append(q.pairs, QueryPair{Key: key})
}

// Len reports the number of pairs.
func (q *Query) Len() int { return len(q.pairs) }

// Pairs returns the pairs in insertion order. The slice must not be mutated.
func (q *Query) Pairs() []QueryPair { return q.pairs }

// Encode renders the query as a leading-'?' string, byte-faithful to
// Excon::Utils.query_string: pairs in order, each key and value escaped with
// [Escape] (CGI.escape), a bare key emitted without '='. An empty query encodes
// to "".
func (q *Query) Encode() string {
	if len(q.pairs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('?')
	for i, p := range q.pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(Escape(p.Key))
		if p.HasVal {
			b.WriteByte('=')
			b.WriteString(Escape(p.Val))
		}
	}
	return b.String()
}
