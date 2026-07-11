// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// This file is the pure-Go port of Rails' HTML sanitization stack —
// ActionView's SanitizeHelper backed by rails-html-sanitizer / Loofah. It
// reproduces the observable behaviour of the HTML5 sanitizer vendor
// (Rails::HTML5::Sanitizer): the SafeList/PermitScrubber allow-list model, the
// full (strip_tags) and link (strip_links) scrubbers, javascript:-URI
// neutralisation, comment/CDATA removal and CSS declaration scrubbing.
//
// HTML is parsed and serialised with the stdlib golang.org/x/net/html — a pure
// Go (cgo-free) HTML5 tree builder. The HTML5 vendor is the faithful match for
// x/net/html's HTML5 parsing; the security-critical outcomes (XSS vectors
// neutralised, disallowed tags/attributes stripped, allowed markup preserved)
// are identical to ActionView's default (HTML4) sanitize helper — see the
// oracle tests, which diff against both.

// DefaultAllowedTags is Rails::HTML's SafeList DEFAULT_ALLOWED_TAGS: the tags
// the SafeListSanitizer keeps when no explicit tag list is supplied. It is also
// the initial value of SanitizedAllowedTags.
var DefaultAllowedTags = []string{
	"a", "abbr", "acronym", "address", "b", "big", "blockquote", "br", "cite",
	"code", "dd", "del", "dfn", "div", "dl", "dt", "em", "h1", "h2", "h3", "h4",
	"h5", "h6", "hr", "i", "img", "ins", "kbd", "li", "mark", "ol", "p", "pre",
	"samp", "small", "span", "strong", "sub", "sup", "time", "tt", "ul", "var",
}

// DefaultAllowedAttributes is Rails::HTML's SafeList DEFAULT_ALLOWED_ATTRIBUTES:
// the attributes the SafeListSanitizer keeps when no explicit attribute list is
// supplied. It is also the initial value of SanitizedAllowedAttributes.
var DefaultAllowedAttributes = []string{
	"abbr", "alt", "cite", "class", "datetime", "height", "href", "lang", "name",
	"src", "title", "width", "xml:lang",
}

// SanitizedAllowedTags and SanitizedAllowedAttributes are the configurable
// allow-lists the package-level Sanitize helper consults when its tags /
// attributes arguments are nil — ActionView's sanitized_allowed_tags /
// sanitized_allowed_attributes class attributes. Assign to them to change the
// default policy process-wide.
var (
	SanitizedAllowedTags       = append([]string(nil), DefaultAllowedTags...)
	SanitizedAllowedAttributes = append([]string(nil), DefaultAllowedAttributes...)
)

// attrValIsURI is Loofah's SafeList::ATTR_VAL_IS_URI: attributes whose value is
// a URI and must be checked against the protocol allow-list.
var attrValIsURI = map[string]bool{
	"action": true, "cite": true, "href": true, "longdesc": true,
	"poster": true, "preload": true, "src": true, "xlink:href": true,
	"xml:base": true,
}

// allowedProtocols is Loofah's SafeList::ALLOWED_PROTOCOLS: the URI schemes a
// URI-valued attribute may use. Anything else (notably "javascript") is
// stripped.
var allowedProtocols = map[string]bool{
	"afs": true, "aim": true, "callto": true, "data": true, "ed2k": true,
	"fax": true, "feed": true, "ftp": true, "gopher": true, "http": true,
	"https": true, "irc": true, "line": true, "mailto": true, "modem": true,
	"news": true, "nntp": true, "rsync": true, "rtsp": true, "sftp": true,
	"sms": true, "ssh": true, "tag": true, "tel": true, "telnet": true,
	"urn": true, "webcal": true, "xmpp": true,
}

// allowedURIDataMediatypes is Loofah's SafeList::ALLOWED_URI_DATA_MEDIATYPES:
// the media types a data: URI may carry.
var allowedURIDataMediatypes = map[string]bool{
	"image/gif": true, "image/jpeg": true, "image/png": true,
	"text/css": true, "text/plain": true,
}

// htmlVoidElements is the HTML5 set of void elements: serialised as a start tag
// with no closing tag (and, unlike x/net/html's own renderer, no trailing
// slash, matching Nokogiri's HTML5 serialiser).
var htmlVoidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "keygen": true, "link": true,
	"meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

// htmlRawTextElements are elements whose text children are serialised verbatim
// (no entity escaping), per the HTML fragment serialisation algorithm.
var htmlRawTextElements = map[string]bool{
	"style": true, "script": true, "xmp": true, "iframe": true,
	"noembed": true, "noframes": true, "plaintext": true,
}

