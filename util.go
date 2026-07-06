// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// toS converts an arbitrary Go value to a string the way Ruby's #to_s would,
// so helper output matches MRI for the common scalar types passed to view
// helpers. It deliberately does not escape; escaping is a separate concern
// handled by safeString / the tag builders.
func toS(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case SafeBuffer:
		return x.s
	case *SafeBuffer:
		if x == nil {
			return ""
		}
		return x.s
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return floatToS(float64(x))
	case float64:
		return floatToS(x)
	case Stringer:
		return x.String()
	default:
		return ""
	}
}

// Stringer mirrors fmt.Stringer; a value that implements it renders through its
// String method, letting callers pass custom types (e.g. model ids) that know
// their own Ruby-facing representation.
type Stringer interface{ String() string }

// floatToS renders a float the way Ruby's Float#to_s does for the values view
// helpers care about: the shortest round-tripping decimal, in fixed notation
// when the decimal exponent lies in [-4, 16) (so 1234567.891 stays fixed) and in
// exponent notation otherwise (1.0e+16), always keeping a decimal point.
func floatToS(f float64) string {
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if math.IsNaN(f) {
		return "NaN"
	}
	if f == 0 {
		return "0.0"
	}
	exp := int(math.Floor(math.Log10(math.Abs(f))))
	if exp >= -4 && exp < 16 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
		return s
	}
	// Exponent form, matching Ruby's "1.0e+16": ensure a decimal point in the
	// mantissa and a sign on the exponent.
	s := strconv.FormatFloat(f, 'e', -1, 64)
	mant, e, _ := strings.Cut(s, "e")
	if !strings.ContainsRune(mant, '.') {
		mant += ".0"
	}
	return mant + "e" + e
}

// toFloat coerces a numeric-ish value to float64 for the number helpers,
// reporting whether the coercion succeeded. Strings are parsed leniently so a
// helper can accept "1234.5" exactly as MRI's Kernel#Float-ish coercion does.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// sortedKeys returns the keys of a string-keyed map in ascending order. The
// helpers use it to emit attributes from Go maps deterministically; see the
// README fidelity note on hash ordering (Ruby preserves insertion order, Go
// maps do not, so this package sorts).
func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
