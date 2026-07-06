// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"math"
	"testing"
)

type stringerT struct{ s string }

func (s stringerT) String() string { return s.s }

func TestToS(t *testing.T) {
	var np *SafeBuffer
	sb := Raw("sb")
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"s", "s"},
		{sb, "sb"},
		{&sb, "sb"},
		{np, ""},
		{true, "true"},
		{false, "false"},
		{int(1), "1"},
		{int8(2), "2"},
		{int16(3), "3"},
		{int32(4), "4"},
		{int64(5), "5"},
		{uint(6), "6"},
		{uint8(7), "7"},
		{uint16(8), "8"},
		{uint32(9), "9"},
		{uint64(10), "10"},
		{float32(1.5), "1.5"},
		{float64(2.5), "2.5"},
		{stringerT{"x"}, "x"},
		{[]int{1}, ""},
	}
	for _, c := range cases {
		if got := toS(c.in); got != c.want {
			t.Errorf("toS(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFloatToS(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
		{0, "0.0"},
		{1, "1.0"},
		{1.5, "1.5"},
		{1234567.891, "1234567.891"},
		{1e16, "1.0e+16"},
		{1e-5, "1.0e-05"},
		{1.5e20, "1.5e+20"},
	}
	for _, c := range cases {
		if got := floatToS(c.in); got != c.want {
			t.Errorf("floatToS(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToFloat(t *testing.T) {
	if _, ok := toFloat("nope"); ok {
		t.Fatal("bad string should not parse")
	}
	if _, ok := toFloat([]int{}); ok {
		t.Fatal("slice should not coerce")
	}
	for _, in := range []any{
		int(1), int8(1), int16(1), int32(1), int64(1),
		uint(1), uint8(1), uint16(1), uint32(1), uint64(1),
		float32(1), float64(1), "1",
	} {
		if f, ok := toFloat(in); !ok || f != 1 {
			t.Errorf("toFloat(%v) = %v,%v", in, f, ok)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{"b": 1, "a": 2, "c": 3}
	got := sortedKeys(m)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("sortedKeys = %v", got)
	}
}