// controlChars is Loofah's CONTROL_CHARACTERS, stripped from a URI before the
// protocol is examined so obfuscated schemes cannot slip through.
var controlChars = regexp.MustCompile("[\\x60\\x00-\\x20\\x7f\\x{0080}-\\x{0101}]")

// uriProtocolRegex is Loofah's URI_PROTOCOL_REGEX (RFC 3986 scheme prefix).
var uriProtocolRegex = regexp.MustCompile(`^[a-z][a-z0-9+\-.]*:`)

// protocolSeparator is Loofah's PROTOCOL_SEPARATOR: a colon or one of its
// HTML/percent-encoded spellings, used to split the scheme off a URI.
var protocolSeparator = regexp.MustCompile(`(?i):|(&#0*58)|(&#x70)|(&#x0*3a)|(%|&#37;)3A`)

// toSet turns a slice of names into a membership set.
func toSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// attrName reconstructs an attribute's qualified name, prefixing the namespace
// (e.g. "xlink" + "href" -> "xlink:href"). x/net keeps colon-bearing HTML
// attributes such as "xml:lang" whole in Key with an empty Namespace, so the
// common case is just Key.
func attrName(a html.Attribute) string {
	if a.Namespace != "" {
		return a.Namespace + ":" + a.Key
	}
	return a.Key
}

// allowedURI reports whether a URI-valued attribute is safe, porting Loofah's
// HTML5::Scrub.allowed_uri?. Control characters are stripped, HTML entities and
// "&colon;" decoded and the string down-cased before the scheme is compared
// against the protocol allow-list; data: URIs additionally have their media
// type checked.
func allowedURI(uri string) bool {
	uri = controlChars.ReplaceAllString(uri, "")
	uri = html.UnescapeString(uri)
	uri = controlChars.ReplaceAllString(uri, "")
	uri = strings.ReplaceAll(uri, "&colon;", ":")
	uri = strings.ToLower(uri)
	if uriProtocolRegex.MatchString(uri) {
		parts := protocolSeparator.Split(uri, -1)
		protocol := parts[0]
		if !allowedProtocols[protocol] {
			return false
		}
		if protocol == "data" && len(parts) > 1 {
			mediatype := parts[1]
			if i := strings.IndexAny(mediatype, ";,"); i >= 0 {
				mediatype = mediatype[:i]
			}
			if mediatype != "" && !allowedURIDataMediatypes[mediatype] {
				return false
			}
		}
	}
	return true
}

// hasNonSpace reports whether s contains a non-whitespace character, mirroring
// Loofah's /[^[:space:]]/ test used to drop blank src attributes.
func hasNonSpace(s string) bool {
	return strings.TrimSpace(s) != ""
}

// scrubber describes one scrubbing pass over the parsed fragment. keepElement
// decides whether an element is preserved (versus stripped, with its children
// promoted); scrubAttrs rewrites a kept element's attributes; dropForeign
// children removes the subtree of a stripped element that lives in a foreign
// (svg / math) namespace, matching PermitScrubber's mutation-XSS guard.
type scrubber struct {
	keepElement func(*html.Node) bool
	scrubAttrs  func(*html.Node)
	dropForeign bool
}

// scrubFragment parses html as an HTML5 body fragment, runs the scrubber over
// it and serialises the result. It is the shared core of every sanitizer.
func scrubFragment(input string, s scrubber) string {
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, _ := html.ParseFragment(strings.NewReader(input), ctx)
	var out []*html.Node
	for _, n := range nodes {
		out = append(out, scrubNode(n, s)...)
	}
	var b strings.Builder
	for _, n := range out {
		detach(n)
		renderNode(&b, n)
	}
	return b.String()
}

// detach clears a node's tree links so it can be re-serialised or re-parented
// without dragging its former siblings along.
func detach(n *html.Node) {
	n.Parent, n.PrevSibling, n.NextSibling = nil, nil, nil
}

