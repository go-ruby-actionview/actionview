// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"errors"
	"fmt"
	"testing"
)

// fakeRenderer is a RenderTemplate seam that records its calls and echoes the
// identifier and a couple of interesting locals, so the pipeline's lookup and
// local-injection behaviour can be asserted without a real template engine.
func fakeRenderer(identifier string, locals map[string]any) (string, error) {
	return fmt.Sprintf("[%s|%v]", identifier, locals["greeting"]), nil
}

func TestRenderNoSeam(t *testing.T) {
	c := &Context{}
	if _, err := c.Render(RenderOptions{Inline: "x"}); !errors.Is(err, ErrNoRenderTemplate) {
		t.Fatalf("want ErrNoRenderTemplate, got %v", err)
	}
}

func TestRenderInlineTemplatePartial(t *testing.T) {
	c := &Context{RenderTemplate: fakeRenderer}
	for _, opt := range []RenderOptions{
		{Inline: "inl", Locals: map[string]any{"greeting": "hi"}},
		{Template: "tmpl", Locals: map[string]any{"greeting": "hi"}},
		{Partial: "part", Locals: map[string]any{"greeting": "hi"}},
	} {
		out, err := c.Render(opt)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() == "" {
			t.Fatalf("empty render for %+v", opt)
		}
	}
}

func TestRenderRequiresSource(t *testing.T) {
	c := &Context{RenderTemplate: fakeRenderer}
	if _, err := c.Render(RenderOptions{}); err == nil {
		t.Fatal("expected error for empty options")
	}
}

func TestRenderCollection(t *testing.T) {
	var seen []map[string]any
	c := &Context{RenderTemplate: func(id string, locals map[string]any) (string, error) {
		seen = append(seen, locals)
		return fmt.Sprintf("<%v>", locals["user"]), nil
	}}
	out, err := c.Render(RenderOptions{
		Partial:    "users/_user",
		Collection: []any{"a", "b"},
		Spacer:     "|",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "<a>|<b>" {
		t.Fatalf("collection = %q", out.String())
	}
	if seen[0]["user"] != "a" || seen[0]["user_counter"] != 0 {
		t.Fatalf("locals[0] = %v", seen[0])
	}
	it := seen[1]["user_iteration"].(PartialIteration)
	if it.Index != 1 || it.Size != 2 || !it.Last() || it.First() {
		t.Fatalf("iteration = %+v", it)
	}
	it0 := seen[0]["user_iteration"].(PartialIteration)
	if !it0.First() || it0.Last() {
		t.Fatalf("iteration0 = %+v", it0)
	}
}

func TestRenderCollectionAsAndError(t *testing.T) {
	c := &Context{RenderTemplate: func(id string, locals map[string]any) (string, error) {
		if locals["thing"] == "boom" {
			return "", errors.New("kaboom")
		}
		return "ok", nil
	}}
	if _, err := c.Render(RenderOptions{Partial: "p", Collection: []any{"boom"}, As: "thing"}); err == nil {
		t.Fatal("expected propagated error")
	}
	out, _ := c.Render(RenderOptions{Partial: "p", Collection: []any{"fine"}, As: "thing"})
	if out.String() != "ok" {
		t.Fatalf("as = %q", out.String())
	}
}

func TestEvalTemplateError(t *testing.T) {
	c := &Context{RenderTemplate: func(id string, l map[string]any) (string, error) {
		return "", errors.New("nope")
	}}
	if _, err := c.Render(RenderOptions{Inline: "x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCollectionLocalName(t *testing.T) {
	if collectionLocalName("users/_user", "") != "user" {
		t.Error("path+underscore")
	}
	if collectionLocalName("user", "") != "user" {
		t.Error("bare")
	}
	if collectionLocalName("user", "member") != "member" {
		t.Error("as override")
	}
}
