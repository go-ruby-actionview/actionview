// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "testing"

func eq(t *testing.T, got SafeBuffer, want string) {
	t.Helper()
	if got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}
}

func TestContentTag(t *testing.T) {
	eq(t, ContentTag("div", "hi", Attrs{{"class", "a"}}), `<div class="a">hi</div>`)
	eq(t, ContentTag("p", "<script>", nil), `<p>&lt;script&gt;</p>`)
	eq(t, ContentTag("p", Raw("<b>safe</b>"), nil), `<p><b>safe</b></p>`)
	eq(t, ContentTag("br", nil, nil), `<br></br>`)
	eq(t, ContentTag("textarea", "x", nil), "<textarea>\nx</textarea>")
	eq(t, ContentTagRaw("div", "<x>", Attrs{{"data-q", `"a"`}}), `<div data-q="&quot;a&quot;"><x></div>`)
}

func TestTagBuilder(t *testing.T) {
	eq(t, Tag("br", nil, nil), `<br>`)
	eq(t, Tag("hr", nil, Attrs{{"class", "sep"}}), `<hr class="sep">`)
	eq(t, Tag("div", "x", nil), `<div>x</div>`)
	eq(t, Tag("circle", nil, Attrs{{"r", "5"}}), `<circle r="5" />`)
	eq(t, Tag("input", nil, Attrs{{"type", "text"}, {"disabled", true}, {"value", nil}}),
		`<input type="text" disabled="disabled">`)
}

func TestTagOpen(t *testing.T) {
	eq(t, TagOpen("br", nil, false), `<br />`)
	eq(t, TagOpen("br", nil, true), `<br>`)
}

func TestTagOptionsData(t *testing.T) {
	got := ContentTag("div", "c", Attrs{{"data", map[string]any{"user_id": 5, "list": []any{1, 2}, "flag": false, "name": nil}}})
	eq(t, got, `<div data-flag="false" data-list="[1,2]" data-user-id="5">c</div>`)
}

func TestTagOptionsAria(t *testing.T) {
	eq(t, ContentTag("div", "c", Attrs{{"aria", map[string]any{
		"hidden": true, "empty": nil, "arr": []any{"a", "b"}, "none": []any{}, "m": map[string]any{"x": true}, "mn": map[string]any{},
	}}}), `<div aria-arr="a b" aria-hidden="true" aria-m="x">c</div>`)
}

func TestTagOptionsBooleanAndBlankKey(t *testing.T) {
	eq(t, Tag("input", nil, Attrs{{"required", true}, {"readonly", false}, {"", "skip"}, {"x", nil}}),
		`<input required="required">`)
}

func TestTagOptionClassArrayAndMap(t *testing.T) {
	eq(t, Tag("div", "x", Attrs{{"class", []any{"a", nil, "b"}}}), `<div class="a b">x</div>`)
	eq(t, Tag("div", "x", Attrs{{"class", map[string]any{"on": true, "off": false}}}), `<div class="on">x</div>`)
}

func TestTagOptionNonClassCollections(t *testing.T) {
	eq(t, ContentTagRaw("div", "x", Attrs{{"rel", []any{"a", "b"}}}), `<div rel="a b">x</div>`)
	// A map under a non-class attribute is an unrealistic edge; it stringifies
	// empty (this package does not join a bare hash), which exercises the branch.
	eq(t, ContentTag("div", "x", Attrs{{"rel", map[string]any{"k": "v"}}}), `<div rel="">x</div>`)
}

func TestTokenList(t *testing.T) {
	eq(t, TokenList("foo", nil, false, 123, "", "foo bar", map[string]any{"baz": true, "no": false}),
		"foo 123 bar baz")
	eq(t, ClassNames("a", "a"), "a")
	eq(t, TokenList(Attrs{{"x", true}, {"y", false}, {"", true}}), "x")
	eq(t, TokenList([]any{"n1", "n2"}), "n1 n2")
}

func TestUnescapeHTML(t *testing.T) {
	if unescapeHTML("plain") != "plain" {
		t.Fatal("plain")
	}
	if got := unescapeHTML("&amp;&lt;&gt;&quot;&#39;&#x27;"); got != `&<>"''` {
		t.Fatalf("unescape = %q", got)
	}
}

func TestCDATASection(t *testing.T) {
	eq(t, CDATASection("x]]>y"), "<![CDATA[x]]]]><![CDATA[>y]]>")
	eq(t, CDATASection("plain"), "<![CDATA[plain]]>")
}

func TestTagOptionsString(t *testing.T) {
	if got := TagOptions(Attrs{{"a", "b"}}); got != ` a="b"` {
		t.Fatalf("TagOptions = %q", got)
	}
}

func TestOpts(t *testing.T) {
	got := ContentTag("div", "x", Opts(map[string]any{"b": "2", "a": "1"}))
	eq(t, got, `<div a="1" b="2">x</div>`)
}

func TestBlank(t *testing.T) {
	for _, v := range []any{nil, false, "", "  ", Raw("  ")} {
		if !blank(v) {
			t.Errorf("blank(%v) should be true", v)
		}
	}
	for _, v := range []any{true, "x", Raw("y"), 0} {
		if blank(v) {
			t.Errorf("blank(%v) should be false", v)
		}
	}
}
