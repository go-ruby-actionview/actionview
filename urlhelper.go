// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"strings"

	"github.com/go-ruby-erb/erb"
)

// hiddenInput builds one <input type="hidden" ...> the way the form helpers do,
// with the trailing autocomplete="off" ActionView adds to hidden fields.
func hiddenInput(name, value string) SafeBuffer {
	return TagOpen("input", Attrs{
		{"type", "hidden"}, {"name", name}, {"value", value}, {"autocomplete", "off"},
	}, false)
}

// LinkTo renders an anchor tag, mirroring ActionView's link_to for a resolved
// string URL. The url becomes the href (appended after the given attributes). A
// "method" attribute (other than get) is converted to rel="nofollow" plus
// data-method, and a truthy "remote" attribute becomes data-remote="true",
// matching convert_options_to_data_attributes. content is html-escaped unless it
// is a SafeBuffer; pass nil content to use the url as the link text.
func LinkTo(content any, url string, opts Attrs) SafeBuffer {
	html := make(Attrs, 0, len(opts)+2)
	var method string
	var remote bool
	rel := ""
	for _, a := range opts {
		switch a.Key {
		case "method":
			method = toS(a.Val)
		case "remote":
			remote = truthy(a.Val)
		case "rel":
			rel = toS(a.Val)
			html = append(html, a)
		default:
			html = append(html, a)
		}
	}
	if remote {
		html = append(html, Attr{"data-remote", "true"})
	}
	if method != "" && method != "get" {
		if !strings.Contains(rel, "nofollow") {
			if rel == "" {
				html = append(html, Attr{"rel", "nofollow"})
			}
		}
		html = append(html, Attr{"data-method", method})
	}
	if content == nil {
		content = Raw(url)
	}
	html = append(html, Attr{"href", url})
	return contentTagString("a", content, html, true)
}

// mailToOptionKeys are the query-parameter options mail_to pulls out of its
// option list, in the order ActionView emits them.
var mailToOptionKeys = []string{"cc", "bcc", "body", "subject", "reply_to"}

// MailTo renders a mailto: anchor, mirroring ActionView's mail_to. Recognised
// option keys (cc, bcc, body, subject, reply_to) become url-encoded query
// parameters in a fixed order; any other attributes are rendered on the anchor.
// content defaults to the email address when nil.
func MailTo(email string, content any, opts Attrs) SafeBuffer {
	optMap := map[string]string{}
	html := make(Attrs, 0, len(opts))
	for _, a := range opts {
		if isMailKey(a.Key) {
			if s := toS(a.Val); strings.TrimSpace(s) != "" {
				optMap[a.Key] = s
			}
			continue
		}
		html = append(html, a)
	}
	var extras []string
	for _, k := range mailToOptionKeys {
		if v, ok := optMap[k]; ok {
			extras = append(extras, dasherize(k)+"="+erb.URLEncode(v))
		}
	}
	query := ""
	if len(extras) > 0 {
		query = "?" + strings.Join(extras, "&")
	}
	encoded := strings.ReplaceAll(erb.URLEncode(email), "%40", "@")
	html = append(html, Attr{"href", "mailto:" + encoded + query})
	if content == nil {
		content = Raw(email)
	}
	return contentTagString("a", content, html, true)
}

// isMailKey reports whether key is one of mail_to's query-parameter options.
func isMailKey(key string) bool {
	for _, k := range mailToOptionKeys {
		if k == key {
			return true
		}
	}
	return false
}

// CurrentPage reports whether target refers to the request's current page,
// mirroring ActionView's current_page?. It resolves target through url_for and
// compares it to the request path (or full path when target carries a query),
// ignoring a single trailing slash. It requires RequestPath to be set; when the
// request method does not match RequestMethod it returns false.
func (c *Context) CurrentPage(target any) bool {
	method := c.RequestMethod
	if method == "" {
		method = "get"
	}
	if method != "get" && method != "head" {
		return false
	}
	urlString := c.urlFor(target)
	requestURI := c.RequestPath
	if strings.Contains(urlString, "?") {
		requestURI = c.RequestFullpath
	}
	return trimTrailingSlash(urlString) == trimTrailingSlash(requestURI)
}

// trimTrailingSlash removes a single trailing slash (but never reduces "/" to
// empty), matching current_page?'s remove_trailing_slash! before comparison.
func trimTrailingSlash(s string) string {
	if len(s) > 1 && strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}

// ButtonTo renders a single-button form that submits to url, mirroring
// ActionView's button_to. method selects the HTTP verb: "get" produces a GET
// form, anything else a POST form with a hidden _method override (for verbs
// other than post). When the context enables forgery protection a hidden
// authenticity_token field is added. name is the button label (defaults to url).
func (c *Context) ButtonTo(name any, url string, method string, opts Attrs) SafeBuffer {
	if method == "" {
		method = "post"
	}
	formMethod := "post"
	if method == "get" {
		formMethod = "get"
	}

	var inner SafeBuffer
	if formMethod == "post" && method != "post" && method != "get" {
		inner.AppendSafe(hiddenInput("_method", method))
	}

	label := name
	if label == nil {
		label = Raw(url)
	}
	button := TagOpen("input", append(Attrs{{"type", "submit"}}, appendValue(opts, safeString(label))...), false)

	var b SafeBuffer
	b.AppendSafe(inner)
	b.AppendSafe(button)
	if formMethod == "post" && c.ProtectAgainstForgery {
		b.AppendSafe(hiddenInput("authenticity_token", c.AuthenticityToken))
	}
	formOpts := Attrs{{"class", "button_to"}, {"method", formMethod}, {"action", url}}
	return ContentTag("form", b, formOpts)
}

// appendValue returns opts with a trailing value="v" attribute, used to place
// the submit button's label after any caller-supplied attributes.
func appendValue(opts Attrs, v string) Attrs {
	out := make(Attrs, 0, len(opts)+1)
	out = append(out, opts...)
	out = append(out, Attr{"value", v})
	return out
}
