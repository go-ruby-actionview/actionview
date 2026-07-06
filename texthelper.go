// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-ruby-activesupport/activesupport/coreext"
	"github.com/go-ruby-activesupport/activesupport/inflector"
	"github.com/go-ruby-erb/erb"
)

// Sanitizer is the seam ActionView's simple_format / highlight use to strip
// unsafe markup before wrapping it. A full HTML sanitizer (Rails::HTML /
// Loofah) is deferred (see the README roadmap); the default is IdentitySanitizer
// which passes text through unchanged, equivalent to calling the helpers with
// sanitize: false.
type Sanitizer func(string) string

// IdentitySanitizer returns text unchanged. It is the default sanitizer, so the
// text helpers behave like their MRI counterparts invoked with sanitize: false.
func IdentitySanitizer(s string) string { return s }

// Truncate shortens text to at most length characters, appending omission if it
// had to cut, and (when separator is non-empty) backing up to the last separator
// so words are not split — ActionView's truncate. The result is html-safe: it is
// html-escaped unless escape is false.
func Truncate(text string, length int, omission, separator string, escape bool) SafeBuffer {
	content := coreext.Truncate(text, length, omission, separator)
	if escape {
		return Raw(erb.HTMLEscape(content))
	}
	return Raw(content)
}

// Pluralize renders "count word", using the singular form when count is 1 (or
// "1"/"1.0") and otherwise the supplied plural or the inflector's plural of
// singular — ActionView's pluralize. Pass an empty plural to defer to the
// inflector.
func Pluralize(count any, singular, plural string) string {
	cs := toS(count)
	word := ""
	if cs == "1" || regexp.MustCompile(`^1(\.0+)?$`).MatchString(cs) {
		word = singular
	} else if plural != "" {
		word = plural
	} else {
		word = inflector.Pluralize(singular)
	}
	if cs == "" {
		cs = "0"
	}
	return cs + " " + word
}

// splitParagraphsPattern and singleNewlinePattern implement split_paragraphs:
// paragraphs are separated by blank lines, and a lone newline inside a paragraph
// becomes a "<br />".
var (
	crlfPattern    = regexp.MustCompile(`\r\n?`)
	paragraphSplit = regexp.MustCompile(`\n\n+`)
)

// insertBreaks emits a "<br />" after every "[^\n]\n" that is immediately
// followed by a non-newline, in a single left-to-right pass. This reproduces
// ActionView's gsub(/([^\n]\n)(?=[^\n])/, '\1<br />') — the look-ahead means the
// following character is not consumed and the inserted markup is never re-scanned
// (RE2 has no look-ahead, so the pass is done by hand).
func insertBreaks(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		b.WriteByte(p[i])
		if p[i] != '\n' && i+1 < len(p) && p[i+1] == '\n' && i+2 < len(p) && p[i+2] != '\n' {
			b.WriteByte('\n')
			b.WriteString("<br />")
			i++ // consume the '\n' we just emitted; the next char stays unread
		}
	}
	return b.String()
}

// splitParagraphs mirrors ActionView's split_paragraphs: normalise line endings,
// split on blank lines, and mark intra-paragraph line breaks with "<br />".
func splitParagraphs(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	text = crlfPattern.ReplaceAllString(text, "\n")
	var out []string
	for _, p := range paragraphSplit.Split(text, -1) {
		out = append(out, insertBreaks(p))
	}
	return out
}

// SimpleFormat wraps text in paragraph tags, converting blank lines to paragraph
// breaks and single newlines to <br /> — ActionView's simple_format. It applies
// sanitize first (default IdentitySanitizer, i.e. no sanitization); pass a real
// Sanitizer to strip markup. htmlOpts are applied to each wrapper tag; wrapperTag
// defaults to "p" when empty.
func SimpleFormat(text string, htmlOpts Attrs, wrapperTag string, sanitize Sanitizer) SafeBuffer {
	if sanitize == nil {
		sanitize = IdentitySanitizer
	}
	if wrapperTag == "" {
		wrapperTag = "p"
	}
	text = sanitize(text)
	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return ContentTag(wrapperTag, nil, htmlOpts)
	}
	parts := make([]string, len(paragraphs))
	for i, p := range paragraphs {
		parts[i] = ContentTag(wrapperTag, Raw(p), htmlOpts).String()
	}
	return Raw(strings.Join(parts, "\n\n"))
}

