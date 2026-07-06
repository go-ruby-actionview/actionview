// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"strings"

	"github.com/go-ruby-activesupport/activesupport/inflector"
)

// NoUncheckedValue is the sentinel uncheckedValue that tells CheckBox to omit
// the companion hidden field (ActionView's check_box(..., include_hidden:
// false)). It is a value no real form would use.
const NoUncheckedValue = "\x00__actionview_no_unchecked__"

// Choice is a single <option>: its visible Text and submitted Value. A slice of
// Choice is what OptionsForSelect and the form builder's Select consume.
type Choice struct {
	Text  string
	Value string
}

// ChoicesFromStrings builds a Choice slice where each string is used as both the
// option text and its value, the shape ActionView's options_for_select produces
// from a flat array of strings.
func ChoicesFromStrings(ss []string) []Choice {
	out := make([]Choice, len(ss))
	for i, s := range ss {
		out[i] = Choice{s, s}
	}
	return out
}

// OptionsForSelect renders a run of <option> tags, mirroring ActionView's
// options_for_select. The option whose value equals selected gets
// selected="selected" (emitted before the value attribute, as MRI does). Options
// are joined by newlines; text and value are html-escaped.
func OptionsForSelect(choices []Choice, selected string) SafeBuffer {
	parts := make([]string, len(choices))
	for i, ch := range choices {
		sel := ""
		if ch.Value == selected {
			sel = ` selected="selected"`
		}
		parts[i] = "<option" + sel + ` value="` + HTMLEscape(ch.Value) + `">` + HTMLEscape(ch.Text) + "</option>"
	}
	return Raw(strings.Join(parts, "\n"))
}

// FormBuilder binds form field helpers to a model: an object name (the parameter
// prefix, e.g. "user") and a map of attribute values. It generates the exact
// user[field] name and user_field id conventions ActionView's FormBuilder does.
type FormBuilder struct {
	// ObjectName is the parameter prefix, e.g. "user" -> user[field].
	ObjectName string
	// Object holds the model's attribute values, keyed by field name.
	Object map[string]any
	// Persisted marks an existing record, switching the default submit label
	// from "Create <Model>" to "Update <Model>".
	Persisted bool
	// ModelName is the humanized model label used in the default submit value;
	// it defaults to the humanized ObjectName.
	ModelName string
}

// FormBuilderFor returns a FormBuilder for the given object name and attribute
// map, seeding ModelName from the object name.
func FormBuilderFor(objectName string, object map[string]any) *FormBuilder {
	return &FormBuilder{
		ObjectName: objectName,
		Object:     object,
		ModelName:  inflector.Humanize(objectName),
	}
}

// FieldName returns the HTML name for a field, e.g. user[email] — the
// convention Rails params parsing expects.
func (b *FormBuilder) FieldName(method string) string {
	return b.ObjectName + "[" + method + "]"
}

// FieldID returns the HTML id for a field, e.g. user_email, derived from the
// field name via sanitize_to_id.
func (b *FormBuilder) FieldID(method string) string {
	return sanitizeToID(b.FieldName(method))
}

// value returns the object's stored value for method, or nil when absent.
func (b *FormBuilder) value(method string) any {
	if b.Object == nil {
		return nil
	}
	return b.Object[method]
}

// TextField renders a model-bound <input type="text">, ActionView's
// FormBuilder#text_field: caller options first, then type and value, then the
// derived name and id (name/id always come last).
func (b *FormBuilder) TextField(method string, opts Attrs) SafeBuffer {
	return b.typedField("text", method, opts)
}

