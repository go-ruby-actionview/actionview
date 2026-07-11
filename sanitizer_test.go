// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// TestSanitizeXSSVectors checks that the SafeList sanitizer neutralises the
// classic cross-site-scripting vectors: script/style bodies are demoted to
// inert text, javascript: URIs (including entity-obfuscated colons) are
// stripped, event handlers are dropped, comments removed, and foreign (svg)
// subtrees pruned — while the surrounding safe markup survives.
func TestSanitizeXSSVectors(t *testing.T) {
	cases := []struct{ in, want string }{
		{`a b c<script language="Javascript">blah</script>d`, `a b cblahd`},
		{`a<style>body{color:red}</style>b`, `abody{color:red}b`},
		{`<a href="javascript:alert('x')">click</a>`, `<a>click</a>`},
		{`<a href="java&colon;script:alert(1)">x</a>`, `<a>x</a>`},
		{`<a href="javascript&#58;alert(1)">x</a>`, `<a>x</a>`},
		{`<img src="javascript:alert('x')">`, `<img>`},
		{`<a href="/x" onclick="evil()">x</a>`, `<a href="/x">x</a>`},
		{`<p>a<!-- c -->b</p>`, `<p>ab</p>`},
		{`<svg><script>alert(1)</script></svg>text`, `text`},
		{`<svg><img src="x"></svg>`, `<img src="x">`},
		{`<foo>bar</foo> <baz qux="1">quux</baz>`, `bar quux`},
		{`<b>unclosed <i>tags`, `<b>unclosed <i>tags</i></b>`},
		{`<img src="data:image/png;base64,AAAA">`, `<img src="data:image/png;base64,AAAA">`},
		{`<img src="data:text/html,<script>">`, `<img>`},
		{`<a href="http://x.com/a?b=1&c=2">x</a>`, `<a href="http://x.com/a?b=1&amp;c=2">x</a>`},
		{`<a href="mailto:a@b.com">m</a>`, `<a href="mailto:a@b.com">m</a>`},
		{`<img src="   ">`, `<img>`},
		{`<B>bold</B> <A HREF="/x">y</A>`, `<b>bold</b> <a href="/x">y</a>`},
		{"", ""},
	}
	for _, c := range cases {
		if got := Sanitize(c.in, nil, nil).String(); got != c.want {
			t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSanitizeTagsAndAttributes exercises the per-call allow-list overrides and
// the SafeListSanitizer instance defaults.
func TestSanitizeTagsAndAttributes(t *testing.T) {
	if got := Sanitize(`<b>bold</b> <i>it</i>`, []string{"b"}, nil).String(); got != `<b>bold</b> it` {
		t.Errorf("tags override = %q", got)
	}
	if got := Sanitize(`<a href="/x" title="t" class="c">y</a>`, nil, []string{"href"}).String(); got != `<a href="/x">y</a>` {
		t.Errorf("attributes override = %q", got)
	}
	// style attribute, when explicitly permitted, is CSS-scrubbed in place.
	got := Sanitize(`<p style="color: red; background: url(javascript:1)">x</p>`, nil, []string{"style"}).String()
	if got != `<p style="color:red;">x</p>` {
		t.Errorf("style scrub = %q", got)
	}
	// Instance method with nil args falls back to the instance allow-lists.
	s := NewSafeListSanitizer()
	if got := s.Sanitize(`<b>x</b><script>y</script>`, nil, nil).String(); got != `<b>x</b>y` {
		t.Errorf("instance defaults = %q", got)
	}
	// Empty input short-circuits.
	if got := s.Sanitize("", nil, nil).String(); got != "" {
		t.Errorf("empty = %q", got)
	}
	// WhiteListSanitizer is the alias.
	var _ *WhiteListSanitizer = s
}

// TestSanitizeConfigurableDefaults verifies the package-level configurable
// allow-lists are consulted when Sanitize is called with nil lists.
func TestSanitizeConfigurableDefaults(t *testing.T) {
	savedTags := SanitizedAllowedTags
	savedAttrs := SanitizedAllowedAttributes
	defer func() {
		SanitizedAllowedTags = savedTags
		SanitizedAllowedAttributes = savedAttrs
	}()
	SanitizedAllowedTags = []string{"b"}
	SanitizedAllowedAttributes = []string{}
	if got := Sanitize(`<b class="c">x</b> <i>y</i>`, nil, nil).String(); got != `<b>x</b> y` {
		t.Errorf("configurable defaults = %q", got)
	}
}

// TestStripTags checks the FullSanitizer strips every tag and comment, keeping
// the text.
func TestStripTags(t *testing.T) {
	cases := []struct{ in, want string }{
		{`<b>hello</b> <a href="/x">world</a>`, `hello world`},
		{`a<script>alert(1)</script>b`, `aalert(1)b`},
		{`a<!-- c -->b`, `ab`},
		{`a &amp; b`, `a &amp; b`},
		{"", ""},
	}
	for _, c := range cases {
		if got := StripTags(c.in).String(); got != c.want {
			t.Errorf("StripTags(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	_ = NewFullSanitizer()
}

// TestStripLinks checks the LinkSanitizer removes anchors (and hrefs on kept
// elements) while preserving their content.
func TestStripLinks(t *testing.T) {
	cases := []struct{ in, want string }{
		{`<a href="/x">click</a> here`, `click here`},
		{`<a href="/x"><b>bold</b></a>`, `<b>bold</b>`},
		{`<a href="1">one</a> and <a href="2">two</a>`, `one and two`},
		{`no links here`, `no links here`},
		{`<p href="x" class="c">keep</p>`, `<p class="c">keep</p>`},
		{"", ""},
	}
	for _, c := range cases {
		if got := StripLinks(c.in).String(); got != c.want {
			t.Errorf("StripLinks(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	_ = NewLinkSanitizer()
}

// TestSanitizeCSS exercises the CSS declaration scrubber across the property /
// keyword / function allow-lists, url() rejection and !important.
func TestSanitizeCSS(t *testing.T) {
	cases := []struct{ in, want string }{
		{`color: red; background: url(javascript:alert(1)); width: 5px`, `color:red;width:5px;`},
		{`width: expression(alert(1)); color: blue`, `color:blue;`},
		{`COLOR: RED`, `color:RED;`},
		{`margin: 0 auto; padding: 10px`, `margin:0 auto;padding:10px;`},
		{`color: red !important`, `color:red !important;`},
		{`font-family: 'Arial'; unknownprop: 5`, `font-family:'Arial';`},
		{`background-color: #000`, `background-color:#000;`},
		{`color: red;;`, `color:red;`},
		{`font-family: "a;b"; color: red`, `font-family:"a;b";color:red;`},
		{`font-family: "a:b"; color: red`, `font-family:"a:b";color:red;`},
		{`background: url(http://x); color: red`, `color:red;`},
		{`colorred; color: blue`, `color:blue;`},
		{`: red; color: blue`, `color:blue;`},
		{`margin: -5px`, `margin:-5px;`},
		{`color: blur(5px)`, `color:blur(5px);`},
		{`color: bogusfunc(1)`, ``},
		{`margin: notakeyword`, ``},
		{`font-family: "a'b"`, ``},
		{`font-family: "unterminated`, ``},
		{"", ""},
	}
	for _, c := range cases {
		if got := SanitizeCSS(c.in).String(); got != c.want {
			t.Errorf("SanitizeCSS(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAllowedURI covers allowedURI's scheme and data-mediatype branches
// directly, including the code paths not reachable via a single sanitize call.
func TestAllowedURI(t *testing.T) {
	cases := []struct {
		uri string
		ok  bool
	}{
		{"http://example.com", true},
		{"/relative/path", true},
		{"#fragment", true},
		{"mailto:a@b.com", true},
		{"javascript:alert(1)", false},
		{"data:image/png;base64,AAAA", true},
		{"data:text/html,<script>", false},
		{"data:", true},          // no media type
		{"data:image/png", true}, // media type without ;/, separators
	}
	for _, c := range cases {
		if got := allowedURI(c.uri); got != c.ok {
			t.Errorf("allowedURI(%q) = %v, want %v", c.uri, got, c.ok)
		}
	}
}

// TestSanitizeAllowsRawTextElement checks that when a raw-text element such as
// <script> is explicitly permitted, its body is serialised verbatim (the
// raw-text serialisation branch).
func TestSanitizeAllowsRawTextElement(t *testing.T) {
	if got := Sanitize(`x<script>a<b</script>y`, []string{"script"}, nil).String(); got != `x<script>a<b</script>y` {
		t.Errorf("raw-text = %q", got)
	}
}

// TestSanitizePreNewline checks the leading-newline handling for <pre>, which
// the HTML5 parser strips a single one of.
func TestSanitizePreNewline(t *testing.T) {
	if got := Sanitize("<pre>\n\ntext</pre>", nil, nil).String(); got != "<pre>\ntext</pre>" {
		t.Errorf("pre double newline = %q", got)
	}
	if got := Sanitize("<pre>\ntext</pre>", nil, nil).String(); got != "<pre>text</pre>" {
		t.Errorf("pre single newline = %q", got)
	}
}

// TestRenderNodeComment covers the comment serialisation branch of renderNode,
// which the scrubbers never reach (they drop comments) but which the serialiser
// implements for completeness.
func TestRenderNodeComment(t *testing.T) {
	n := &html.Node{Type: html.CommentNode, Data: " hi "}
	var b strings.Builder
	renderNode(&b, n)
	if b.String() != "<!-- hi -->" {
		t.Errorf("comment render = %q", b.String())
	}
}

// TestAttrNameNamespace covers attrName's namespaced branch, which foreign
// (svg / math) attributes take but which the default allow-list never keeps.
func TestAttrNameNamespace(t *testing.T) {
	if got := attrName(html.Attribute{Namespace: "xlink", Key: "href"}); got != "xlink:href" {
		t.Errorf("attrName namespaced = %q", got)
	}
}

// TestSanitizeSanitizerSeam wires the real sanitizer into the SimpleFormat /
// Highlight seam, replacing the identity default.
func TestSanitizeSanitizerSeam(t *testing.T) {
	if got := SanitizeSanitizer(`<a href="javascript:x">hi</a>`); got != `<a>hi</a>` {
		t.Errorf("SanitizeSanitizer = %q", got)
	}
	got := SimpleFormat(`before <img onerror="alert(1)" src="x">`, nil, "", SanitizeSanitizer).String()
	if got != `<p>before <img src="x"></p>` {
		t.Errorf("SimpleFormat with sanitizer = %q", got)
	}
}

// TestCSSTokenizerEdgeCases covers the string-escape and unterminated-paren
// paths in the CSS tokenizer that the oracle cases do not force.
func TestCSSTokenizerEdgeCases(t *testing.T) {
	// Escaped quote inside a string keeps the string a single token (which then
	// fails the embedded-quote check, dropping the property).
	if got := SanitizeCSS(`font-family: "a\"b"`).String(); got != "" {
		t.Errorf("escaped-quote string = %q", got)
	}
	// Unterminated function paren still tokenises to the end of input.
	if got := SanitizeCSS(`color: rgb(0,0,0`).String(); got != `color:rgb(0,0,0;` {
		t.Errorf("unterminated paren = %q", got)
	}
	// A quote inside a function's parentheses is balanced correctly.
	if got := SanitizeCSS(`color: attr("x)y")`).String(); got != `color:attr("x)y");` {
		t.Errorf("quote in parens = %q", got)
	}
	// A lone / empty string token is dropped (len<2 and empty-inner branches).
	if got := SanitizeCSS(`font-family: "`).String(); got != "" {
		t.Errorf("lone quote = %q", got)
	}
	if got := SanitizeCSS(`font-family: ""`).String(); got != "" {
		t.Errorf("empty string = %q", got)
	}
	// Parentheses and quotes preceding the first top-level colon exercise the
	// property-name splitter's bracket/quote tracking; the names are invalid so
	// the declarations drop, leaving only the following valid one.
	if got := SanitizeCSS(`foo(x): red; color: blue`).String(); got != `color:blue;` {
		t.Errorf("paren before colon = %q", got)
	}
	if got := SanitizeCSS(`"a:b": red; color: blue`).String(); got != `color:blue;` {
		t.Errorf("quote before colon = %q", got)
	}
}