// scrubNode scrubs one node bottom-up and returns the nodes that replace it: the
// node itself (kept), its promoted children (stripped element), or nothing
// (comment / dropped foreign subtree). Children are scrubbed before their parent
// so a promoted subtree is already clean, exactly like Loofah's bottom-up walk.
func scrubNode(n *html.Node, s scrubber) []*html.Node {
	switch n.Type {
	case html.TextNode:
		return []*html.Node{n}
	case html.ElementNode:
		var kids []*html.Node
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			kids = append(kids, scrubNode(c, s)...)
			c = next
		}
		n.FirstChild, n.LastChild = nil, nil
		for _, k := range kids {
			detach(k)
		}
		if s.keepElement(n) {
			for _, k := range kids {
				n.AppendChild(k)
			}
			if s.scrubAttrs != nil {
				s.scrubAttrs(n)
			}
			return []*html.Node{n}
		}
		if s.dropForeign && n.Namespace != "" {
			return nil
		}
		return kids
	default:
		// Comments, doctypes and anything else are dropped.
		return nil
	}
}

// renderNode serialises a single node and its subtree following the HTML
// fragment serialisation algorithm (WHATWG), matching Nokogiri's HTML5 output:
// void elements get no closing tag or trailing slash and raw-text elements keep
// their children verbatim.
func renderNode(b *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		if n.Parent != nil && htmlRawTextElements[n.Parent.Data] {
			b.WriteString(n.Data)
		} else {
			b.WriteString(escapeText(n.Data))
		}
	case html.CommentNode:
		b.WriteString("<!--")
		b.WriteString(n.Data)
		b.WriteString("-->")
	case html.ElementNode:
		b.WriteByte('<')
		b.WriteString(n.Data)
		for _, a := range n.Attr {
			b.WriteByte(' ')
			b.WriteString(attrName(a))
			b.WriteString(`="`)
			b.WriteString(escapeAttr(a.Val))
			b.WriteByte('"')
		}
		b.WriteByte('>')
		if htmlVoidElements[n.Data] {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(b, c)
		}
		b.WriteString("</")
		b.WriteString(n.Data)
		b.WriteByte('>')
	}
}

// textEscaper and attrEscaper implement the HTML5 serialisation escaping rules:
// text nodes escape &, <, >, and the no-break space; attribute values escape &,
// the double quote and the no-break space.
var (
	textEscaper = strings.NewReplacer("&", "&amp;", " ", "&nbsp;", "<", "&lt;", ">", "&gt;")
	attrEscaper = strings.NewReplacer("&", "&amp;", " ", "&nbsp;", `"`, "&quot;")
)

func escapeText(s string) string { return textEscaper.Replace(s) }
func escapeAttr(s string) string { return attrEscaper.Replace(s) }

// SafeListSanitizer is the pure-Go Rails::HTML5::SafeListSanitizer: it keeps only
// the tags and attributes on its allow-lists, neutralises unsafe URIs and scrubs
// style declarations. The zero value is not usable; construct one with
// NewSafeListSanitizer.
type SafeListSanitizer struct {
	AllowedTags       []string
	AllowedAttributes []string
}

// NewSafeListSanitizer returns a SafeListSanitizer seeded with the Rails default
// allow-lists. Set AllowedTags / AllowedAttributes to override them.
func NewSafeListSanitizer() *SafeListSanitizer {
	return &SafeListSanitizer{
		AllowedTags:       append([]string(nil), DefaultAllowedTags...),
		AllowedAttributes: append([]string(nil), DefaultAllowedAttributes...),
	}
}

// WhiteListSanitizer is the deprecated Rails alias for SafeListSanitizer.
type WhiteListSanitizer = SafeListSanitizer

// Sanitize scrubs html against the sanitizer's allow-lists. tags and attributes
// override the instance allow-lists for this call when non-nil (ActionView's
// sanitize(html, tags:, attributes:)). The result is html-safe.
func (s *SafeListSanitizer) Sanitize(input string, tags, attributes []string) SafeBuffer {
	if input == "" {
		return Raw(input)
	}
	if tags == nil {
		tags = s.AllowedTags
	}
	if attributes == nil {
		attributes = s.AllowedAttributes
	}
	tagSet := toSet(tags)
	attrSet := toSet(attributes)
	scr := scrubber{
		keepElement: func(n *html.Node) bool { return tagSet[n.Data] },
		scrubAttrs:  func(n *html.Node) { scrubSafeListAttrs(n, attrSet) },
		dropForeign: true,
	}
	return Raw(scrubFragment(input, scr))
}

// SanitizeCSS scrubs a CSS declaration list against the safe CSS property /
// keyword / function allow-lists — ActionView's sanitize_css. The result is
// html-safe.
func (s *SafeListSanitizer) SanitizeCSS(style string) SafeBuffer {
	return Raw(scrubCSS(style))
}

