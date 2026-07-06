// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/go-ruby-erb/erb"
)

// Attr is a single HTML attribute: a name and a value. Attributes are carried in
// an ordered Attrs slice (not a map) because ActionView emits attributes in a
// deterministic order and that order is part of the byte-for-byte output the
// helpers reproduce.
//
// The value's dynamic type selects the rendering rule, matching MRI's
// tag_options: a bool toggles a boolean attribute, nil drops the attribute, a
// map[string]any under the "data" or "aria" key expands to prefixed sub-attrs,
// and a slice/map under "class" is token-joined.
type Attr struct {
	Key string
	Val any
}

// Attrs is an ordered list of HTML attributes. Construct it with a slice
// literal, e.g. Attrs{{"class", "btn"}, {"id", "go"}}; the emission order is the
// slice order, mirroring Ruby hash insertion order.
type Attrs []Attr

// Opts converts a Go map to an Attrs slice with keys in ascending order. It is a
// convenience for callers who have a map and do not care about attribute order;
// the sort makes the output deterministic. See the README fidelity note: MRI
// preserves hash insertion order, this package sorts map input.
func Opts(m map[string]any) Attrs {
	out := make(Attrs, 0, len(m))
	for _, k := range sortedKeys(m) {
		out = append(out, Attr{k, m[k]})
	}
	return out
}

// booleanAttributes is ActionView's BOOLEAN_ATTRIBUTES set: attributes that HTML
// treats as present/absent flags. When such an attribute has a truthy value the
// helpers emit name="name"; a falsey value drops it entirely.
var booleanAttributes = func() map[string]bool {
	names := strings.Fields(`allowfullscreen allowpaymentrequest async autofocus
		autoplay checked compact controls declare default defaultchecked
		defaultmuted defaultselected defer disabled enabled formnovalidate hidden
		indeterminate inert ismap itemscope loop multiple muted nohref nomodule
		noresize noshade novalidate nowrap open pauseonexit playsinline readonly
		required reversed scoped seamless selected sortable truespeed typemustmatch
		visible`)
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}()

// voidElements is the HTML5 void-element set the tag builder renders without a
// closing tag and with a bare ">" suffix (ActionView define_void_element).
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "keygen": true, "link": true,
	"meta": true, "source": true, "track": true, "wbr": true,
}

// selfClosingElements is the SVG self-closing set the tag builder renders with a
// " />" suffix (ActionView define_self_closing_element). The method-name forms
// with underscores (animate_motion, animate_transform) map to their camelCase
// SVG tag names.
var selfClosingElements = map[string]string{
	"animate": "animate", "animate_motion": "animateMotion",
	"animate_transform": "animateTransform", "circle": "circle",
	"ellipse": "ellipse", "line": "line", "path": "path", "polygon": "polygon",
	"polyline": "polyline", "rect": "rect", "set": "set", "stop": "stop",
	"use": "use", "view": "view",
}

// truthy reports Ruby truthiness: everything is truthy except nil and false.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// blank reports Ruby's Object#blank?: nil, false, an empty/whitespace string,
// and an empty collection are blank; everything else is present.
func blank(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case bool:
		return !x
	case string:
		return strings.TrimSpace(x) == ""
	case SafeBuffer:
		return strings.TrimSpace(x.s) == ""
	default:
		return false
	}
}

// jsonCompact renders v as compact JSON matching Ruby's #to_json for the values
// that appear in data-* attributes: arrays, numbers, booleans and (sorted) maps.
// HTML escaping is disabled so "<" is not turned into "<"; the surrounding
// tag_option handles HTML-escaping afterwards, exactly as MRI does.
func jsonCompact(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimRight(buf.String(), "\n")
}

// dasherize replaces underscores with dashes, like String#dasherize, so a data
// key of user_id becomes the attribute data-user-id.
func dasherize(s string) string { return strings.ReplaceAll(s, "_", "-") }

