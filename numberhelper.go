// Copyright (c) the go-ruby-actionview/actionview authors
//
// SPDX-License-Identifier: BSD-3-Clause

package actionview

import (
	"math"
	"math/big"
	"strings"
)

// numberOpts holds the formatting knobs shared by the number helpers, mirroring
// the option hash ActiveSupport::NumberHelper builds from its locale defaults.
// Zero values are never used directly; each helper seeds a full set of defaults
// and callers override individual fields via functional Option values.
type numberOpts struct {
	precision      int
	significant    bool
	separator      string
	delimiter      string
	unit           string
	format         string
	negativeFormat string
	stripZeros     bool
}

// Option overrides a single number-formatting field, letting callers write e.g.
// NumberToCurrency(x, Precision(0), Unit("€")) without a positional option soup.
type Option func(*numberOpts)

// Precision sets the number of digits kept (significant or decimal, per helper).
func Precision(p int) Option { return func(o *numberOpts) { o.precision = p } }

// Significant switches precision between decimal places (false) and significant
// figures (true).
func Significant(b bool) Option { return func(o *numberOpts) { o.significant = b } }

// Separator sets the decimal separator (default ".").
func Separator(s string) Option { return func(o *numberOpts) { o.separator = s } }

// Delimiter sets the thousands delimiter (default "," for most helpers).
func Delimiter(s string) Option { return func(o *numberOpts) { o.delimiter = s } }

// Unit sets the currency/measurement unit string.
func Unit(s string) Option { return func(o *numberOpts) { o.unit = s } }

// Format sets the positive template (with %n for number, %u for unit).
func Format(s string) Option { return func(o *numberOpts) { o.format = s } }

// NegativeFormat sets the template used for negative currency values.
func NegativeFormat(s string) Option { return func(o *numberOpts) { o.negativeFormat = s } }

// StripInsignificantZeros toggles trailing-zero removal after the separator.
func StripInsignificantZeros(b bool) Option { return func(o *numberOpts) { o.stripZeros = b } }

func apply(o numberOpts, opts []Option) numberOpts {
	for _, f := range opts {
		f(&o)
	}
	return o
}

// pow10 returns 10**n as an exact big.Rat, supporting negative n for
// significant-figure rounding to tens/hundreds.
func pow10(n int) *big.Rat {
	ten := big.NewInt(10)
	if n >= 0 {
		return new(big.Rat).SetInt(new(big.Int).Exp(ten, big.NewInt(int64(n)), nil))
	}
	p := new(big.Int).Exp(ten, big.NewInt(int64(-n)), nil)
	return new(big.Rat).SetFrac(big.NewInt(1), p)
}

// ratFromFloat converts a float to the exact big.Rat of its shortest decimal
// representation, mirroring BigDecimal(number.to_s): Ruby rounds through the
// float's #to_s, not its raw binary value, so 1234.5 becomes exactly 1234.5.
func ratFromFloat(f float64) *big.Rat {
	r, _ := new(big.Rat).SetString(floatToS(f))
	return r
}

// digitCount returns ActiveSupport's RoundingHelper#digit_count for a rational:
// 1 for zero, otherwise floor(log10(abs))+1 computed exactly from the decimal
// magnitude so significant-figure rounding lands on the right power of ten.
func digitCount(r *big.Rat) int {
	if r.Sign() == 0 {
		return 1
	}
	a := new(big.Rat).Abs(r)
	one := big.NewRat(1, 1)
	if a.Cmp(one) >= 0 {
		// Number of digits in the integer part.
		return len(new(big.Int).Quo(a.Num(), a.Denom()).String())
	}
	// a < 1: count leading zeros after the decimal point.
	ten := big.NewRat(10, 1)
	z := 0
	t := new(big.Rat).Set(a)
	for t.Cmp(one) < 0 {
		t.Mul(t, ten)
		z++
	}
	return -(z - 1)
}

// roundHalfUp rounds r to places decimal places, half away from zero (the
// BigDecimal :default round mode). places may be negative to round to tens,
// hundreds, and so on.
func roundHalfUp(r *big.Rat, places int) *big.Rat {
	scale := pow10(places)
	scaled := new(big.Rat).Mul(r, scale)
	num := new(big.Int).Abs(scaled.Num())
	den := scaled.Denom()
	q := new(big.Int).Quo(num, den)
	rem := new(big.Int).Mul(new(big.Int).Rem(num, den), big.NewInt(2))
	if rem.Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if scaled.Sign() < 0 {
		q.Neg(q)
	}
	out := new(big.Rat).SetInt(q)
	return out.Quo(out, scale)
}