// WordWrap wraps text to lines no longer than lineWidth characters, breaking on
// whitespace, joined by breakSequence — a faithful port of ActionView's
// word_wrap regexp. lineWidth defaults to 80 when non-positive; breakSequence
// defaults to "\n" when empty.
func WordWrap(text string, lineWidth int, breakSequence string) string {
	if text == "" {
		return ""
	}
	if lineWidth <= 0 {
		lineWidth = 80
	}
	if breakSequence == "" {
		breakSequence = "\n"
	}
	pattern := regexp.MustCompile(fmt.Sprintf(`(.{1,%d})(?:[^\S\n]+\n?|\n*\z|\n)|\n`, lineWidth))
	wrapped := pattern.ReplaceAllString(text, "$1"+breakSequence)
	return strings.TrimSuffix(wrapped, breakSequence)
}

// Highlight wraps every occurrence of any phrase in text with highlighter
// (default "<mark>\1</mark>", where \1 is the match), skipping text inside HTML
// tags — ActionView's highlight. Matching is case-insensitive. sanitize runs
// first (default IdentitySanitizer). The result is html-safe.
func Highlight(text string, phrases []string, highlighter string, sanitize Sanitizer) SafeBuffer {
	if sanitize == nil {
		sanitize = IdentitySanitizer
	}
	text = sanitize(text)
	if highlighter == "" {
		highlighter = `<mark>\1</mark>`
	}
	if strings.TrimSpace(text) == "" || len(phrases) == 0 {
		return Raw(text)
	}
	escaped := make([]string, 0, len(phrases))
	for _, p := range phrases {
		if p != "" {
			escaped = append(escaped, regexp.QuoteMeta(p))
		}
	}
	if len(escaped) == 0 {
		return Raw(text)
	}
	pattern := regexp.MustCompile(`(?i)(` + strings.Join(escaped, "|") + `)`)
	segments := regexp.MustCompile(`<[^>]*|[^<]+`).FindAllString(text, -1)
	var b strings.Builder
	for _, seg := range segments {
		if strings.HasPrefix(seg, "<") {
			b.WriteString(seg)
			continue
		}
		b.WriteString(pattern.ReplaceAllStringFunc(seg, func(m string) string {
			return strings.ReplaceAll(highlighter, `\1`, m)
		}))
	}
	return Raw(b.String())
}

// Excerpt returns the first occurrence of phrase in text with up to radius
// characters (or separator-delimited tokens) of surrounding context, prefixing
// and suffixing omission when the excerpt does not reach the text boundaries —
// ActionView's excerpt. It returns ("", false) when phrase is not found.
func Excerpt(text, phrase string, radius int, omission, separator string) (string, bool) {
	if text == "" || phrase == "" {
		return "", false
	}
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(phrase))
	loc := re.FindString(text)
	if loc == "" {
		return "", false
	}
	phrase = loc
	if separator != "" {
		for _, value := range strings.Split(text, separator) {
			if re.MatchString(value) {
				phrase = value
				break
			}
		}
	}
	idx := strings.Index(text, phrase)
	firstPart, secondPart := text[:idx], text[idx+len(phrase):]

	prefix, firstPart := cutExcerptPart(true, firstPart, radius, omission, separator)
	postfix, secondPart := cutExcerptPart(false, secondPart, radius, omission, separator)

	sepFirst, sepSecond := "", ""
	if firstPart != "" {
		sepFirst = separator
	}
	if secondPart != "" {
		sepSecond = separator
	}
	affix := strings.TrimSpace(firstPart + sepFirst + phrase + sepSecond + secondPart)
	return prefix + affix + postfix, true
}

// cutExcerptPart trims one side of an excerpt to radius characters or tokens,
// reporting the omission marker to attach when it had to cut. first selects
// keeping the tail (text before the phrase) vs the head (text after it).
func cutExcerptPart(first bool, part string, radius int, omission, separator string) (affix, out string) {
	if separator != "" {
		tokens := strings.Split(part, separator)
		filtered := tokens[:0]
		for _, t := range tokens {
			if t != "" {
				filtered = append(filtered, t)
			}
		}
		tokens = filtered
		if len(tokens) > radius {
			affix = omission
		}
		if first {
			tokens = lastN(tokens, radius)
		} else {
			tokens = firstN(tokens, radius)
		}
		return affix, strings.Join(tokens, separator)
	}
	runes := []rune(part)
	if len(runes) > radius {
		affix = omission
	}
	if first {
		if len(runes) > radius {
			runes = runes[len(runes)-radius:]
		}
	} else {
		if len(runes) > radius {
			runes = runes[:radius]
		}
	}
	return affix, string(runes)
}

// firstN returns the first n elements of s (all of them when n exceeds len).
func firstN[T any](s []T, n int) []T {
	if n > len(s) {
		n = len(s)
	}
	if n < 0 {
		n = 0
	}
	return s[:n]
}

// lastN returns the last n elements of s (all of them when n exceeds len).
func lastN[T any](s []T, n int) []T {
	if n > len(s) {
		n = len(s)
	}
	if n < 0 {
		n = 0
	}
	return s[len(s)-n:]
}
