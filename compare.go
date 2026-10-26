package decimal

// Sign reports -1, 0 or +1 as d is negative, zero or positive.
func (d Decimal) Sign() int {
	switch {
	case d.coef.isZero():
		return 0
	case d.neg:
		return -1
	default:
		return 1
	}
}

// IsZero reports whether d is zero. Zero has one
// representation, so it is the same as d == Zero.
func (d Decimal) IsZero() bool {
	return d.coef.isZero()
}

// IsPositive reports whether d > 0, so false for zero.
func (d Decimal) IsPositive() bool {
	return !d.coef.isZero() && !d.neg
}

// IsNegative reports whether d < 0. Zero is never negative.
func (d Decimal) IsNegative() bool {
	return d.neg
}

// IsInteger reports whether d has no fractional part.
func (d Decimal) IsInteger() bool {
	return d.prec == 0
}

// Equal reports whether d and o hold the same value.
// Canonical representation makes it plain ==.
func (d Decimal) Equal(o Decimal) bool {
	return d == o
}

// Cmp reports -1, 0 or +1 as d sorts before, with, or after o.
func (d Decimal) Cmp(o Decimal) int {
	if d == o {
		return 0
	}
	ds, os := d.Sign(), o.Sign()
	switch {
	case ds < os:
		return -1
	case ds > os:
		return 1
	case ds < 0:
		return -cmpAbs(d, o)
	default:
		return cmpAbs(d, o)
	}
}

// GreaterThan reports whether d is greater than o.
func (d Decimal) GreaterThan(o Decimal) bool {
	return d.Cmp(o) > 0
}

// GreaterThanOrEqual reports whether d is greater than or equal to o.
func (d Decimal) GreaterThanOrEqual(o Decimal) bool {
	return d.Cmp(o) >= 0
}

// LessThan reports whether d is less than o.
func (d Decimal) LessThan(o Decimal) bool {
	return d.Cmp(o) < 0
}

// LessThanOrEqual reports whether d is less than or equal to o.
func (d Decimal) LessThanOrEqual(o Decimal) bool {
	return d.Cmp(o) <= 0
}

// cmpAbs compares magnitudes, lifting the coarser
// value to the finer precision in 256 bits.
func cmpAbs(a, b Decimal) int {
	if a.prec == b.prec {
		return a.coef.cmp(b.coef)
	}
	prec := max(a.prec, b.prec)
	x := u256From(a.coef).mulPow10(int32(prec - a.prec))
	y := u256From(b.coef).mulPow10(int32(prec - b.prec))

	return x.cmp(y)
}