// tagOption renders a single key="value" pair. When escape is true the value is
// html-escaped (ActionView tag_option via unwrapped_html_escape); the class key
// token-joins a slice/map value first.
func tagOption(key string, value any, escape bool) string {
	var sval string
	switch v := value.(type) {
	case []any:
		if key == "class" {
			sval = strings.Join(buildTagValues(v...), " ")
		} else {
			parts := make([]string, len(v))
			for i, e := range v {
				parts[i] = toS(e)
			}
			sval = strings.Join(parts, " ")
		}
	case map[string]any:
		if key == "class" {
			sval = strings.Join(buildTagValues(v), " ")
		} else {
			sval = toS(v)
		}
	default:
		sval = toS(value)
	}
	if escape {
		sval = erb.HTMLEscape(sval)
	} else {
		sval = strings.ReplaceAll(sval, `"`, "&quot;")
	}
	return key + `="` + sval + `"`
}

// prefixTagOption renders a data-/aria- prefixed sub-attribute. Non
// string/number scalars are JSON-encoded first (ActionView prefix_tag_option),
// then the pair is escaped by tagOption.
func prefixTagOption(prefix, key string, value any, escape bool) string {
	k := prefix + "-" + dasherize(key)
	switch value.(type) {
	case string, SafeBuffer, *SafeBuffer:
		// kept as-is (String/Symbol path)
	default:
		value = jsonCompact(value)
	}
	return tagOption(k, value, escape)
}

// renderAttrs builds the attribute string (each pair preceded by a space) for a
// tag, implementing ActionView TagBuilder#tag_options: data/aria hash expansion,
// boolean attributes, class token-joining, and nil-dropping.
func renderAttrs(opts Attrs, escape bool) string {
	var b strings.Builder
	for _, a := range opts {
		key, value := a.Key, a.Val
		if key == "" {
			continue
		}
		switch {
		case (key == "data" || key == "aria") && isHash(value):
			b.WriteString(renderPrefixed(key, value, escape))
		case booleanAttributes[key]:
			if truthy(value) {
				b.WriteByte(' ')
				b.WriteString(key + `="` + key + `"`)
			}
		case value != nil:
			b.WriteByte(' ')
			b.WriteString(tagOption(key, value, escape))
		}
	}
	return b.String()
}

