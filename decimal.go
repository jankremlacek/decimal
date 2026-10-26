// Package decimal implements a fixed-point decimal for financial values.
//
// A [Decimal] is coefficient / 10^prec held by value: a 128-bit coefficient,
// at most nineteen fractional digits, no pointer, no allocation, nothing
// for the garbage collector to scan. Construction normalizes, so each
// value has exactly one representation and == is value equality.
//
// The representable range trades precision against magnitude,
// because the two bounds bind together: the coefficient is at most
// 2^128−1 = 340282366920938463463374607431768211455 and prec is at
// most [MaxFractionDigits]. Carrying no fraction a value reaches
// ±340282366920938463463374607431768211455; carrying the full nineteen-digit
// fraction it reaches ±34028236692093846346.3374607431768211455;
// and the smallest nonzero magnitude is ±0.0000000000000000001.
// A result that does not fit is an error or a panic, never
// a slower fallback; this package does not reach for math/big.
//
// Arithmetic is exact-or-panic: operations return the exact
// value or panic with [ErrRange]. Only parsing untrusted input
// ([NewFromString]) returns an error. Division alone rounds, matching
// shopspring/decimal: every quotient rounds half away from zero at
// sixteen fractional digits, so one that fits sixteen digits is exact.
//
// The text and JSON forms are canonical decimal text: no exponent, no trailing
// fraction zeros, JSON quoted. The binary and gob form is this package's own
// compact layout; callers that persist it own their versioning.
package decimal

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"strconv"
)

var (
	// ErrSyntax reports text that does not spell a decimal.
	ErrSyntax = errors.New("decimal: invalid syntax")

	// ErrRange reports a value outside the 128-bit
	// coefficient or the nineteen-digit fraction.
	ErrRange = errors.New("decimal: out of range")

	// ErrDivByZero reports a zero divisor; [Decimal.Div] panics with it.
	ErrDivByZero = errors.New("decimal: division by zero")
)

// MaxFractionDigits is the widest fraction a [Decimal] holds:
// nineteen digits, so every scale factor fits a uint64,
// and wei (10^-18) is covered with a digit to spare.
const MaxFractionDigits = 19

// MaxDigits is the longest coefficient a [Decimal] holds, in base-ten
// digits: 2^128−1 has 39. Not every 39-digit coefficient fits.
const MaxDigits = 39

// Zero is the zero value of [Decimal], named for comparisons such as d.Cmp(Zero).
var Zero Decimal

// Decimal is a signed fixed-point decimal:
// coefficient / 10^prec. The zero
// value is zero and ready to use.
//
// A Decimal is immutable: every method returns a new value and never mutates
// its receiver or arguments, so a value is safe to copy and share.
//
// Every value is normalized — the coefficient carries no trailing zeros and
// zero is never negative — so == compares values and Decimal keys maps.
type Decimal struct {
	coef u128
	prec uint8
	neg  bool
}

var (
	// Decimal must stay comparable for == and map keys; a slice, map or func field fails to compile here.
	_ map[Decimal]struct{}

	_ driver.Valuer = Decimal{}
	_ fmt.Stringer  = Decimal{}
	_ sql.Scanner   = (*Decimal)(nil)
)

// New returns v * 10^exp.
//
// It panics when the result does not fit, which suits literals and constants;
// parse input that may be out of range with [NewFromString].
func New(v int64, exp int32) Decimal {
	d, err := newFromParts(v < 0, magnitude(v), exp)
	if err != nil {
		panic(fmt.Errorf("decimal.New(%d, %d): %w", v, exp, err))
	}

	return d
}

// NewFromInt returns v. Every int64 fits, so it never panics.
func NewFromInt(v int64) Decimal {
	return Decimal{
		coef: u128{lo: magnitude(v)},
		neg:  v < 0,
	}
}

// NewFromFloat returns the decimal of the shortest text that round-trips
// to v, exactly as [strconv.FormatFloat] with precision -1 renders it.
//
// It panics on NaN, an infinity, or a value needing
// more than nineteen fractional digits.
func NewFromFloat(v float64) Decimal {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		panic(fmt.Errorf("decimal.NewFromFloat(%v): %w", v, ErrSyntax))
	}

	var buf [32]byte
	d, err := parse(strconv.AppendFloat(buf[:0], v, 'f', -1, 64))
	if err != nil {
		panic(fmt.Errorf("decimal.NewFromFloat(%v): %w", v, err))
	}

	return d
}

// NewFromString parses decimal text: an optional sign, digits with
// an optional fraction, and an optional power-of-ten exponent, like
// "-1.5" or "1.5e-8". Either minus may also be U+2212 MINUS SIGN, as
// in "−1.5" or "1.5e−8". The error wraps [ErrSyntax] or [ErrRange].
func NewFromString(v string) (Decimal, error) {
	d, err := parse([]byte(v))
	if err != nil {
		return Decimal{}, fmt.Errorf("decimal.NewFromString(%q): %w", v, err)
	}

	return d, nil
}

// MustFromString is [NewFromString] for text known
// to be well formed. It panics on anything else.
func MustFromString(v string) Decimal {
	d, err := NewFromString(v)
	if err != nil {
		panic(err)
	}

	return d
}

// Exponent reports the power of ten the coefficient sits over: the negated
// fraction width of the canonical form, so zero or negative.
func (d Decimal) Exponent() int32 { return -int32(d.prec) }

// newFromParts returns ±mag * 10^exp, or [ErrRange] when it does not fit.
func newFromParts(neg bool, mag uint64, exp int32) (Decimal, error) {
	coef := u128{lo: mag}
	if exp > 0 {
		scaled, ok := coef.scale(exp)
		if !ok {
			return Decimal{}, ErrRange
		}

		return build(neg, scaled, 0)
	}
	if exp >= -(MaxDigits + MaxFractionDigits) {
		return build(neg, coef, -exp)
	}

	// Deeper exponents overflow an int32 negation, and no coefficient
	// sheds that many zeros anyway — only zero survives down here.
	if coef.isZero() {
		return Decimal{}, nil
	}

	return Decimal{}, ErrRange
}

// build narrows a fraction of any width down to MaxFractionDigits — the shed
// must be exact, a nonzero digit falling off the end means
// the value does not fit — then normalizes through canonical.
func build(neg bool, coef u128, prec int32) (Decimal, error) {
	for prec > MaxFractionDigits {
		quo, rem := coef.divMod(10)
		if rem != 0 {
			return Decimal{}, ErrRange
		}
		coef, prec = quo, prec-1
	}

	return canonical(neg, coef, prec), nil
}

// canonical normalizes a coefficient with an in-range fraction
// width into the one canonical representation: trailing
// zeros shed, zero never negative. The width must already
// sit within MaxFractionDigits; build narrows wider ones.
func canonical(neg bool, coef u128, prec int32) Decimal {
	if coef.isZero() {
		return Decimal{}
	}
	for prec > 0 {
		quo, rem := coef.divMod(10)
		if rem != 0 {
			break
		}
		coef, prec = quo, prec-1
	}

	return Decimal{
		coef: coef,
		prec: uint8(prec),
		neg:  neg,
	}
}

// magnitude reports the absolute value of v, including at the int64 floor.
func magnitude(v int64) uint64 {
	if v < 0 {
		return -uint64(v)
	}
	return uint64(v)
}
