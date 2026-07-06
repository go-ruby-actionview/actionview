// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "testing"

func TestTruncate(t *testing.T) {
	eq(t, Truncate("Once upon a time in a world far far away", 17, "...", "", true), "Once upon a ti...")
	eq(t, Truncate("Once upon a time", 10, "...", " ", true), "Once...")
	eq(t, Truncate("<b>&", 10, "...", "", true), "&lt;b&gt;&amp;")
	eq(t, Truncate("<b>&", 10, "...", "", false), "<b>&")
}

func TestPluralize(t *testing.T) {
	cases := []struct {
		count          any
		singular, plur string
		want           string
	}{
		{0, "person", "", "0 people"},
		{1, "person", "", "1 person"},
		{2, "person", "", "2 people"},
		{1.0, "person", "", "1.0 person"},
		{3, "person", "users", "3 users"},
		{nil, "person", "", "0 people"},
	}
	for _, c := range cases {
		if got := Pluralize(c.count, c.singular, c.plur); got != c.want {
			t.Errorf("Pluralize(%v) = %q, want %q", c.count, got, c.want)
		}
	}
}

func TestSimpleFormat(t *testing.T) {
	eq(t, SimpleFormat("Hello\n\nWorld\nLine", nil, "", nil), "<p>Hello</p>\n\n<p>World\n<br />Line</p>")
	eq(t, SimpleFormat("", nil, "", nil), "<p></p>")
	eq(t, SimpleFormat("x", Attrs{{"class", "c"}}, "div", nil), `<div class="c">x</div>`)
	eq(t, SimpleFormat("<b>drop</b>keep", nil, "", func(s string) string { return "clean" }), "<p>clean</p>")
}

func TestWordWrap(t *testing.T) {
	if got := WordWrap("The quick brown fox jumped", 10, ""); got != "The quick\nbrown fox\njumped" {
		t.Errorf("= %q", got)
	}
	if got := WordWrap("", 10, ""); got != "" {
		t.Errorf("empty = %q", got)
	}
	if got := WordWrap("a b c", 0, "|"); got != "a b c" {
		t.Errorf("defaults = %q", got)
	}
}

func TestHighlight(t *testing.T) {
	eq(t, Highlight("You searched for foo and FOO", []string{"foo"}, "", nil),
		"You searched for <mark>foo</mark> and <mark>FOO</mark>")
	eq(t, Highlight("a <b>foo</b> c", []string{"foo"}, "", nil), "a <b><mark>foo</mark></b> c")
	eq(t, Highlight("hit", []string{"hit"}, `<em>\1</em>`, nil), "<em>hit</em>")
	eq(t, Highlight("   ", []string{"x"}, "", nil), "   ")
	eq(t, Highlight("text", nil, "", nil), "text")
	eq(t, Highlight("text", []string{""}, "", nil), "text")
	eq(t, Highlight("bad", []string{"x"}, "", func(s string) string { return "clean" }), "clean")
}

func TestExcerpt(t *testing.T) {
	cases := []struct {
		text, phrase  string
		radius        int
		omission, sep string
		want          string
		found         bool
	}{
		{"This is an example", "an", 5, "...", "", "...s is an exam...", true},
		{"This next thing is an example", "ex", 2, "...", "", "...next...", true},
		{"This is a very beautiful morning", "very", 1, "...", " ", "...a very beautiful...", true},
		{"hello", "xyz", 5, "...", "", "", false},
		{"", "x", 5, "...", "", "", false},
		{"abc", "b", 10, "...", "", "abc", true},
	}
	for _, c := range cases {
		got, found := Excerpt(c.text, c.phrase, c.radius, c.omission, c.sep)
		if got != c.want || found != c.found {
			t.Errorf("Excerpt(%q,%q) = %q,%v want %q,%v", c.text, c.phrase, got, found, c.want, c.found)
		}
	}
}

func TestFirstLastN(t *testing.T) {
	if got := firstN([]int{1, 2, 3}, 5); len(got) != 3 {
		t.Errorf("firstN overflow = %v", got)
	}
	if got := lastN([]int{1, 2, 3}, -1); len(got) != 0 {
		t.Errorf("lastN neg = %v", got)
	}
}
