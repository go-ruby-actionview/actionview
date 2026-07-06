// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"math"
	"testing"
)

func TestNumberEdgeBranches(t *testing.T) {
	// digitCount a<1 branch (significant rounding of a sub-1 value).
	if got := NumberWithPrecision(0.0456, Precision(2), Significant(true)); got != "0.046" {
		t.Errorf("sub-1 significant = %q", got)
	}
	// roundHalfUp negative + decimalParts negative (places<=0 and places>0).
	if got := NumberWithPrecision(-1.5, Precision(0)); got != "-2" {
		t.Errorf("neg round places<=0 = %q", got)
	}
	if got := NumberWithPrecision(-1.5, Precision(2)); got != "-1.50" {
		t.Errorf("neg round places>0 = %q", got)
	}
	if got := NumberWithPrecision(-1234567, Precision(2), Significant(true)); got != "-1200000" {
		t.Errorf("neg significant = %q", got)
	}
	// NumberToHumanSize exponent clamp above the largest unit.
	if got := NumberToHumanSize(math.Pow(1024, 9)); got != "1050000 ZB" {
		t.Errorf("clamp = %q", got)
	}
	// numRawString float32 path.
	if got := NumberWithDelimiter(float32(1234.5)); got != "1,234.5" {
		t.Errorf("float32 = %q", got)
	}
	// digitCount zero path via NumberToHuman(0) already covered; hit >=1 too.
	if got := NumberToHuman(5); got != "5" {
		t.Errorf("human small = %q", got)
	}
}

func TestTruthyBranches(t *testing.T) {
	// nil on a boolean attribute -> dropped (truthy(nil) == false).
	eq(t, Tag("input", nil, Attrs{{"disabled", nil}}), `<input>`)
	// non-bool non-nil truthy -> emitted.
	eq(t, Tag("input", nil, Attrs{{"disabled", "x"}}), `<input disabled="disabled">`)
}

func TestFirstLastNBounds(t *testing.T) {
	if got := firstN([]int{1, 2, 3}, -1); len(got) != 0 {
		t.Errorf("firstN neg = %v", got)
	}
	if got := lastN([]int{1, 2, 3}, 9); len(got) != 3 {
		t.Errorf("lastN overflow = %v", got)
	}
}

func TestRenderCollectionWithLocals(t *testing.T) {
	c := &Context{RenderTemplate: func(id string, l map[string]any) (string, error) {
		return toS(l["prefix"]) + toS(l["item"]), nil
	}}
	out, err := c.Render(RenderOptions{
		Partial:    "item",
		Collection: []any{"a"},
		Locals:     map[string]any{"prefix": "P"},
	})
	if err != nil || out.String() != "Pa" {
		t.Fatalf("collection with base locals = %q err=%v", out.String(), err)
	}
}
