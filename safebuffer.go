// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "github.com/go-ruby-erb/erb"

// HTMLEscape is ActionView's html_escape (ERB::Util.html_escape): it replaces
// the five HTML-significant characters with their entity references, escaping
// the apostrophe as "&#39;". It is re-exported from go-ruby-erb so callers can
// escape a raw string exactly as MRI does.
func HTMLEscape(s string) string { return erb.HTMLEscape(s) }

// SafeBuffer is ActionView's html-safe string, the backing type of
// ActiveSupport::SafeBuffer / ActionView::OutputBuffer. A SafeBuffer carries a
// string that is known to be HTML-safe (already escaped or trusted markup).
//
// The whole point of SafeBuffer is automatic escaping on concatenation: when a
// raw (non-safe) fragment is appended with Concat, it is html-escaped first,
// while an already-safe fragment is appended verbatim. This is what lets view
// helpers compose trusted markup with untrusted user data without
// double-escaping and without opening XSS holes.
//
// The zero value is a usable, empty, safe buffer.
type SafeBuffer struct {
	s string
}

// Raw wraps an already-trusted markup string in a SafeBuffer without escaping
// it. It is the Go spelling of String#html_safe / ActionView's raw helper: use
// it only for markup you have produced or vetted yourself.
func Raw(s string) SafeBuffer { return SafeBuffer{s: s} }

// NewSafeBuffer returns an empty SafeBuffer, the equivalent of
// ActionView::OutputBuffer.new. The zero value works too; this exists for
// readability at call sites that build up a buffer.
func NewSafeBuffer() *SafeBuffer { return &SafeBuffer{} }

// String returns the buffer's contents (ActiveSupport::SafeBuffer#to_s). The
// result is the raw markup with no further escaping.
func (b SafeBuffer) String() string { return b.s }

// HTMLSafe reports that the value is html-safe. It always returns true for a
// SafeBuffer, mirroring ActiveSupport::SafeBuffer#html_safe? which is always
// true once a string has been marked safe.
func (b SafeBuffer) HTMLSafe() bool { return true }

// Len reports the byte length of the buffer, like String#length would on the
// underlying bytes. It is provided so callers can cheaply test for emptiness.
func (b SafeBuffer) Len() int { return len(b.s) }

// Concat appends value, html-escaping it first (ActiveSupport::SafeBuffer#<<
// / #concat with an unsafe argument). Use it for untrusted content: the raw
// characters are escaped exactly once. It returns the receiver so calls chain.
func (b *SafeBuffer) Concat(value string) *SafeBuffer {
	b.s += erb.HTMLEscape(value)
	return b
}

// SafeConcat appends value verbatim, without escaping
// (ActiveSupport::SafeBuffer#safe_concat). value MUST already be html-safe
// markup; passing untrusted data through SafeConcat is an XSS bug. It returns
// the receiver so calls chain.
func (b *SafeBuffer) SafeConcat(value string) *SafeBuffer {
	b.s += value
	return b
}

// AppendSafe appends another SafeBuffer's already-safe contents verbatim, the
// safe-vs-safe case of ActiveSupport::SafeBuffer#concat. It returns the
// receiver so calls chain.
func (b *SafeBuffer) AppendSafe(other SafeBuffer) *SafeBuffer {
	b.s += other.s
	return b
}

// safeString is the internal escaping policy shared by the helpers: an
// already-safe SafeBuffer contributes its raw contents, any other value is
// stringified and html-escaped. It centralises the "escape unless safe" rule so
// every helper treats html_safe input identically to MRI.
func safeString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case SafeBuffer:
		return x.s
	case *SafeBuffer:
		if x == nil {
			return ""
		}
		return x.s
	case string:
		return erb.HTMLEscape(x)
	default:
		return erb.HTMLEscape(toS(v))
	}
}
