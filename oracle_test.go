// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyPreamble sets up a view context exposing the ActionView helper modules,
// so an oracle expression can be evaluated exactly as MRI's actionview gem would
// render it. url_for resolves strings to themselves (and anything else to a
// fixed path) so link/form output is deterministic.
const rubyPreamble = `
require "bigdecimal"
require "active_support/all"
require "action_view"
require "ostruct"
class V
  include ActionView::Helpers::TagHelper
  include ActionView::Helpers::UrlHelper
  include ActionView::Helpers::TextHelper
  include ActionView::Helpers::NumberHelper
  include ActionView::Helpers::FormTagHelper
  include ActionView::Helpers::FormHelper
  include ActionView::Helpers::CaptureHelper
  def protect_against_forgery?; false; end
  def url_for(o); o.is_a?(String) ? o : "/resolved"; end
end
$v = V.new
$stdout.binmode
print(eval(ARGV[0]))
`

// rubyAV locates a ruby that can load the actionview gem, skipping the oracle
// when ruby or the gem is absent (the qemu cross-arch and Windows CI lanes). The
// deterministic tests alone keep coverage at 100%, so skipping here never fails
// the gate.
func rubyAV(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI actionview oracle")
	}
	if err := exec.Command(bin, "-e", `require "action_view"`).Run(); err != nil {
		t.Skip("actionview gem not installed; skipping MRI oracle")
	}
	return bin
}

// avOracle evaluates a Ruby expression in the helper context and returns its
// rendered output.
func avOracle(t *testing.T, bin, expr string) string {
	t.Helper()
	out, err := exec.Command(bin, "-e", rubyPreamble, "--", expr).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error for %q: %v\n%s", expr, err, out)
	}
	return string(out)
}

// TestOracleHelpers diffs a representative expression from every helper family
// against MRI's actionview, byte-for-byte. The Go value on the left and the Ruby
// expression on the right must render identically.
func TestOracleHelpers(t *testing.T) {
	bin := rubyAV(t)
	cases := []struct {
		got  string
		expr string
	}{
		{ContentTag("div", "hi", Attrs{{"class", "a"}}).String(), `$v.content_tag(:div, "hi", class: "a")`},
		{ContentTag("p", "<script>", nil).String(), `$v.content_tag(:p, "<script>")`},
		{Tag("br", nil, nil).String(), `$v.tag.br`},
		{Tag("input", nil, Attrs{{"type", "text"}, {"disabled", true}}).String(), `$v.tag.input(type: "text", disabled: true)`},
		{ContentTag("div", "c", Attrs{{"data", map[string]any{"user_id": 5}}}).String(), `$v.content_tag(:div, "c", data: {user_id: 5})`},
		{TokenList("a", "a", "b").String(), `$v.token_list("a", "a", "b")`},
		{CDATASection("x]]>y").String(), `$v.cdata_section("x]]>y")`},

		{LinkTo("Home", "/home", nil).String(), `$v.link_to("Home", "/home")`},
		{LinkTo(Raw("Del"), "/p/1", Attrs{{"method", "delete"}, {"class", "x"}}).String(),
			`$v.link_to("Del", "/p/1", method: :delete, class: "x")`},
		{MailTo("a@b.com", "Email", Attrs{{"cc", "c@d.com"}, {"subject", "Hi There"}}).String(),
			`$v.mail_to("a@b.com", "Email", cc: "c@d.com", subject: "Hi There")`},

		{NumberToCurrency(1234.5), `$v.number_to_currency(1234.5)`},
		{NumberToCurrency(-1234.5), `$v.number_to_currency(-1234.5)`},
		{NumberToPercentage(99.5), `$v.number_to_percentage(99.5)`},
		{NumberWithDelimiter(1234567.891), `$v.number_with_delimiter(1234567.891)`},
		{NumberWithPrecision(111.2345, Precision(2)), `$v.number_with_precision(111.2345, precision: 2)`},
		{NumberToHumanSize(1234567), `$v.number_to_human_size(1234567)`},
		{NumberToHuman(1234567), `$v.number_to_human(1234567)`},

		{Truncate("Once upon a time in a world far far away", 17, "...", "", true).String(),
			`$v.truncate("Once upon a time in a world far far away", length: 17)`},
		{SimpleFormat("Hello\n\nWorld\nLine", nil, "", nil).String(), `$v.simple_format("Hello\n\nWorld\nLine")`},
		{Pluralize(2, "person", ""), `$v.pluralize(2, "person")`},
		{Highlight("You searched for foo", []string{"foo"}, "", nil).String(), `$v.highlight("You searched for foo", "foo")`},
		{WordWrap("The quick brown fox", 10, ""), `$v.word_wrap("The quick brown fox", line_width: 10)`},

		{TextFieldTag("name", "val", nil).String(), `$v.text_field_tag("name", "val")`},
		{CheckBoxTag("agree", "", false, nil).String(), `$v.check_box_tag("agree")`},
		{SubmitTag("", nil).String(), `$v.submit_tag`},
		{HiddenFieldTag("token", "abc", nil).String(), `$v.hidden_field_tag("token", "abc")`},
	}
	for _, c := range cases {
		want := avOracle(t, bin, c.expr)
		if c.got != want {
			t.Errorf("oracle mismatch\n  expr: %s\n   got: %q\n  want: %q", c.expr, c.got, want)
		}
	}
}

// TestOracleFormBuilder diffs the model-bound form field name/id conventions —
// the exactness that matters most — against MRI's FormBuilder.
func TestOracleFormBuilder(t *testing.T) {
	bin := rubyAV(t)
	setup := `b = ActionView::Helpers::FormBuilder.new(:user, OpenStruct.new(name: "Dave", admin: true, bio: "hi"), $v, {}); `
	b := FormBuilderFor("user", map[string]any{"name": "Dave", "admin": true, "bio": "hi"})
	cases := []struct {
		got  string
		expr string
	}{
		{b.TextField("name", nil).String(), setup + `b.text_field(:name)`},
		{b.TextArea("bio", nil).String(), setup + `b.text_area(:bio)`},
		{b.CheckBox("admin", "", "", nil).String(), setup + `b.check_box(:admin)`},
		{b.RadioButton("color", "red", nil).String(), setup + `b.radio_button(:color, "red")`},
		{b.Label("name", "", nil).String(), setup + `b.label(:name)`},
		{b.HiddenField("name", nil).String(), setup + `b.hidden_field(:name)`},
		{b.Submit("", nil).String(), setup + `b.submit`},
	}
	for _, c := range cases {
		want := avOracle(t, bin, c.expr)
		if c.got != want {
			t.Errorf("form-builder mismatch\n  expr: %s\n   got: %q\n  want: %q",
				strings.TrimPrefix(c.expr, setup), c.got, want)
		}
	}
}
