// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "testing"

func TestLinkTo(t *testing.T) {
	eq(t, LinkTo("Home", "/home", nil), `<a href="/home">Home</a>`)
	eq(t, LinkTo(nil, "/home", nil), `<a href="/home">/home</a>`)
	eq(t, LinkTo(Raw("Del"), "/p/1", Attrs{{"method", "delete"}, {"class", "x"}}),
		`<a class="x" rel="nofollow" data-method="delete" href="/p/1">Del</a>`)
	eq(t, LinkTo("G", "/g", Attrs{{"method", "get"}}), `<a href="/g">G</a>`)
	eq(t, LinkTo("R", "/r", Attrs{{"remote", true}, {"rel", "help"}, {"method", "post"}}),
		`<a rel="help" data-remote="true" data-method="post" href="/r">R</a>`)
}

func TestMailTo(t *testing.T) {
	eq(t, MailTo("a@b.com", "Email", Attrs{{"cc", "c@d.com"}, {"subject", "Hi There"}, {"bcc", "  "}}),
		`<a href="mailto:a@b.com?cc=c%40d.com&amp;subject=Hi%20There">Email</a>`)
	eq(t, MailTo("a@b.com", nil, nil), `<a href="mailto:a@b.com">a@b.com</a>`)
	eq(t, MailTo("a@b.com", "E", Attrs{{"class", "m"}}), `<a class="m" href="mailto:a@b.com">E</a>`)
}

func TestCurrentPage(t *testing.T) {
	c := &Context{RequestPath: "/foo", RequestFullpath: "/foo?x=1", RequestMethod: "get"}
	if !c.CurrentPage("/foo") {
		t.Error("should match path")
	}
	if !c.CurrentPage("/foo?x=1") {
		t.Error("should match fullpath when query present")
	}
	if c.CurrentPage("/bar") {
		t.Error("should not match")
	}
	post := &Context{RequestPath: "/foo", RequestMethod: "post"}
	if post.CurrentPage("/foo") {
		t.Error("non-get method should be false")
	}
	def := &Context{RequestPath: "/x/", RequestMethod: ""}
	if !def.CurrentPage("/x") {
		t.Error("trailing slash should be ignored, default method get")
	}
}

func TestTrimTrailingSlash(t *testing.T) {
	if trimTrailingSlash("/") != "/" {
		t.Error("root slash preserved")
	}
	if trimTrailingSlash("/a/") != "/a" {
		t.Error("trailing removed")
	}
	if trimTrailingSlash("/a") != "/a" {
		t.Error("no slash unchanged")
	}
}

func TestButtonTo(t *testing.T) {
	c := &Context{}
	eq(t, c.ButtonTo("Delete", "/posts/1", "delete", nil),
		`<form class="button_to" method="post" action="/posts/1"><input type="hidden" name="_method" value="delete" autocomplete="off" /><input type="submit" value="Delete" /></form>`)
	eq(t, c.ButtonTo(nil, "/x", "", nil),
		`<form class="button_to" method="post" action="/x"><input type="submit" value="/x" /></form>`)
	eq(t, c.ButtonTo("Go", "/g", "get", nil),
		`<form class="button_to" method="get" action="/g"><input type="submit" value="Go" /></form>`)
	csrf := &Context{ProtectAgainstForgery: true, AuthenticityToken: "TOK"}
	eq(t, csrf.ButtonTo("Save", "/s", "post", nil),
		`<form class="button_to" method="post" action="/s"><input type="submit" value="Save" /><input type="hidden" name="authenticity_token" value="TOK" autocomplete="off" /></form>`)
}

func TestFormTagHelpers(t *testing.T) {
	eq(t, TextFieldTag("q", nil, Attrs{{"placeholder", "Search"}}), `<input type="text" name="q" id="q" placeholder="Search" />`)
	eq(t, TextFieldTag("name", "val", nil), `<input type="text" name="name" id="name" value="val" />`)
	eq(t, PasswordFieldTag("pw", nil, nil), `<input type="password" name="pw" id="pw" />`)
	eq(t, HiddenFieldTag("token", "abc", nil), `<input type="hidden" name="token" id="token" value="abc" autocomplete="off" />`)
	eq(t, TextAreaTag("body", "content", nil), "<textarea name=\"body\" id=\"body\">\ncontent</textarea>")
	eq(t, CheckBoxTag("agree", "", false, nil), `<input type="checkbox" name="agree" id="agree" value="1" />`)
	eq(t, CheckBoxTag("agree", "yes", true, nil), `<input type="checkbox" name="agree" id="agree" value="yes" checked="checked" />`)
	eq(t, RadioButtonTag("color", "red", false, nil), `<input type="radio" name="color" id="color_red" value="red" />`)
	eq(t, RadioButtonTag("color", "red", true, nil), `<input type="radio" name="color" id="color_red" value="red" checked="checked" />`)
	eq(t, LabelTag("name", "", nil), `<label for="name">Name</label>`)
	eq(t, LabelTag("name", "Full Name", nil), `<label for="name">Full Name</label>`)
	eq(t, SubmitTag("", nil), `<input type="submit" name="commit" value="Save changes" data-disable-with="Save changes" />`)
	eq(t, ButtonTag(Raw("Go"), nil), `<button name="button" type="submit">Go</button>`)
	eq(t, ButtonTag(nil, nil), `<button name="button" type="submit">Save changes</button>`)
	eq(t, SelectTag("c", Raw("<option>x</option>"), false, nil), `<select name="c" id="c"><option>x</option></select>`)
	eq(t, SelectTag("c", Raw("<option>x</option>"), true, nil), `<select name="c" id="c"><option value="" label=" "></option><option>x</option></select>`)
}

