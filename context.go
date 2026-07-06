// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

// Context is a view context: the small bundle of request/routing/CSRF state and
// injectable seams that the stateful helpers (button_to, form_tag, form_with,
// current_page?, render) need. The pure helpers — tag, url, text and number
// formatting — are package functions and do not require a Context.
//
// The zero value is usable and mirrors a bare ActionView helper include with
// forgery protection disabled: no CSRF hidden field is emitted and url_for is
// the identity on strings. Populate the fields to opt into CSRF tokens, routing,
// current-page detection and template rendering.
type Context struct {
	// URLFor resolves a routing argument to a URL string (the routes seam). When
	// nil, a string argument passes through unchanged and any other value is
	// stringified, so simple string URLs work with no wiring.
	URLFor func(any) string

	// AuthenticityToken is the CSRF token value emitted in the hidden
	// authenticity_token field when ProtectAgainstForgery is true.
	AuthenticityToken string

	// ProtectAgainstForgery toggles emission of the CSRF hidden field in
	// form_tag / form_with / button_to. False (the default) matches a context
	// whose protect_against_forgery? returns false.
	ProtectAgainstForgery bool

	// SuppressUTF8Enforcer drops the hidden utf8 "✓" field that form_tag /
	// form_with emit by default. Leave false to match ActionView's default
	// enforce_utf8 behaviour.
	SuppressUTF8Enforcer bool

	// RequestMethod is the current request's HTTP method symbol (lower-case,
	// e.g. "get"), used by current_page? to decide whether the method matches.
	// Empty is treated as "get".
	RequestMethod string

	// RequestPath and RequestFullpath are the current request's path (without
	// query) and full path (with query), consulted by current_page?.
	RequestPath     string
	RequestFullpath string

	// RenderTemplate is the template-evaluation seam used by Render for
	// :template / :partial / :inline sources and by partial iteration. It
	// receives a resolved identifier and the locals to expose and returns the
	// rendered markup. When nil, Render returns an ErrNoRenderTemplate error.
	RenderTemplate func(identifier string, locals map[string]any) (string, error)
}

// urlFor resolves o to a URL string through the URLFor seam, defaulting to the
// identity on strings when no seam is configured.
func (c *Context) urlFor(o any) string {
	if c.URLFor != nil {
		return c.URLFor(o)
	}
	if s, ok := o.(string); ok {
		return s
	}
	return toS(o)
}
