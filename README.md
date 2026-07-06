<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-actionview/brand/main/social/go-ruby-actionview-actionview.png" alt="go-ruby-actionview/actionview" width="720"></p>

# actionview — go-ruby-actionview

[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the core of Rails'
[ActionView](https://api.rubyonrails.org/classes/ActionView/Helpers.html)** — the
html-safe output buffer, the `tag` / `url` / `form` / `text` / `number` view
helpers, and a rendering pipeline — faithful to **MRI 4.0.5 / actionview 8.1**
output, **without any Ruby runtime**.

It reproduces ActionView's observable markup **byte-for-byte** where it matters:
`tag_options` attribute rendering, the form field `name`/`id` conventions
(`user[name]` / `user_name`), currency and human number formatting, `link_to`
and `content_tag` markup. It is intended as the ActionView backend for a future
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) binding, but is a
**standalone, reusable** module.

It reuses [go-ruby-activesupport](https://github.com/go-ruby-activesupport/activesupport)
(the inflector and core-ext string helpers) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (HTML / URL escaping), and is a
sibling of [go-ruby-set](https://github.com/go-ruby-set/set) and the rest of the
[go-ruby-* stdlib ecosystem](https://github.com/go-ruby-stdlib).

> **MRI-faithful.** This mirrors ActionView's *observable output*, not its Ruby
> object model. `SafeBuffer` is the html-safe string; the helpers are Go
> functions (and `Context` methods for the request/CSRF-aware ones). Template
> **evaluation** (ERB / Ruby) is intentionally *not* here — it is an injectable
> seam, so this package stays free of any template engine.

## The html-safe buffer

`SafeBuffer` is the foundation (`ActiveSupport::SafeBuffer` /
`ActionView::OutputBuffer`): a string known to be HTML-safe, which **auto-escapes
non-safe fragments on concatenation**.

```go
var b actionview.SafeBuffer
b.Concat("<script>")           // untrusted -> escaped:   &lt;script&gt;
b.SafeConcat("<br>")           // trusted markup -> verbatim: <br>
b.AppendSafe(actionview.Raw("<b>")) // another safe buffer -> verbatim
// b.String() == "&lt;script&gt;<br><b>"
```

Every helper returns a `SafeBuffer` (or, for the formatters, a plain `string`),
and escapes its inputs *unless* you pass a `SafeBuffer` / `Raw`, exactly like
Rails' `raw` / `html_safe`.

## Helpers

```go
// TagHelper
actionview.ContentTag("div", "hi", actionview.Attrs{{"class", "a"}}) // <div class="a">hi</div>
actionview.Tag("br", nil, nil)                                       // <br>
actionview.ContentTag("div", "c", actionview.Attrs{{"data", map[string]any{"user_id": 5}}})
// <div data-user-id="5">c</div>
actionview.TokenList("a", "a", map[string]any{"b": true})            // a b
actionview.CDATASection("x]]>y")

// UrlHelper
actionview.LinkTo("Home", "/home", nil)                             // <a href="/home">Home</a>
actionview.MailTo("a@b.com", "Email", actionview.Attrs{{"subject", "Hi"}})
ctx := &actionview.Context{}
ctx.ButtonTo("Delete", "/posts/1", "delete", nil)

// FormTagHelper / FormHelper — exact name/id conventions
b := actionview.FormBuilderFor("user", map[string]any{"name": "Dave"})
b.TextField("name", nil)   // <input type="text" value="Dave" name="user[name]" id="user_name" />
b.Label("name", "", nil)   // <label for="user_name">Name</label>
b.CheckBox("admin", "", "", nil)

// TextHelper
actionview.Truncate("Once upon a time", 10, "...", " ", true)       // Once...
actionview.SimpleFormat("Hello\n\nWorld", nil, "", nil)
actionview.Pluralize(2, "person", "")                               // 2 people

// NumberHelper
actionview.NumberToCurrency(1234.5)                                 // $1,234.50
actionview.NumberToHuman(1234567)                                   // 1.23 Million
actionview.NumberToHumanSize(1234567)                               // 1.18 MB
actionview.NumberToPercentage(99.5)                                 // 99.500%
```

## Rendering pipeline & the `RenderTemplate` seam

`Context.Render` owns the lookup, partial-name derivation and collection
iteration (`_counter` / `_iteration` locals); the actual **template evaluation is
an injectable seam** (`Context.RenderTemplate`), so you can plug in ERB, Ruby, or
anything else:

```go
ctx := &actionview.Context{
    RenderTemplate: func(id string, locals map[string]any) (string, error) {
        return myEngine.Render(id, locals) // ERB / go-ruby-erb / rbgo / ...
    },
}
out, _ := ctx.Render(actionview.RenderOptions{
    Partial:    "users/_user",
    Collection: []any{userA, userB},   // each element -> user, user_counter, user_iteration
    Spacer:     "<hr>",
})
```

## Fidelity

Helper output is validated **byte-for-byte against the real `actionview` gem**
(8.1 on MRI 4.0.5) in `oracle_test.go` — `content_tag`, `link_to`, `mail_to`,
the number formatters, `truncate` / `simple_format` / `highlight` / `word_wrap`,
and the `FormBuilder` field `name`/`id` conventions all match exactly. CI
installs the gem and runs the diff on the ubuntu/macOS lanes.

Two documented, deterministic divergences: attributes supplied via a Go `map`
are emitted in **sorted-key** order (Go maps have no insertion order; MRI
preserves it — use the ordered `Attrs` slice for exact control), and HTML
**sanitization** is not yet implemented (the `Sanitizer` seam defaults to
identity, i.e. `simple_format` / `highlight` behave as with `sanitize: false`).

## Roadmap (deferred from v0.1)

This is the **foundation**. Deliberately deferred, in rough priority order:

- **Template resolver / LookupContext** — formats, variants, locales, digests,
  the real partial/template file lookup (only the `RenderTemplate` seam exists now).
- **Layouts** and `content_for` / `yield` / `provide`.
- **Fragment / Russian-doll caching.**
- **HTML sanitization** (`SanitizeHelper`, Rails::HTML / Loofah) behind the
  existing `Sanitizer` seam.
- **AssetTagHelper** (image/stylesheet/javascript tags, the asset pipeline).
- **FormOptionsHelper** beyond the basics (grouped/collection selects, time zones).
- **DateHelper** (`distance_of_time_in_words`, date/time selects).
- **TranslationHelper / i18n** (`t` / `l`, locale-driven number & date formats).
- **Streaming** (`render stream:`) and **CSP nonce** helpers.
- **Branding** (family banner / logo / favicon) push.

## Tests & coverage

```console
$ GOWORK=off CGO_ENABLED=0 go test -race \
    -coverpkg=$(go list ./... | paste -sd, -) -coverprofile=cover.out ./...
$ go tool cover -func=cover.out | tail -1
total:   (statements)   100.0%
```

**100% line coverage** is enforced in CI, across the six supported 64-bit
targets (amd64, arm64, riscv64, loong64, ppc64le, s390x — including big-endian).
The MRI oracle tests skip themselves where ruby or the gem is unavailable, so the
deterministic suite alone holds the gate.

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright (c) 2026, the
go-ruby-actionview/actionview authors.
