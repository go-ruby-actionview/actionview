// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"os/exec"
	"testing"
)

// These tests diff the Go sanitizers against Ruby, byte-for-byte, over the
// rails-html-sanitizer spec-style corpus (XSS vectors, allow-list scrubbing,
// nested / malformed markup, CSS). Two oracles are used:
//
//   - rails-html-sanitizer's Rails::HTML5 sanitizer classes, the exact library
//     this port mirrors (HTML5 vendor, the natural match for x/net/html); and
//   - ActionView's own SanitizeHelper (sanitize / strip_tags / strip_links /
//     sanitize_css), proving faithfulness to the helper surface for the cases
//     where its default (HTML4) vendor and HTML5 agree — every XSS vector here.
//
// Both oracles skip themselves when ruby or the gems are unavailable (the qemu
// cross-arch and Windows CI lanes); the deterministic tests hold the coverage
// gate on their own.

// rubySanitizer returns a ruby that can load rails-html-sanitizer / action_view,
// skipping otherwise.
func rubySanitizer(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping sanitizer oracle")
	}
	if err := exec.Command(bin, "-e", `require "rails-html-sanitizer"; require "action_view"`).Run(); err != nil {
		t.Skip("rails-html-sanitizer / action_view not installed; skipping sanitizer oracle")
	}
	return bin
}

// html5Preamble drives the rails-html-sanitizer HTML5 vendor. It reads the
// method (sanitize / full / link / css / tags / attrs) and the input from ARGV.
const html5Preamble = `
require "rails-html-sanitizer"
s = Rails::HTML5::SafeListSanitizer.new
f = Rails::HTML5::FullSanitizer.new
l = Rails::HTML5::LinkSanitizer.new
$stdout.binmode
kind, input = ARGV[0], ARGV[1]
print(case kind
  when "sanitize" then s.sanitize(input)
  when "full"     then f.sanitize(input)
  when "link"     then l.sanitize(input)
  when "css"      then s.sanitize_css(input)
  when "tags-b"   then s.sanitize(input, tags: %w[b])
  when "attrs-h"  then s.sanitize(input, attributes: %w[href])
  when "tags-s"   then s.sanitize(input, tags: %w[script])
  end)
`

// avSanitizePreamble drives ActionView's SanitizeHelper the way an application
// calls it, through a view context.
const avSanitizePreamble = `
require "action_view"
class V; include ActionView::Helpers::SanitizeHelper; end
$v = V.new
$stdout.binmode
kind, input = ARGV[0], ARGV[1]
print(case kind
  when "sanitize" then $v.sanitize(input)
  when "full"     then $v.strip_tags(input)
  when "link"     then $v.strip_links(input)
  when "css"      then $v.sanitize_css(input)
  end)
`

// runOracle evaluates one sanitizer call in ruby and returns its output.
func runOracle(t *testing.T, bin, preamble, kind, input string) string {
	t.Helper()
	out, err := exec.Command(bin, "-e", preamble, "--", kind, input).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error for [%s] %q: %v\n%s", kind, input, err, out)
	}
	return string(out)
}

// goSanitize dispatches to the Go sanitizer matching a preamble kind.
func goSanitize(kind, input string) string {
	switch kind {
	case "sanitize":
		return Sanitize(input, nil, nil).String()
	case "full":
		return StripTags(input).String()
	case "link":
		return StripLinks(input).String()
	case "css":
		return SanitizeCSS(input).String()
	case "tags-b":
		return Sanitize(input, []string{"b"}, nil).String()
	case "attrs-h":
		return Sanitize(input, nil, []string{"href"}).String()
	case "tags-s":
		return Sanitize(input, []string{"script"}, nil).String()
	}
	return ""
}

