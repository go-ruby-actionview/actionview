// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"errors"
	"strings"
)

// ErrNoRenderTemplate is returned by Render when the context has no
// RenderTemplate seam wired up, so there is nothing to evaluate a template with.
var ErrNoRenderTemplate = errors.New("actionview: Context.RenderTemplate is nil")

// PartialIteration is the per-element iteration state ActionView exposes to a
// collection partial as the <name>_iteration local. Index is zero-based and Size
// is the collection length.
type PartialIteration struct {
	Index int
	Size  int
}

// First reports whether this is the first element (index 0), like
// ActionView::PartialIteration#first?.
func (p PartialIteration) First() bool { return p.Index == 0 }

// Last reports whether this is the last element, like
// ActionView::PartialIteration#last?.
func (p PartialIteration) Last() bool { return p.Index == p.Size-1 }

// RenderOptions selects what to render, mirroring the keyword forms of
// ActionView's render. Exactly one of Inline, Template, Partial (optionally with
// Collection) should be set; Locals are passed to the evaluated template. This
// package owns the lookup, partial-name and collection-iteration logic; the
// actual template evaluation is delegated to Context.RenderTemplate.
type RenderOptions struct {
	// Inline is a template source rendered directly (render(inline: "...")).
	Inline string
	// Template is a full-template identifier (render(template: "...")).
	Template string
	// Partial is a partial identifier (render(partial: "...")).
	Partial string
	// Collection renders Partial once per element (render(collection: [...],
	// partial: "...")), exposing each element plus its _counter and _iteration.
	Collection []any
	// Locals are exposed to every rendered template.
	Locals map[string]any
	// As overrides the local name a collection element is bound to; it defaults
	// to the partial's base name.
	As string
	// Spacer is html-safe markup inserted between collection elements
	// (render(..., spacer_template:)).
	Spacer string
}

// Render evaluates a template, partial, inline source, or partial collection and
// returns the html-safe result. It resolves identifiers and, for collections,
// iterates the elements while injecting the element, its zero-based _counter and
// its _iteration locals — then defers the actual template evaluation to the
// context's RenderTemplate seam. It returns ErrNoRenderTemplate when no seam is
// configured, and propagates any error the seam returns.
func (c *Context) Render(opts RenderOptions) (SafeBuffer, error) {
	if c.RenderTemplate == nil {
		return SafeBuffer{}, ErrNoRenderTemplate
	}
	switch {
	case opts.Inline != "":
		return c.evalTemplate(opts.Inline, opts.Locals)
	case opts.Template != "":
		return c.evalTemplate(opts.Template, opts.Locals)
	case opts.Collection != nil:
		return c.renderCollection(opts)
	case opts.Partial != "":
		return c.evalTemplate(opts.Partial, opts.Locals)
	default:
		return SafeBuffer{}, errors.New("actionview: Render requires one of Inline, Template, Partial")
	}
}

// evalTemplate calls the RenderTemplate seam and wraps its output as html-safe.
func (c *Context) evalTemplate(identifier string, locals map[string]any) (SafeBuffer, error) {
	s, err := c.RenderTemplate(identifier, locals)
	if err != nil {
		return SafeBuffer{}, err
	}
	return Raw(s), nil
}

// renderCollection renders the partial once per collection element, binding the
// element to its local name, an <name>_counter (zero-based) and an
// <name>_iteration, joining the results with the spacer.
func (c *Context) renderCollection(opts RenderOptions) (SafeBuffer, error) {
	name := collectionLocalName(opts.Partial, opts.As)
	size := len(opts.Collection)
	parts := make([]string, 0, size)
	for i, item := range opts.Collection {
		locals := mergeLocals(opts.Locals, map[string]any{
			name:                item,
			name + "_counter":   i,
			name + "_iteration": PartialIteration{Index: i, Size: size},
		})
		s, err := c.RenderTemplate(opts.Partial, locals)
		if err != nil {
			return SafeBuffer{}, err
		}
		parts = append(parts, s)
	}
	return Raw(strings.Join(parts, opts.Spacer)), nil
}

// collectionLocalName picks the local a collection element is bound to: the
// explicit As, otherwise the partial's base name (the segment after the last
// "/", with any leading underscore stripped).
func collectionLocalName(partial, as string) string {
	if as != "" {
		return as
	}
	base := partial
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimPrefix(base, "_")
}

// mergeLocals returns a new map combining base and extra, with extra taking
// precedence, so per-element collection locals never mutate the caller's map.
func mergeLocals(base, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
