// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"regexp"
	"strings"
)

// This file ports Loofah's HTML5::Scrub.scrub_css: it parses a CSS declaration
// list and keeps only properties, keywords and functions on the safe list,
// dropping anything that carries a url() (a classic exfiltration / script
// vector) or an unknown function such as expression().

// allowedCSSProperties is Loofah's SafeList::ALLOWED_CSS_PROPERTIES.
var allowedCSSProperties = toSet(strings.Fields(`
	align-content align-items align-self aspect-ratio azimuth background-color
	border-bottom-color border-collapse border-color border-left-color
	border-right-color border-top-color clear color cursor direction display
	elevation flex flex-basis flex-direction flex-flow flex-grow flex-shrink
	flex-wrap float font font-family font-size font-style font-variant font-weight
	height justify-content letter-spacing line-height list-style list-style-type
	max-height max-width min-height min-width order overflow overflow-x overflow-y
	page-break-after page-break-before page-break-inside pause pause-after
	pause-before pitch pitch-range richness speak speak-header speak-numeral
	speak-punctuation speech-rate stress text-align text-decoration text-indent
	unicode-bidi vertical-align voice-family volume white-space width`))

// allowedSVGProperties is Loofah's SafeList::ALLOWED_SVG_PROPERTIES.
var allowedSVGProperties = toSet(strings.Fields(`
	fill fill-opacity fill-rule stroke stroke-linecap stroke-linejoin
	stroke-opacity stroke-width`))

// shorthandCSSProperties is Loofah's SafeList::SHORTHAND_CSS_PROPERTIES; for
// these the value idents are additionally validated against the keyword list.
var shorthandCSSProperties = toSet(strings.Fields(`background border margin padding`))

// allowedCSSKeywords is Loofah's SafeList::ALLOWED_CSS_KEYWORDS (named colours
// and layout keywords) that may appear in a shorthand property value.
var allowedCSSKeywords = toSet(strings.Fields(`
	!important aliceblue antiquewhite aqua aquamarine auto azure beige bisque
	black blanchedalmond block blue blueviolet bold both bottom brown burlywood
	cadetblue center chartreuse chocolate collapse coral cornflowerblue cornsilk
	crimson cyan darkblue darkcyan darkgoldenrod darkgray darkgreen darkgrey
	darkkhaki darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon
	darkseagreen darkslateblue darkslategray darkslategrey darkturquoise
	darkviolet dashed deeppink deepskyblue dimgray dimgrey dodgerblue dotted
	double firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold
	goldenrod gray green greenyellow grey groove hidden honeydew hotpink
	indianred indigo inherit initial inset italic ivory khaki lavender
	lavenderblush lawngreen left lemonchiffon lightblue lightcoral lightcyan
	lightgoldenrodyellow lightgray lightgreen lightgrey lightpink lightsalmon
	lightseagreen lightskyblue lightslategray lightslategrey lightsteelblue
	lightyellow lime limegreen linen magenta maroon medium mediumaquamarine
	mediumblue mediumorchid mediumpurple mediumseagreen mediumslateblue
	mediumspringgreen mediumturquoise mediumvioletred midnightblue mintcream
	mistyrose moccasin navajowhite navy none normal nowrap oldlace olive olivedrab
	orange orangered orchid outset palegoldenrod palegreen paleturquoise
	palevioletred papayawhip peachpuff peru pink plum pointer powderblue purple
	red revert ridge right rosybrown royalblue saddlebrown salmon sandybrown
	seagreen seashell separate sienna silver skyblue slateblue slategray slategrey
	snow solid springgreen steelblue tan teal thick thin thistle tomato top
	transparent turquoise underline unset violet wheat white whitesmoke yellow
	yellowgreen`))

// allowedCSSFunctions is Loofah's SafeList::ALLOWED_CSS_FUNCTIONS; a value
// function whose name is not here is dropped (which drops the whole property
// when the function was its only value).
var allowedCSSFunctions = toSet(strings.Fields(`
	attr blur brightness calc circle contrast counter counters cubic-bezier
	drop-shadow ellipse grayscale hsl hsla hue-rotate hwb inset invert
	linear-gradient matrix matrix3d opacity perspective polygon radial-gradient
	repeating-linear-gradient repeating-radial-gradient rgb rgba rotate rotate3d
	rotatex rotatey rotatez saturate scale scale3d scalex scaley scalez sepia
	skew skewx skewy symbols translate translate3d translatex translatey
	translatez`))

// cssKeywordish is Loofah's CSS_KEYWORDISH: hex colours, rgb() literals and
// bare dimensions that are accepted as shorthand-property idents without being
// on the keyword list.
var cssKeywordish = regexp.MustCompile(`^(#[0-9a-fA-F]+|rgb\(\d+%?,\d*%?,?\d*%?\)?|-?\d{0,3}\.?\d{0,10}(ch|cm|r?em|ex|in|lh|mm|pc|pt|px|Q|vmax|vmin|vw|vh|%|,|\))?)$`)

// cssComp is one tokenised component of a CSS property value.
type cssComp struct {
	kind string // "ws", "string", "func", "ident", "other"
	raw  string
	name string // function name (kind == "func")
}