// decimalParts renders a terminating rational (denominator a power of ten) as a
// signed integer-part string and a fractional-digit string of exactly places
// digits (empty when places <= 0). It is the exact equivalent of splitting
// BigDecimal#to_s("F") on the decimal point after rounding.
func decimalParts(r *big.Rat, places int) (intPart, frac string) {
	neg := r.Sign() < 0
	a := new(big.Rat).Abs(r)
	scaled := new(big.Int).Quo(new(big.Int).Mul(a.Num(), pow10(places).Num()), a.Denom())
	if places <= 0 {
		// places<=0 means the value is integral at this scale.
		whole := new(big.Int).Quo(a.Num(), a.Denom())
		s := whole.String()
		if neg && whole.Sign() != 0 {
			s = "-" + s
		}
		return s, ""
	}
	digits := scaled.String()
	if len(digits) <= places {
		digits = strings.Repeat("0", places-len(digits)+1) + digits
	}
	intPart = digits[:len(digits)-places]
	frac = digits[len(digits)-places:]
	if neg && !(intPart == "0" && strings.Trim(frac, "0") == "") {
		intPart = "-" + intPart
	}
	return intPart, frac
}

// insertDelimiter groups the integer-part digits into threes from the right,
// joined by delim, leaving any leading sign untouched — the behaviour of
// ActiveSupport's DEFAULT_DELIMITER_REGEX.
func insertDelimiter(left, delim string) string {
	sign := ""
	if strings.HasPrefix(left, "-") {
		sign, left = "-", left[1:]
	}
	if delim == "" || len(left) <= 3 {
		return sign + left
	}
	var parts []string
	for len(left) > 3 {
		parts = append([]string{left[len(left)-3:]}, parts...)
		left = left[:len(left)-3]
	}
	parts = append([]string{left}, parts...)
	return sign + strings.Join(parts, delim)
}

// numberToRounded is ActiveSupport's NumberToRoundedConverter: round to the
// requested precision (decimal or significant), re-slice the fractional digits,
// insert the delimiter, and optionally strip trailing zeros. It is the shared
// core of the precision, currency, percentage, human and human_size helpers.
func numberToRounded(f float64, o numberOpts) string {
	r := ratFromFloat(f)
	roundPlaces := o.precision
	if o.significant && o.precision > 0 {
		roundPlaces = o.precision - digitCount(r)
	}
	rounded := roundHalfUp(r, roundPlaces)
	if rounded.Sign() == 0 {
		rounded.Abs(rounded)
	}

	formatPrecision := o.precision
	if o.significant && o.precision > 0 {
		formatPrecision -= digitCount(rounded)
		if formatPrecision < 0 {
			formatPrecision = 0
		}
	}

	rp := roundPlaces
	if rp < 0 {
		rp = 0
	}
	intPart, frac := decimalParts(rounded, rp)
	result := intPart
	if formatPrecision != 0 {
		frac += strings.Repeat("0", formatPrecision)
		result = intPart + "." + frac[:formatPrecision]
	}

	// Delimit the integer part, keep the (already sliced) fraction verbatim.
	left := result
	right := ""
	if i := strings.IndexByte(result, '.'); i >= 0 {
		left, right = result[:i], result[i+1:]
	}
	left = insertDelimiter(left, o.delimiter)
	out := left
	if strings.IndexByte(result, '.') >= 0 {
		out = left + o.separator + right
	}
	if o.stripZeros {
		out = stripInsignificantZeros(out, o.separator)
	}
	return out
}

// stripInsignificantZeros drops trailing zeros (and a now-bare separator) after
// the decimal separator, matching NumberToRoundedConverter#format_number.
func stripInsignificantZeros(s, sep string) string {
	i := strings.Index(s, sep)
	if i < 0 {
		return s
	}
	head, tail := s[:i], s[i+len(sep):]
	tail = strings.TrimRight(tail, "0")
	if tail == "" {
		return head
	}
	return head + sep + tail
}

// numRawString renders v the way Ruby's number.to_s does before delimiting: a
// string passes through, an integer stays integral (no ".0"), and a float uses
// its shortest decimal. This preserves the integer-vs-float distinction that
// NumberWithDelimiter depends on.
func numRawString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float32:
		return floatToS(float64(x))
	case float64:
		return floatToS(x)
	default:
		return toS(v)
	}
}

// NumberWithDelimiter formats a number with grouped thousands and the given
// decimal separator, without rounding — ActionView's number_with_delimiter
// (a.k.a. number_to_delimited). Defaults: delimiter ",", separator ".".
func NumberWithDelimiter(v any, opts ...Option) string {
	o := apply(numberOpts{delimiter: ",", separator: "."}, opts)
	s := numRawString(v)
	left, right := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		left, right = s[:i], s[i+1:]
	}
	left = insertDelimiter(left, o.delimiter)
	if right == "" {
		return left
	}
	return left + o.separator + right
}

// NumberToDelimited is the modern name for NumberWithDelimiter.
func NumberToDelimited(v any, opts ...Option) string { return NumberWithDelimiter(v, opts...) }

// NumberWithPrecision rounds to a fixed number of decimal places (or significant
// figures) and delimits the result — ActionView's number_with_precision /
// number_to_rounded. Defaults: precision 3, delimiter "", separator ".".
func NumberWithPrecision(v any, opts ...Option) string {
	o := apply(numberOpts{precision: 3, separator: ".", delimiter: ""}, opts)
	f, ok := toFloat(v)
	if !ok {
		return toS(v)
	}
	return numberToRounded(f, o)
}

