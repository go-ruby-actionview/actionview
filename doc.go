// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package actionview is a pure-Go (no cgo), MRI-faithful reimplementation of the
// core of Rails' ActionView view layer: the html-safe output buffer, the tag /
// url / form / text / number view helpers, and a rendering pipeline whose
// template evaluation is an injectable seam.
//
// It reproduces ActionView's observable output byte-for-byte where it matters —
// tag_options attribute rendering, form field name/id conventions
// (user[name] / user_name), currency and human number formatting, link_to and
// content_tag markup — without any Ruby runtime. It reuses
// go-ruby-activesupport (inflector, core-ext string helpers) and go-ruby-erb
// (HTML/URL escaping), and is intended as the ActionView backend for a future
// go-embedded-ruby binding.
//
// The pure helpers (tag, text, number, link_to, mail_to) are package functions.
// Helpers that need request, routing or CSRF state — button_to, form_tag,
// form_with, current_page? and Render — are methods on Context, whose zero value
// behaves like a bare helper include with forgery protection disabled. Template
// evaluation (ERB / Ruby) is delegated to Context.RenderTemplate, keeping this
// package free of any template engine.
//
// See the README for the v0.1 surface and the roadmap of deferred pieces (the
// full template resolver with formats/variants/digests, layouts, fragment
// caching, AssetTagHelper, DateHelper, i18n TranslationHelper, streaming and
// HTML sanitization).
package actionview
