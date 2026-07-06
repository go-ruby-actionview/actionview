// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"strings"

	"github.com/go-ruby-activesupport/activesupport/inflector"
)

// utf8EnforcerTag is the hidden field form_tag / form_with emit to force browsers
// to submit as UTF-8. ActionView writes it as raw markup so the "✓" stays an
// HTML entity rather than being re-escaped, so it is reproduced verbatim here.
const utf8EnforcerTag = `<input type="hidden" name="utf8" value="&#x2713;" autocomplete="off" />`

// sanitizeToID converts a field name to an HTML id, mirroring ActionView's
// sanitize_to_id: drop "]" and replace every character outside [-a-zA-Z0-9:._]
// with "_". So user[name] becomes user_name.
func sanitizeToID(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == ']':
			// dropped
		case r == '-' || r == '.' || r == ':' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// mergeOpts appends the caller's attributes after a helper's fixed leading
// attributes, matching ActionView's `{ ... }.update(options)` idiom that lets
// user options override/extend the defaults while preserving emission order.
func mergeOpts(base Attrs, extra Attrs) Attrs {
	out := make(Attrs, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

// TextFieldTag renders <input type="text">, mirroring ActionView's
// text_field_tag: name, id (sanitized from name) and value, then any extra
// attributes. A nil value is omitted.
func TextFieldTag(name string, value any, opts Attrs) SafeBuffer {
	return fieldInput("text", name, value, opts)
}

// PasswordFieldTag renders <input type="password">, ActionView's
// password_field_tag. A nil value is omitted (the password is not echoed back).
func PasswordFieldTag(name string, value any, opts Attrs) SafeBuffer {
	return fieldInput("password", name, value, opts)
}

// fieldInput is the shared builder for the simple typed text inputs: it emits
// type, name, id and value in ActionView's order, followed by the caller's opts.
func fieldInput(fieldType, name string, value any, opts Attrs) SafeBuffer {
	base := Attrs{{"type", fieldType}, {"name", name}, {"id", sanitizeToID(name)}}
	if value != nil {
		base = append(base, Attr{"value", value})
	}
	return TagOpen("input", mergeOpts(base, opts), false)
}

// HiddenFieldTag renders <input type="hidden">, ActionView's hidden_field_tag,
// including the trailing autocomplete="off" ActionView adds to hidden fields.
func HiddenFieldTag(name string, value any, opts Attrs) SafeBuffer {
	base := Attrs{{"type", "hidden"}, {"name", name}, {"id", sanitizeToID(name)}, {"value", value}}
	return TagOpen("input", append(mergeOpts(base, opts), Attr{"autocomplete", "off"}), false)
}

// TextAreaTag renders a <textarea>, ActionView's text_area_tag. Content is
// html-escaped unless it is a SafeBuffer, and the tag carries the leading
// newline ActionView inserts before textarea content.
func TextAreaTag(name string, content any, opts Attrs) SafeBuffer {
	base := Attrs{{"name", name}, {"id", sanitizeToID(name)}}
	return ContentTag("textarea", content, mergeOpts(base, opts))
}

// CheckBoxTag renders <input type="checkbox">, ActionView's check_box_tag. value
// defaults to "1" when empty; checked adds checked="checked".
func CheckBoxTag(name, value string, checked bool, opts Attrs) SafeBuffer {
	if value == "" {
		value = "1"
	}
	base := Attrs{{"type", "checkbox"}, {"name", name}, {"id", sanitizeToID(name)}, {"value", value}}
	if checked {
		base = append(base, Attr{"checked", true})
	}
	return TagOpen("input", mergeOpts(base, opts), false)
}

// RadioButtonTag renders <input type="radio">, ActionView's radio_button_tag.
// The id combines the sanitized name and value (name_value) so a group of radios
// gets distinct ids.
func RadioButtonTag(name, value string, checked bool, opts Attrs) SafeBuffer {
	id := sanitizeToID(name) + "_" + sanitizeToID(value)
	base := Attrs{{"type", "radio"}, {"name", name}, {"id", id}, {"value", value}}
	if checked {
		base = append(base, Attr{"checked", true})
	}
	return TagOpen("input", mergeOpts(base, opts), false)
}

// SelectTag renders a <select> around the given option markup, ActionView's
// select_tag. optionTags must already be html-safe option elements. When
// includeBlank is true a leading blank option is prepended.
func SelectTag(name string, optionTags SafeBuffer, includeBlank bool, opts Attrs) SafeBuffer {
	inner := optionTags.String()
	if includeBlank {
		inner = `<option value="" label=" "></option>` + inner
	}
	base := Attrs{{"name", name}, {"id", sanitizeToID(name)}}
	return ContentTag("select", Raw(inner), mergeOpts(base, opts))
}

// LabelTag renders a <label>, ActionView's label_tag. The for attribute is the
// sanitized name; the text defaults to the humanized name when empty.
func LabelTag(name, text string, opts Attrs) SafeBuffer {
	if text == "" {
		text = inflector.Humanize(name)
	}
	base := Attrs{{"for", sanitizeToID(name)}}
	return ContentTag("label", text, mergeOpts(base, opts))
}

// SubmitTag renders <input type="submit">, ActionView's submit_tag. value
// defaults to "Save changes"; a data-disable-with attribute defaulting to the
// value is added to prevent double submission.
func SubmitTag(value string, opts Attrs) SafeBuffer {
	if value == "" {
		value = "Save changes"
	}
	base := Attrs{{"type", "submit"}, {"name", "commit"}, {"value", value},
		{"data-disable-with", value}}
	return TagOpen("input", mergeOpts(base, opts), false)
}

// ButtonTag renders a <button type="submit">, ActionView's button_tag. content
// defaults to "Save changes" when nil.
func ButtonTag(content any, opts Attrs) SafeBuffer {
	if content == nil {
		content = Raw("Save changes")
	}
	base := Attrs{{"name", "button"}, {"type", "submit"}}
	return ContentTag("button", content, mergeOpts(base, opts))
}

// extraTagsForForm builds the hidden fields that follow a form's opening tag:
// the utf8 enforcer (unless suppressed), a _method override for verbs other than
// get/post, and a CSRF token field when forgery protection is enabled. It also
// reports the actual form method attribute ("get" or "post").
func (c *Context) extraTagsForForm(method string) (formMethod string, extra SafeBuffer) {
	method = strings.ToLower(method)
	if method == "" {
		method = "post"
	}
	formMethod = "post"
	if method == "get" {
		formMethod = "get"
	}
	if !c.SuppressUTF8Enforcer {
		extra.SafeConcat(utf8EnforcerTag)
	}
	if formMethod == "post" && method != "post" {
		extra.AppendSafe(hiddenInput("_method", method))
	}
	if formMethod == "post" && c.ProtectAgainstForgery {
		extra.AppendSafe(hiddenInput("authenticity_token", c.AuthenticityToken))
	}
	return formMethod, extra
}

// FormTagOpen renders a form's opening tag plus its hidden enforcer/method/CSRF
// fields, without a closing tag — ActionView's form_tag called without a block.
// url becomes the action; method selects the HTTP verb.
func (c *Context) FormTagOpen(url, method string, opts Attrs) SafeBuffer {
	formMethod, extra := c.extraTagsForForm(method)
	base := Attrs{{"action", url}, {"accept-charset", "UTF-8"}, {"method", formMethod}}
	open := TagOpen("form", mergeOpts(base, opts), true)
	open.AppendSafe(extra)
	return open
}

// FormTag renders a complete form around content, ActionView's form_tag with a
// block: the opening tag, hidden fields, the body, and the closing </form>.
func (c *Context) FormTag(url, method string, opts Attrs, content SafeBuffer) SafeBuffer {
	out := c.FormTagOpen(url, method, opts)
	out.AppendSafe(content)
	out.SafeConcat("</form>")
	return out
}