// NumberToRounded is the modern name for NumberWithPrecision.
func NumberToRounded(v any, opts ...Option) string { return NumberWithPrecision(v, opts...) }

// NumberToCurrency formats a number as currency, mirroring ActionView's
// number_to_currency: precision 2, unit "$", format "%u%n", negative_format
// "-%u%n". A custom Format also updates the negative template unless
// NegativeFormat is given.
func NumberToCurrency(v any, opts ...Option) string {
	o := numberOpts{
		precision: 2, significant: false, separator: ".", delimiter: ",",
		unit: "$", format: "%u%n", negativeFormat: "-%u%n", stripZeros: false,
	}
	// A user-supplied :format overrides the negative default (ActiveSupport).
	for _, f := range opts {
		before := o.format
		f(&o)
		if o.format != before && o.negativeFormat == "-%u%n" {
			o.negativeFormat = "-" + o.format
		}
	}
	f, ok := toFloat(v)
	if !ok {
		s := strings.TrimSpace(toS(v))
		format := o.format
		if strings.HasPrefix(s, "-") {
			s = s[1:]
			format = o.negativeFormat
		}
		return strings.NewReplacer("%n", s, "%u", o.unit).Replace(format)
	}
	format := o.format
	if f < 0 {
		f = -f
		if f*math.Pow(10, float64(o.precision)) >= 0.5 {
			format = o.negativeFormat
		}
	}
	numStr := numberToRounded(f, o)
	return strings.NewReplacer("%n", numStr, "%u", o.unit).Replace(format)
}

// NumberToPercentage formats a number as a percentage, mirroring ActionView's
// number_to_percentage: precision 3, no delimiter, format "%n%".
func NumberToPercentage(v any, opts ...Option) string {
	o := apply(numberOpts{precision: 3, separator: ".", delimiter: "", format: "%n%"}, opts)
	f, ok := toFloat(v)
	if !ok {
		return toS(v)
	}
	return strings.ReplaceAll(o.format, "%n", numberToRounded(f, o))
}

// storageUnits are the NumberToHumanSizeConverter base-1024 unit labels.
var storageUnits = []string{"Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB"}

// NumberToHumanSize formats a byte count with a binary (1024) unit suffix,
// mirroring ActionView's number_to_human_size: 3 significant figures, trailing
// zeros stripped. Values below 1024 render as an integer count of Bytes/Byte.
func NumberToHumanSize(v any, opts ...Option) string {
	o := apply(numberOpts{precision: 3, significant: true, separator: ".", delimiter: "", stripZeros: true}, opts)
	f, ok := toFloat(v)
	if !ok {
		return toS(v)
	}
	base := 1024.0
	if math.Abs(math.Trunc(f)) < base {
		n := int64(f)
		unit := "Bytes"
		if n == 1 || n == -1 {
			unit = "Byte"
		}
		return toS(int(n)) + " " + unit
	}
	exp := int(math.Log(math.Abs(f)) / math.Log(base))
	if exp > len(storageUnits)-1 {
		exp = len(storageUnits) - 1
	}
	size := f / math.Pow(base, float64(exp))
	return numberToRounded(size, o) + " " + storageUnits[exp]
}

// decimalUnitExponents and decimalUnitNames give the base-10 human units in the
// order NumberToHumanConverter selects them (largest exponent first).
var decimalUnitExponents = []int{15, 12, 9, 6, 3, 0}
var decimalUnitNames = map[int]string{
	0: "", 3: "Thousand", 6: "Million", 9: "Billion", 12: "Trillion", 15: "Quadrillion",
}

// NumberToHuman formats a number with a decimal-magnitude word suffix
// (Thousand/Million/...), mirroring ActionView's number_to_human: the value is
// first rounded to 3 significant figures, then scaled to its unit and rounded
// again, with trailing zeros stripped.
func NumberToHuman(v any, opts ...Option) string {
	o := apply(numberOpts{precision: 3, significant: true, separator: ".", delimiter: "", stripZeros: true}, opts)
	f, ok := toFloat(v)
	if !ok {
		return toS(v)
	}
	// Initial significant-figure round on the raw magnitude.
	pre := roundHalfUp(ratFromFloat(f), o.precision-digitCount(ratFromFloat(f)))
	pf, _ := pre.Float64()
	exp := 0
	if pf != 0 {
		exp = int(math.Floor(math.Log10(math.Abs(pf))))
	}
	unitExp := 0
	for _, e := range decimalUnitExponents {
		if exp >= e {
			unitExp = e
			break
		}
	}
	scaled := pf / math.Pow(10, float64(unitExp))
	numStr := numberToRounded(scaled, o)
	unit := decimalUnitNames[unitExp]
	out := strings.ReplaceAll(strings.ReplaceAll("%n %u", "%n", numStr), "%u", unit)
	return strings.TrimSpace(out)
}