// oracleCases is the shared corpus of (method, input) pairs diffed against
// ruby. The XSS vectors, allow-list and CSS cases all agree between the HTML4
// and HTML5 vendors; the handful that are HTML5-serialisation specific (nested
// stripping, strip_tags of a script body) are only diffed against the HTML5
// oracle below.
var oracleCases = []struct{ kind, input string }{
	// XSS vectors — neutralised identically by both vendors.
	{"sanitize", `a b c<script language="Javascript">blah</script>d`},
	{"sanitize", `a<style>body{color:red}</style>b`},
	{"sanitize", `<a href="javascript:alert('x')">click</a>`},
	{"sanitize", `<a href="java&colon;script:alert(1)">x</a>`},
	{"sanitize", `<a href="javascript&#58;alert(1)">x</a>`},
	{"sanitize", `<img src="javascript:alert('x')">`},
	{"sanitize", `<a href="/x" onclick="evil()">x</a>`},
	{"sanitize", `<p>a<!-- comment -->b</p>`},
	{"sanitize", `<svg><img src="x"></svg>`},
	{"sanitize", `<img src="data:image/png;base64,AAAA">`},
	{"sanitize", `<img src="data:text/html,<script>alert(1)</script>">`},
	{"sanitize", `<img src="   ">`},
	// Allow-list scrubbing and encoding.
	{"sanitize", `<foo>bar</foo> <baz qux="1">quux</baz>`},
	{"sanitize", `<a href="http://x.com/a?b=1&c=2">x</a>`},
	{"sanitize", `<a href="mailto:a@b.com">m</a>`},
	{"sanitize", `<B>bold</B> <A HREF="/x">y</A>`},
	{"sanitize", `a &amp; b &lt; c &gt; d`},
	{"sanitize", `<b>unclosed <i>tags`},
	{"tags-b", `<b>bold</b> <i>it</i> <em>e</em>`},
	{"attrs-h", `<a href="/x" title="t" class="c">y</a>`},
	{"tags-s", `x<script>a<b</script>y`},
	// strip_tags / strip_links.
	{"full", `<b>hello</b> <a href="/x">world</a>`},
	{"full", `a<!-- c -->b`},
	{"full", `a &amp; b`},
	{"link", `<a href="/x">click</a> here`},
	{"link", `<a href="/x"><b>bold</b></a>`},
	{"link", `<a href="1">one</a> and <a href="2">two</a>`},
	{"link", `no links here`},
	// CSS.
	{"css", `color: red; background: url(javascript:alert(1)); width: 5px`},
	{"css", `width: expression(alert(1)); color: blue`},
	{"css", `margin: 0 auto; padding: 10px`},
	{"css", `color: red !important`},
	{"css", `font-family: 'Arial'; unknownprop: 5`},
	{"css", `background-color: #000`},
	{"css", `font-family: "a;b"; color: red`},
}

// TestSanitizerOracleHTML5 diffs every case against rails-html-sanitizer's
// HTML5 vendor — the library this package mirrors.
func TestSanitizerOracleHTML5(t *testing.T) {
	bin := rubySanitizer(t)
	// HTML5-serialisation-specific cases that differ from the HTML4 helper
	// default: nested stripping without libxml2's spurious whitespace, foreign
	// (svg) subtree pruning, and a script body kept as text under strip_tags.
	cases := append(append([]struct{ kind, input string }{}, oracleCases...),
		struct{ kind, input string }{"sanitize", `<div><script>x</script><b>bold</b></div>`},
		struct{ kind, input string }{"sanitize", `<div><p>keep</p><unknown>me</unknown></div>`},
		struct{ kind, input string }{"sanitize", `<svg><script>alert(1)</script></svg>text`},
		struct{ kind, input string }{"full", `a<script>alert(1)</script>b`},
	)
	for _, c := range cases {
		want := runOracle(t, bin, html5Preamble, c.kind, c.input)
		if got := goSanitize(c.kind, c.input); got != want {
			t.Errorf("HTML5 oracle mismatch [%s]\n  input: %q\n    got: %q\n   want: %q",
				c.kind, c.input, got, want)
		}
	}
}

// TestSanitizerOracleActionView diffs the corpus against ActionView's own
// SanitizeHelper (its default HTML4 vendor), proving the helper surface behaves
// identically on the security-relevant cases, where HTML4 and HTML5 agree.
func TestSanitizerOracleActionView(t *testing.T) {
	bin := rubySanitizer(t)
	for _, c := range oracleCases {
		// The tags:/attributes: override kinds are not exercised through the
		// bare helper preamble (which takes no options); the HTML5 oracle covers
		// them. Diff only the option-free helper methods here.
		switch c.kind {
		case "sanitize", "full", "link", "css":
		default:
			continue
		}
		want := runOracle(t, bin, avSanitizePreamble, c.kind, c.input)
		if got := goSanitize(c.kind, c.input); got != want {
			t.Errorf("ActionView oracle mismatch [%s]\n  input: %q\n    got: %q\n   want: %q",
				c.kind, c.input, got, want)
		}
	}
}