// PasswordField renders a model-bound <input type="password">. The stored value
// is not echoed back.
func (b *FormBuilder) PasswordField(method string, opts Attrs) SafeBuffer {
	out := append(cloneAttrs(opts), Attr{"type", "password"})
	out = append(out, Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	return TagOpen("input", out, false)
}

// typedField is the shared builder for value-carrying model inputs.
func (b *FormBuilder) typedField(fieldType, method string, opts Attrs) SafeBuffer {
	out := append(cloneAttrs(opts), Attr{"type", fieldType})
	if v := b.value(method); v != nil {
		out = append(out, Attr{"value", v})
	}
	out = append(out, Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	return TagOpen("input", out, false)
}

// HiddenField renders a model-bound <input type="hidden">, with the leading
// autocomplete="off" ActionView prepends to hidden fields.
func (b *FormBuilder) HiddenField(method string, opts Attrs) SafeBuffer {
	out := append(cloneAttrs(opts), Attr{"autocomplete", "off"}, Attr{"type", "hidden"})
	out = append(out, Attr{"value", b.value(method)},
		Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	return TagOpen("input", out, false)
}

// TextArea renders a model-bound <textarea>, ActionView's FormBuilder#text_area,
// with the stored value as its (escaped) content.
func (b *FormBuilder) TextArea(method string, opts Attrs) SafeBuffer {
	out := append(cloneAttrs(opts), Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	return ContentTag("textarea", b.value(method), out)
}

// Label renders a model-bound <label>, defaulting the text to the humanized
// field name and pointing for at the field's id.
func (b *FormBuilder) Label(method, text string, opts Attrs) SafeBuffer {
	if text == "" {
		text = inflector.Humanize(method)
	}
	out := append(cloneAttrs(opts), Attr{"for", b.FieldID(method)})
	return ContentTag("label", text, out)
}

// CheckBox renders a model-bound checkbox pair, ActionView's
// FormBuilder#check_box: a hidden field carrying uncheckedValue followed by the
// checkbox carrying checkedValue, with checked set from the stored value. An
// checkedValue defaults to "1" and uncheckedValue to "0" when empty, matching
// ActionView's defaults; pass NoUncheckedValue as uncheckedValue to omit the
// hidden field entirely.
func (b *FormBuilder) CheckBox(method, checkedValue, uncheckedValue string, opts Attrs) SafeBuffer {
	if checkedValue == "" {
		checkedValue = "1"
	}
	if uncheckedValue == "" {
		uncheckedValue = "0"
	}
	checked := b.isChecked(b.value(method), checkedValue)

	var out SafeBuffer
	if uncheckedValue != NoUncheckedValue {
		out.AppendSafe(TagOpen("input", Attrs{
			{"name", b.FieldName(method)}, {"type", "hidden"},
			{"value", uncheckedValue}, {"autocomplete", "off"},
		}, false))
	}
	box := append(cloneAttrs(opts), Attr{"type", "checkbox"}, Attr{"value", checkedValue})
	if checked {
		box = append(box, Attr{"checked", true})
	}
	box = append(box, Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	out.AppendSafe(TagOpen("input", box, false))
	return out
}

// isChecked decides a checkbox's checked state from the stored value: a bool is
// used directly, nil is unchecked, otherwise the value's string form must equal
// checkedValue.
func (b *FormBuilder) isChecked(v any, checkedValue string) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	default:
		return toS(v) == checkedValue
	}
}

// RadioButton renders a model-bound radio input, ActionView's
// FormBuilder#radio_button: checked when the stored value equals tagValue, with
// an id of field_id + "_" + value.
func (b *FormBuilder) RadioButton(method, tagValue string, opts Attrs) SafeBuffer {
	checked := toS(b.value(method)) == tagValue
	id := b.FieldID(method) + "_" + sanitizeToID(tagValue)
	out := append(cloneAttrs(opts), Attr{"type", "radio"}, Attr{"value", tagValue})
	if checked {
		out = append(out, Attr{"checked", true})
	}
	out = append(out, Attr{"name", b.FieldName(method)}, Attr{"id", id})
	return TagOpen("input", out, false)
}

// Select renders a model-bound <select>, ActionView's FormBuilder#select,
// marking the option matching the stored value as selected.
func (b *FormBuilder) Select(method string, choices []Choice, opts Attrs) SafeBuffer {
	options := OptionsForSelect(choices, toS(b.value(method)))
	out := append(cloneAttrs(opts), Attr{"name", b.FieldName(method)}, Attr{"id", b.FieldID(method)})
	return ContentTag("select", options, out)
}

// Submit renders the form's submit button, ActionView's FormBuilder#submit. An
// empty value defaults to "Create <Model>" for a new record or "Update <Model>"
// for a persisted one.
func (b *FormBuilder) Submit(value string, opts Attrs) SafeBuffer {
	if value == "" {
		verb := "Create"
		if b.Persisted {
			verb = "Update"
		}
		value = verb + " " + b.ModelName
	}
	return SubmitTag(value, opts)
}

// cloneAttrs returns a copy of opts so builder methods can append their fixed
// attributes without mutating the caller's slice.
func cloneAttrs(opts Attrs) Attrs {
	out := make(Attrs, 0, len(opts)+4)
	out = append(out, opts...)
	return out
}

// FormWith renders a complete model-bound form, a pragmatic subset of
// ActionView's form_with / form_for: it opens a form at url, exposes a
// FormBuilder to fn to render the fields, and closes the form. An empty method
// defaults to "post" for a new record and "patch" for a persisted one. url is
// resolved through the context's url_for seam.
func (c *Context) FormWith(objectName string, object map[string]any, url string, opts Attrs, persisted bool, fn func(*FormBuilder) SafeBuffer) SafeBuffer {
	method := "post"
	if persisted {
		method = "patch"
	}
	b := FormBuilderFor(objectName, object)
	b.Persisted = persisted
	body := fn(b)
	return c.FormTag(c.urlFor(url), method, opts, body)
}