// scrubSafeListAttrs applies the SafeList attribute policy to a kept element:
// drop attributes not on the allow-list, strip URI attributes with a disallowed
// scheme, drop blank src attributes, and scrub a permitted style attribute.
func scrubSafeListAttrs(n *html.Node, allowed map[string]bool) {
	kept := n.Attr[:0]
	for _, a := range n.Attr {
		name := attrName(a)
		if !allowed[name] {
			continue
		}
		if attrValIsURI[name] && !allowedURI(a.Val) {
			continue
		}
		if name == "src" && !hasNonSpace(a.Val) {
			continue
		}
		if name == "style" {
			a.Val = scrubCSS(a.Val)
		}
		kept = append(kept, a)
	}
	n.Attr = kept
}

// FullSanitizer is the pure-Go Rails::HTML5::FullSanitizer: it removes every tag
// and comment, keeping only the text — ActionView's strip_tags.
type FullSanitizer struct{}

// NewFullSanitizer returns a FullSanitizer.
func NewFullSanitizer() *FullSanitizer { return &FullSanitizer{} }

// Sanitize strips all markup from html, leaving the text content. The result is
// html-safe.
func (s *FullSanitizer) Sanitize(input string) SafeBuffer {
	if input == "" {
		return Raw(input)
	}
	scr := scrubber{
		keepElement: func(*html.Node) bool { return false },
		dropForeign: false,
	}
	return Raw(scrubFragment(input, scr))
}

// LinkSanitizer is the pure-Go Rails::HTML5::LinkSanitizer: it removes <a> tags
// (and href attributes) while keeping their inner content — ActionView's
// strip_links.
type LinkSanitizer struct{}

// NewLinkSanitizer returns a LinkSanitizer.
func NewLinkSanitizer() *LinkSanitizer { return &LinkSanitizer{} }

// Sanitize removes anchor tags from html, keeping the link text. The result is
// html-safe.
func (s *LinkSanitizer) Sanitize(input string) SafeBuffer {
	if input == "" {
		return Raw(input)
	}
	scr := scrubber{
		keepElement: func(n *html.Node) bool { return n.Data != "a" },
		scrubAttrs:  stripHref,
		dropForeign: true,
	}
	return Raw(scrubFragment(input, scr))
}

// stripHref removes href attributes, the LinkSanitizer's TargetScrubber
// attribute rule applied to the elements it keeps.
func stripHref(n *html.Node) {
	kept := n.Attr[:0]
	for _, a := range n.Attr {
		if attrName(a) == "href" {
			continue
		}
		kept = append(kept, a)
	}
	n.Attr = kept
}

// Package-level default sanitizer instances, the equivalents of ActionView's
// full_sanitizer / link_sanitizer / safe_list_sanitizer accessors.
var (
	defaultFullSanitizer = NewFullSanitizer()
	defaultLinkSanitizer = NewLinkSanitizer()
	defaultSafeSanitizer = NewSafeListSanitizer()
)

// Sanitize scrubs html with the SafeList policy, keeping only allowed tags and
// attributes and neutralising unsafe URIs — ActionView's sanitize. Pass nil
// tags / attributes to use the configurable SanitizedAllowedTags /
// SanitizedAllowedAttributes defaults, or explicit lists to override them for
// this call. The result is html-safe.
func Sanitize(input string, tags, attributes []string) SafeBuffer {
	if tags == nil {
		tags = SanitizedAllowedTags
	}
	if attributes == nil {
		attributes = SanitizedAllowedAttributes
	}
	return defaultSafeSanitizer.Sanitize(input, tags, attributes)
}

// SanitizeCSS scrubs a CSS declaration list, dropping properties, keywords and
// functions that are not on the safe list — ActionView's sanitize_css. The
// result is html-safe.
func SanitizeCSS(style string) SafeBuffer {
	return defaultSafeSanitizer.SanitizeCSS(style)
}

// StripTags removes every tag and comment from html, leaving only the text —
// ActionView's strip_tags. The result is html-safe.
func StripTags(input string) SafeBuffer {
	return defaultFullSanitizer.Sanitize(input)
}

// StripLinks removes anchor tags from html, keeping the link text —
// ActionView's strip_links. The result is html-safe.
func StripLinks(input string) SafeBuffer {
	return defaultLinkSanitizer.Sanitize(input)
}

// SanitizeSanitizer adapts the SafeList sanitizer to the Sanitizer seam so it
// can be handed to SimpleFormat / Highlight as their sanitize argument, giving
// those helpers the sanitize: true behaviour instead of the identity default.
func SanitizeSanitizer(s string) string {
	return Sanitize(s, nil, nil).String()
}