// scrubCSS sanitises a CSS declaration list, returning the safe subset in
// canonical "name:value;" form. It mirrors Loofah's scrub_css: unknown or
// url()-bearing properties are dropped, idents/strings/functions are filtered
// against the allow-lists, and !important is preserved.
func scrubCSS(style string) string {
	var out strings.Builder
	for _, decl := range splitCSSDeclarations(style) {
		name, value, ok := splitCSSDeclaration(decl)
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		important := false
		if v, cut := stripImportant(value); cut {
			value, important = v, true
		}
		comps, hasURL := tokenizeCSSValue(value)
		if hasURL {
			continue
		}
		first := strings.SplitN(name, "-", 2)[0]
		shorthand := shorthandCSSProperties[first]
		if !allowedCSSProperties[name] && !allowedSVGProperties[name] && !shorthand {
			continue
		}
		built := buildCSSValue(comps, shorthand)
		if built == "" {
			continue
		}
		if important {
			built += " !important"
		}
		out.WriteString(name)
		out.WriteByte(':')
		out.WriteString(built)
		out.WriteByte(';')
	}
	return out.String()
}

// buildCSSValue assembles the kept value components into a single string,
// applying the per-component allow-list rules (strings without embedded quotes,
// allowed functions, allowed/keywordish idents for shorthand properties).
func buildCSSValue(comps []cssComp, shorthand bool) string {
	var parts []string
	for _, c := range comps {
		switch c.kind {
		case "ws":
			parts = append(parts, " ")
		case "string":
			if cssStringOK(c.raw) {
				parts = append(parts, c.raw)
			}
		case "func":
			if allowedCSSFunctions[strings.ToLower(c.name)] {
				parts = append(parts, c.raw)
			}
		case "ident":
			if !shorthand || allowedCSSKeywords[c.raw] || cssKeywordish.MatchString(c.raw) {
				parts = append(parts, c.raw)
			}
		default:
			parts = append(parts, c.raw)
		}
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// cssStringOK ports Loofah's CSS_PROPERTY_STRING_WITHOUT_EMBEDDED_QUOTES: a
// value string is kept only when it is properly closed and carries no embedded
// quote characters. It is only ever called on tokenised string components, which
// always begin with their delimiter, so a bare unterminated delimiter (last
// character does not match the opener) is rejected. Written out by hand because
// RE2 has no back-references for the matched delimiter.
func cssStringOK(raw string) bool {
	if len(raw) < 2 || raw[len(raw)-1] != raw[0] {
		return false
	}
	inner := raw[1 : len(raw)-1]
	return inner != "" && !strings.ContainsAny(inner, `"'`)
}

// splitCSSDeclarations splits a declaration list on top-level semicolons,
// keeping semicolons inside strings and parentheses intact.
func splitCSSDeclarations(s string) []string {
	var decls []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case c == ';' && depth == 0:
			decls = append(decls, s[start:i])
			start = i + 1
		}
	}
	decls = append(decls, s[start:])
	return decls
}

// splitCSSDeclaration splits one declaration at its first top-level colon into a
// property name and value. It reports false when there is no colon.
func splitCSSDeclaration(decl string) (name, value string, ok bool) {
	depth := 0
	var quote byte
	for i := 0; i < len(decl); i++ {
		c := decl[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case c == ':' && depth == 0:
			return decl[:i], decl[i+1:], true
		}
	}
	return "", "", false
}

// stripImportant removes a trailing "!important" (case-insensitive, with
// surrounding whitespace) from a value, reporting whether it was present.
func stripImportant(value string) (string, bool) {
	trimmed := strings.TrimRight(value, " \t\r\n")
	if len(trimmed) >= len("!important") &&
		strings.EqualFold(trimmed[len(trimmed)-len("!important"):], "!important") {
		return trimmed[:len(trimmed)-len("!important")], true
	}
	return value, false
}

// tokenizeCSSValue splits a property value into components and reports whether
// it contains a url() token (which disqualifies the whole property).
func tokenizeCSSValue(s string) (comps []cssComp, hasURL bool) {
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\r' || s[j] == '\n') {
				j++
			}
			comps = append(comps, cssComp{kind: "ws", raw: s[i:j]})
			i = j
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(s) && s[j] != c {
				if s[j] == '\\' && j+1 < len(s) {
					j++
				}
				j++
			}
			if j < len(s) {
				j++ // closing quote
			}
			comps = append(comps, cssComp{kind: "string", raw: s[i:j]})
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentChar(s[j]) {
				j++
			}
			ident := s[i:j]
			if j < len(s) && s[j] == '(' {
				k := scanBalancedParens(s, j)
				raw := s[i:k]
				if strings.EqualFold(ident, "url") {
					hasURL = true
				}
				comps = append(comps, cssComp{kind: "func", raw: raw, name: ident})
				i = k
			} else {
				comps = append(comps, cssComp{kind: "ident", raw: ident})
				i = j
			}
		default:
			j := i
			for j < len(s) && !isSpace(s[j]) && s[j] != '(' && s[j] != '"' && s[j] != '\'' {
				j++
			}
			comps = append(comps, cssComp{kind: "other", raw: s[i:j]})
			i = j
		}
	}
	return comps, hasURL
}

// scanBalancedParens returns the index just past the ")" that closes the "("
// at position open (or the end of s if unterminated).
func scanBalancedParens(s string, open int) int {
	depth := 0
	var quote byte
	for i := open; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

func isIdentStart(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
