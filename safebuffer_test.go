// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "testing"

func TestSafeBufferConcatEscapes(t *testing.T) {
	b := NewSafeBuffer()
	if b.Len() != 0 || b.HTMLSafe() != true {
		t.Fatalf("empty buffer: len=%d safe=%v", b.Len(), b.HTMLSafe())
	}
	b.Concat("<b>")
	if got := b.String(); got != "&lt;b&gt;" {
		t.Fatalf("Concat escaped = %q", got)
	}
	b.SafeConcat("<i>")
	if got := b.String(); got != "&lt;b&gt;<i>" {
		t.Fatalf("SafeConcat = %q", got)
	}
	b.AppendSafe(Raw("<u>"))
	if got := b.String(); got != "&lt;b&gt;<i><u>" {
		t.Fatalf("AppendSafe = %q", got)
	}
	if b.Len() == 0 {
		t.Fatal("Len should be non-zero")
	}
}

func TestRawIsNotEscaped(t *testing.T) {
	if got := Raw("<b>&").String(); got != "<b>&" {
		t.Fatalf("Raw = %q", got)
	}
	if !Raw("x").HTMLSafe() {
		t.Fatal("Raw must be html-safe")
	}
}

func TestHTMLEscape(t *testing.T) {
	if got := HTMLEscape(`&<>"'`); got != "&amp;&lt;&gt;&quot;&#39;" {
		t.Fatalf("HTMLEscape = %q", got)
	}
}

func TestSafeString(t *testing.T) {
	var np *SafeBuffer
	sb := Raw("<x>")
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{sb, "<x>"},
		{&sb, "<x>"},
		{np, ""},
		{"<a>", "&lt;a&gt;"},
		{5, "5"},
	}
	for _, c := range cases {
		if got := safeString(c.in); got != c.want {
			t.Errorf("safeString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