func TestSanitizeToID(t *testing.T) {
	if got := sanitizeToID("user[name]"); got != "user_name" {
		t.Errorf("= %q", got)
	}
	if got := sanitizeToID("a.b:c-d"); got != "a.b:c-d" {
		t.Errorf("allowed chars = %q", got)
	}
	if got := sanitizeToID("a b"); got != "a_b" {
		t.Errorf("space = %q", got)
	}
}

func TestFormTagOpenAndClose(t *testing.T) {
	c := &Context{}
	eq(t, c.FormTagOpen("/s", "post", nil),
		`<form action="/s" accept-charset="UTF-8" method="post"><input type="hidden" name="utf8" value="&#x2713;" autocomplete="off" />`)
	eq(t, c.FormTagOpen("/x", "patch", nil),
		`<form action="/x" accept-charset="UTF-8" method="post"><input type="hidden" name="utf8" value="&#x2713;" autocomplete="off" /><input type="hidden" name="_method" value="patch" autocomplete="off" />`)
	eq(t, c.FormTagOpen("/g", "get", nil),
		`<form action="/g" accept-charset="UTF-8" method="get"><input type="hidden" name="utf8" value="&#x2713;" autocomplete="off" />`)
	suppressed := &Context{SuppressUTF8Enforcer: true, ProtectAgainstForgery: true, AuthenticityToken: "T"}
	eq(t, suppressed.FormTag("/s", "", nil, Raw("BODY")),
		`<form action="/s" accept-charset="UTF-8" method="post"><input type="hidden" name="authenticity_token" value="T" autocomplete="off" />BODY</form>`)
}

func TestFormBuilder(t *testing.T) {
	b := FormBuilderFor("user", map[string]any{"name": "Dave", "admin": true, "bio": "hi", "color": "g", "n": 5})
	eq(t, b.TextField("name", Attrs{{"class", "fc"}}), `<input class="fc" type="text" value="Dave" name="user[name]" id="user_name" />`)
	eq(t, b.TextField("email", nil), `<input type="text" name="user[email]" id="user_email" />`)
	eq(t, b.PasswordField("pw", nil), `<input type="password" name="user[pw]" id="user_pw" />`)
	eq(t, b.TextArea("bio", nil), "<textarea name=\"user[bio]\" id=\"user_bio\">\nhi</textarea>")
	eq(t, b.HiddenField("name", nil), `<input autocomplete="off" type="hidden" value="Dave" name="user[name]" id="user_name" />`)
	eq(t, b.Label("name", "", nil), `<label for="user_name">Name</label>`)
	eq(t, b.CheckBox("admin", "", "", nil),
		`<input name="user[admin]" type="hidden" value="0" autocomplete="off" /><input type="checkbox" value="1" checked="checked" name="user[admin]" id="user_admin" />`)
	eq(t, b.CheckBox("missing", "yes", "no", nil),
		`<input name="user[missing]" type="hidden" value="no" autocomplete="off" /><input type="checkbox" value="yes" name="user[missing]" id="user_missing" />`)
	eq(t, b.CheckBox("n", "5", NoUncheckedValue, nil),
		`<input type="checkbox" value="5" checked="checked" name="user[n]" id="user_n" />`)
	eq(t, b.RadioButton("color", "red", nil), `<input type="radio" value="red" name="user[color]" id="user_color_red" />`)
	eq(t, b.RadioButton("color", "g", nil), `<input type="radio" value="g" checked="checked" name="user[color]" id="user_color_g" />`)
	eq(t, b.Submit("", nil), `<input type="submit" name="commit" value="Create User" data-disable-with="Create User" />`)
	eq(t, b.Select("color", []Choice{{"Red", "r"}, {"Green", "g"}}, nil),
		`<select name="user[color]" id="user_color"><option value="r">Red</option>`+"\n"+`<option selected="selected" value="g">Green</option></select>`)
}

func TestFormBuilderPersistedAndNilObject(t *testing.T) {
	b := &FormBuilder{ObjectName: "post", ModelName: "Post", Persisted: true}
	eq(t, b.Submit("", nil), `<input type="submit" name="commit" value="Update Post" data-disable-with="Update Post" />`)
	eq(t, b.TextField("title", nil), `<input type="text" name="post[title]" id="post_title" />`)
	if b.FieldName("x") != "post[x]" || b.FieldID("x") != "post_x" {
		t.Error("name/id helpers")
	}
}

func TestFormWith(t *testing.T) {
	c := &Context{}
	out := c.FormWith("user", map[string]any{"name": "A"}, "/users", nil, false, func(b *FormBuilder) SafeBuffer {
		return b.TextField("name", nil)
	})
	eq(t, out, `<form action="/users" accept-charset="UTF-8" method="post"><input type="hidden" name="utf8" value="&#x2713;" autocomplete="off" /><input type="text" value="A" name="user[name]" id="user_name" /></form>`)
	out2 := c.FormWith("user", nil, "/users/1", nil, true, func(b *FormBuilder) SafeBuffer {
		return Raw("")
	})
	if got := out2.String(); got == "" || got[:5] != "<form" {
		t.Errorf("persisted form = %q", got)
	}
}

func TestOptionsAndChoices(t *testing.T) {
	ch := ChoicesFromStrings([]string{"Red", "Green"})
	eq(t, OptionsForSelect(ch, "Green"), `<option value="Red">Red</option>`+"\n"+`<option selected="selected" value="Green">Green</option>`)
}

func TestContextURLFor(t *testing.T) {
	c := &Context{}
	if c.urlFor("/x") != "/x" {
		t.Error("string passthrough")
	}
	if c.urlFor(5) != "5" {
		t.Error("non-string default")
	}
	c2 := &Context{URLFor: func(a any) string { return "/seam" }}
	if c2.urlFor(nil) != "/seam" {
		t.Error("seam")
	}
}