// isHash reports whether v is a map[string]any, the shape data/aria expansion
// recognises.
func isHash(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

// renderPrefixed expands a data or aria hash into prefixed sub-attributes. data
// drops only nil values; aria additionally token-joins array/hash values and
// drops empty ones, matching ActionView's two branches.
func renderPrefixed(prefix string, value any, escape bool) string {
	m := value.(map[string]any)
	var b strings.Builder
	for _, k := range sortedKeys(m) {
		v := m[k]
		if k == "" || v == nil {
			continue
		}
		if prefix == "aria" {
			switch vv := v.(type) {
			case []any:
				tokens := buildTagValues(vv...)
				if len(tokens) == 0 {
					continue
				}
				v = strings.Join(tokens, " ")
			case map[string]any:
				tokens := buildTagValues(vv)
				if len(tokens) == 0 {
					continue
				}
				v = strings.Join(tokens, " ")
			default:
				v = toS(v)
			}
		}
		b.WriteByte(' ')
		b.WriteString(prefixTagOption(prefix, k, v, escape))
	}
	return b.String()
}

// buildTagValues flattens class/token arguments into a slice of present token
// strings, implementing ActionView build_tag_values: hash keys with truthy
// values, nested arrays, and present scalars.
func buildTagValues(args ...any) []string {
	var out []string
	for _, arg := range args {
		switch x := arg.(type) {
		case map[string]any:
			for _, k := range sortedKeys(x) {
				if truthy(x[k]) && k != "" {
					out = append(out, k)
				}
			}
		case []any:
			out = append(out, buildTagValues(x...)...)
		case Attrs:
			for _, a := range x {
				if truthy(a.Val) && a.Key != "" {
					out = append(out, a.Key)
				}
			}
		default:
			if !blank(arg) {
				out = append(out, toS(arg))
			}
		}
	}
	return out
}

// contentTagString is the shared block-tag renderer used by ContentTag and the
// non-void tag builder: <name attrs>PRE_CONTENT content</name>, escaping content
// unless it is already html-safe. Only <textarea> has a leading-newline
// PRE_CONTENT string.
func contentTagString(name string, content any, opts Attrs, escape bool) SafeBuffer {
	attrs := renderAttrs(opts, escape)
	var body string
	if escape {
		body = safeString(content)
	} else {
		body = toS(content)
	}
	pre := ""
	if name == "textarea" {
		pre = "\n"
	}
	return Raw("<" + name + attrs + ">" + pre + body + "</" + name + ">")
}

// ContentTag renders an HTML block tag surrounding content, mirroring
// ActionView's content_tag. It always emits a closing tag (even for names like
// br), html-escapes content unless it is a SafeBuffer, and renders opts via the
// tag_options rules. Pass a nil opts for no attributes.
func ContentTag(name string, content any, opts Attrs) SafeBuffer {
	return contentTagString(name, content, opts, true)
}

// ContentTagRaw is ContentTag with escaping disabled (content_tag(..., escape:
// false)): content and attribute values are emitted verbatim. Use only with
// trusted input.
func ContentTagRaw(name string, content any, opts Attrs) SafeBuffer {
	return contentTagString(name, content, opts, false)
}

// Tag renders a tag through the ActionView tag builder (tag.NAME). A void
// element (br, input, ...) becomes <name attrs>; an SVG self-closing element
// becomes <name attrs />; anything else becomes <name attrs>content</name>. Pass
// content=nil for an empty non-void element.
func Tag(name string, content any, opts Attrs) SafeBuffer {
	if voidElements[name] {
		return Raw("<" + name + renderAttrs(opts, true) + ">")
	}
	if svg, ok := selfClosingElements[name]; ok {
		return Raw("<" + svg + renderAttrs(opts, true) + " />")
	}
	return contentTagString(name, content, opts, true)
}

// TagOpen renders the legacy standalone tag helper (tag(name, options, open)):
// <name attrs /> by default, or <name attrs> when open is true. This XHTML-style
// " />" suffix is what the form-tag helpers build on, distinct from the HTML5
// tag builder above.
func TagOpen(name string, opts Attrs, open bool) SafeBuffer {
	suffix := " />"
	if open {
		suffix = ">"
	}
	return Raw("<" + name + renderAttrs(opts, true) + suffix)
}

// TagOptions renders just the attribute string for opts (with a leading space
// per attribute), exposing ActionView's tag_options for callers assembling tags
// by hand.
func TagOptions(opts Attrs) string { return renderAttrs(opts, true) }

// TokenList builds a deduplicated, space-joined token string from its arguments,
// implementing ActionView's token_list / class_names: hash keys with truthy
// values, nested arrays, and present scalars, with HTML entities decoded and
// whitespace-split before de-duplication.
func TokenList(args ...any) SafeBuffer {
	raw := buildTagValues(args...)
	seen := map[string]bool{}
	var tokens []string
	for _, r := range raw {
		for _, tok := range strings.Fields(unescapeHTML(r)) {
			if !seen[tok] {
				seen[tok] = true
				tokens = append(tokens, tok)
			}
		}
	}
	return Raw(erb.HTMLEscape(strings.Join(tokens, " ")))
}

// ClassNames is ActionView's alias for TokenList.
func ClassNames(args ...any) SafeBuffer { return TokenList(args...) }

// unescapeHTML reverses the five html_escape entity references, matching the
// CGI.unescape_html token_list performs before splitting.
func unescapeHTML(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	r := strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`,
		"&#39;", "'", "&#x27;", "'",
	)
	return r.Replace(s)
}

// CDATASection wraps content in a CDATA section, escaping any embedded "]]>" by
// splitting it across two sections, exactly like ActionView's cdata_section.
func CDATASection(content string) SafeBuffer {
	splitted := strings.ReplaceAll(content, "]]>", "]]]]><![CDATA[>")
	return Raw("<![CDATA[" + splitted + "]]>")
}
