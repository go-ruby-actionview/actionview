// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import "testing"

func TestNumberToCurrency(t *testing.T) {
	cases := []struct {
		in   any
		opts []Option
		want string
	}{
		{1234.5, nil, "$1,234.50"},
		{-1234.5, nil, "-$1,234.50"},
		{-0.001, nil, "$0.00"}, // negative that rounds to zero keeps positive format
		{1234.5, []Option{Unit("€"), Format("%n %u"), Separator(","), Delimiter(".")}, "1.234,50 €"},
		{1234.5, []Option{NegativeFormat("(%u%n)")}, "$1,234.50"},
		{"12.1", nil, "$12.10"},
		{"-12.1", nil, "-$12.10"},
		{"abc", nil, "$abc"},
		{"-abc", nil, "-$abc"},
	}
	for _, c := range cases {
		if got := NumberToCurrency(c.in, c.opts...); got != c.want {
			t.Errorf("NumberToCurrency(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNumberToPercentage(t *testing.T) {
	if got := NumberToPercentage(99.5); got != "99.500%" {
		t.Errorf("= %q", got)
	}
	if got := NumberToPercentage(100, Precision(0)); got != "100%" {
		t.Errorf("= %q", got)
	}
	if got := NumberToPercentage("x"); got != "x" {
		t.Errorf("bad = %q", got)
	}
}

func TestNumberWithDelimiter(t *testing.T) {
	if got := NumberWithDelimiter(1234567.891); got != "1,234,567.891" {
		t.Errorf("= %q", got)
	}
	if got := NumberToDelimited(12); got != "12" {
		t.Errorf("no-delim = %q", got)
	}
	if got := NumberWithDelimiter(-1234); got != "-1,234" {
		t.Errorf("neg = %q", got)
	}
	if got := NumberWithDelimiter("x"); got != "x" {
		t.Errorf("bad = %q", got)
	}
}

func TestNumberWithPrecision(t *testing.T) {
	if got := NumberWithPrecision(111.2345, Precision(2)); got != "111.23" {
		t.Errorf("= %q", got)
	}
	if got := NumberToRounded(111.0, Precision(2)); got != "111.00" {
		t.Errorf("= %q", got)
	}
	if got := NumberWithPrecision(0.0, Precision(2)); got != "0.00" {
		t.Errorf("zero = %q", got)
	}
	if got := NumberWithPrecision("x"); got != "x" {
		t.Errorf("bad = %q", got)
	}
	// Significant with strip and a rounding that reduces the digit count.
	if got := NumberWithPrecision(9.995, Precision(3), Significant(true), StripInsignificantZeros(true)); got != "10" {
		t.Errorf("sig = %q", got)
	}
}

func TestNumberToHumanSize(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{1234567, "1.18 MB"},
		{1234567890, "1.15 GB"},
		{500, "500 Bytes"},
		{1, "1 Byte"},
		{0, "0 Bytes"},
		{"x", "x"},
	}
	for _, c := range cases {
		if got := NumberToHumanSize(c.in); got != c.want {
			t.Errorf("NumberToHumanSize(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNumberToHuman(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{1234567, "1.23 Million"},
		{489939, "490 Thousand"},
		{500, "500"},
		{0, "0"},
		{"x", "x"},
	}
	for _, c := range cases {
		if got := NumberToHuman(c.in); got != c.want {
			t.Errorf("NumberToHuman(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDecimalPartsNegativeAndPow10(t *testing.T) {
	// Exercise negative rounding places via a large significant number.
	if got := NumberWithPrecision(1234567, Precision(2), Significant(true)); got != "1200000" {
		t.Errorf("neg-places = %q", got)
	}
}

func TestStripInsignificantZerosNoSeparator(t *testing.T) {
	if got := stripInsignificantZeros("100", "."); got != "100" {
		t.Errorf("= %q", got)
	}
	if got := stripInsignificantZeros("1.500", "."); got != "1.5" {
		t.Errorf("= %q", got)
	}
	if got := stripInsignificantZeros("1.000", "."); got != "1" {
		t.Errorf("= %q", got)
	}
}
