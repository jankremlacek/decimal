package decimal

import "fmt"

// divisionPrecision is the fraction width a quotient rounds to, matching shopspring's DivisionPrecision.
const divisionPrecision = 16

// Add returns d + o exactly. It panics with [ErrRange] when the sum does not fit.
func (d Decimal) Add(o Decimal) Decimal {
	r, err := add(d, o)
	if err != nil {
		panic(fmt.Errorf("decimal.Add(%s, %s): %w", d, o, err))
	}

	return r
}

// Sub returns d - o exactly. It panics with [ErrRange] when the difference does not fit.
func (d Decimal) Sub(o Decimal) Decimal {
	r, err := add(d, o.Neg())
	if err != nil {
		panic(fmt.Errorf("decimal.Sub(%s, %s): %w", d, o, err))
	}

	return r
}

// add is the exact core of Add and Sub: align to the finer
// precision in 256 bits — the lift can leave 128 while
// the result still fits — then combine magnitudes by sign.
func add(a, b Decimal) (Decimal, error) {
	prec := max(a.prec, b.prec)
	x := u256From(a.coef).mulPow10(int32(prec - a.prec))
	y := u256From(b.coef).mulPow10(int32(prec - b.prec))
	if a.neg == b.neg {
		return buildWide(a.neg, x.add(y), int32(prec))
	}

	switch x.cmp(y) {
	case 1:
		return buildWide(a.neg, x.sub(y), int32(prec))
	case -1:
		return buildWide(b.neg, y.sub(x), int32(prec))
	default:
		return Decimal{}, nil
	}
}

// Mul returns d * o exactly. It panics with [ErrRange] when the product does not
// fit — including a product whose nonzero fraction needs more than nineteen digits.
func (d Decimal) Mul(o Decimal) Decimal {
	r, err := buildWide(d.neg != o.neg, mul128(d.coef, o.coef), int32(d.prec)+int32(o.prec))
	if err != nil {
		panic(fmt.Errorf("decimal.Mul(%s, %s): %w", d, o, err))
	}

	return r
}

// Div returns d / o rounded half away from zero at sixteen fractional
// digits, exactly as shopspring rounds. It panics with [ErrDivByZero]
// on a zero divisor, and with [ErrRange] when the quotient does not fit.
func (d Decimal) Div(o Decimal) Decimal {
	if o.coef.isZero() {
		panic(fmt.Errorf("decimal.Div(%s, 0): %w", d, ErrDivByZero))
	}

	// The quotient at divisionPrecision is num/den with the fraction
	// widths folded into whichever side keeps both integral.
	shift := divisionPrecision + int32(o.prec) - int32(d.prec)
	num, den := u256From(d.coef), u256From(o.coef)
	if shift >= 0 {
		num = num.mulPow10(shift)
	} else {
		den = den.mulPow10(-shift)
	}
	q, rem := num.divMod(den)
	if rem.shl1().cmp(den) >= 0 {
		q = q.inc()
	}
	r, err := buildWide(d.neg != o.neg, q, divisionPrecision)
	if err != nil {
		panic(fmt.Errorf("decimal.Div(%s, %s): %w", d, o, err))
	}

	return r
}

// Neg returns -d. Zero stays zero, never negative.
func (d Decimal) Neg() Decimal {
	if d.coef.isZero() {
		return d
	}
	d.neg = !d.neg

	return d
}

// Abs returns |d|.
func (d Decimal) Abs() Decimal {
	d.neg = false

	return d
}

// Min returns the least of its arguments. first
// is separate so a call cannot pass none.
func Min(first Decimal, rest ...Decimal) Decimal {
	m := first
	for _, v := range rest {
		if v.Cmp(m) < 0 {
			m = v
		}
	}

	return m
}

// Max returns the greatest of its arguments.
// first is separate so a call cannot pass none.
func Max(first Decimal, rest ...Decimal) Decimal {
	m := first
	for _, v := range rest {
		if v.Cmp(m) > 0 {
			m = v
		}
	}

	return m
}

// Sum returns the arguments added exactly, with [Decimal.Add]'s panic on a sum that does not fit.
func Sum(first Decimal, rest ...Decimal) Decimal {
	s := first
	for _, v := range rest {
		s = s.Add(v)
	}

	return s
}

// buildWide is [build] over a widened intermediate: shed trailing
// zeros, then the result must fold back into a coefficient.
func buildWide(neg bool, x u256, prec int32) (Decimal, error) {
	if x.isZero() {
		return Decimal{}, nil
	}
	for prec > 0 {
		quo, rem := x.divMod64(10)
		if rem != 0 {
			break
		}
		x, prec = quo, prec-1
	}
	if prec > MaxFractionDigits || !x.fitsU128() {
		return Decimal{}, ErrRange
	}

	return Decimal{coef: x.u128(), prec: uint8(prec), neg: neg}, nil
}
