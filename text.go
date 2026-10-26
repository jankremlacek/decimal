package decimal

import "math"

// String renders the canonical text form: no exponent, no trailing fraction zeros.
func (d Decimal) String() string {
	var buf [MaxDigits + 3]byte

	return string(d.Append(buf[:0]))
}

// Append writes what [Decimal.String] renders onto b, allocation-free.
func (d Decimal) Append(b []byte) []byte {
	if d.coef.isZero() {
		return append(b, '0')
	}
	if d.neg {
		b = append(b, '-')
	}

	var buf [MaxDigits]byte
	digits := d.coef.appendDigits(buf[:0])
	if d.prec == 0 {
		return append(b, digits...)
	}

	// Split the digits at the point, standing in zeros where the coefficient is too short to reach it.
	prec := int(d.prec)
	if len(digits) > prec {
		b = append(b, digits[:len(digits)-prec]...)
		digits = digits[len(digits)-prec:]
	} else {
		b = append(b, '0')
	}
	b = append(b, '.')
	for range prec - len(digits) {
		b = append(b, '0')
	}

	return append(b, digits...)
}

// TextLen reports how many bytes [Decimal.Append] writes.
func (d Decimal) TextLen() int {
	if d.coef.isZero() {
		return 1
	}

	n := 0
	if d.neg {
		n = 1
	}
	digits := d.coef.digits()
	prec := int(d.prec)
	if prec == 0 {
		return n + digits
	}
	if digits > prec {
		n += digits - prec
	} else {
		n++
	}

	return n + 1 + prec
}

// StringFixed renders d rounded half away from zero to exactly
// places fraction digits, zero-padded. Negative places
// renders the rounded integer: 545 at places -1 is "550".
func (d Decimal) StringFixed(places int32) string {
	r := d.Round(places)
	if places <= 0 {
		return r.String()
	}

	b := make([]byte, 0, MaxDigits+3+int(places))
	if r.neg {
		b = append(b, '-')
	}
	var buf [MaxDigits]byte
	digits := r.coef.appendDigits(buf[:0])
	prec := int(r.prec)
	if len(digits) > prec {
		b = append(b, digits[:len(digits)-prec]...)
		digits = digits[len(digits)-prec:]
	} else {
		b = append(b, '0')
	}
	b = append(b, '.')
	for range prec - len(digits) {
		b = append(b, '0')
	}
	b = append(b, digits...)
	for range int(places) - prec {
		b = append(b, '0')
	}

	return string(b)
}

// MarshalText implements [encoding.TextMarshaler].
func (d Decimal) MarshalText() ([]byte, error) {
	return d.Append(make([]byte, 0, MaxDigits+3)), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler]. It accepts what
// [NewFromString] does and leaves the receiver untouched on error.
func (d *Decimal) UnmarshalText(text []byte) error {
	parsed, err := parse(text)
	if err != nil {
		return err
	}
	*d = parsed

	return nil
}

// parse reads decimal text: an optional sign, digits with at most
// one point among them, and an optional power-of-ten exponent.
// ".5" and "5." both spell a number; "." alone does not.
//
// Zeros fold into the coefficient only once a later nonzero digit proves
// them significant, so a value spelled with any number of leading or
// trailing zeros parses as long as the value itself fits — "7000.00…0"
// is 7000, not a spilled 39-digit coefficient. The coefficient this
// leaves carries no trailing zeros, which is the canonical form directly.
func parse(b []byte) (Decimal, error) {
	if len(b) == 0 {
		return Decimal{}, ErrSyntax
	}

	neg, i := readSign(b, 0)

	var (
		coef    u128
		frac    int64 // fraction digit positions consumed
		pending int64 // zero run after the last nonzero digit
		digits  bool
		seenDot bool
	)
mantissa:
	for ; i < len(b); i++ {
		switch c := b[i]; {
		case c >= '0' && c <= '9':
			digits = true
			if seenDot {
				frac++
			}
			if c == '0' {
				if !coef.isZero() {
					pending++
				}
				continue
			}
			// A nonzero digit makes the pending zeros significant.
			if pending > MaxDigits {
				return Decimal{}, ErrRange
			}
			flushed, ok := coef.scale(int32(pending))
			if !ok {
				return Decimal{}, ErrRange
			}
			next, spilled := flushed.mulAdd(10, uint64(c-'0'))
			if spilled {
				return Decimal{}, ErrRange
			}
			coef, pending = next, 0
		case c == '.' && !seenDot:
			seenDot = true
		default:
			break mantissa
		}
	}
	if !digits {
		return Decimal{}, ErrSyntax
	}

	exp := int64(0)
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		var err error
		if exp, i, err = parseExp(b, i+1); err != nil {
			return Decimal{}, err
		}
	}
	if i != len(b) {
		return Decimal{}, ErrSyntax
	}
	if coef.isZero() {
		return Decimal{}, nil
	}

	// The pending zeros and the exponent both shift the point; what remains
	// of the fraction width is the precision the value keeps.
	switch prec := frac - exp - pending; {
	case prec > MaxFractionDigits:
		return Decimal{}, ErrRange
	case prec >= 0:
		return Decimal{coef: coef, prec: uint8(prec), neg: neg}, nil
	case prec < -MaxDigits:
		return Decimal{}, ErrRange
	default:
		scaled, ok := coef.scale(int32(-prec))
		if !ok {
			return Decimal{}, ErrRange
		}

		return Decimal{coef: scaled, neg: neg}, nil
	}
}

// parseExp reads the exponent after an 'e', reporting it and the index
// past it. Its bounds mirror the int32 line shopspring draws: an exponent
// beyond int32 is rejected, anything within parses — range-checking against
// the value is the caller's concern. shopspring's tighter MaxDecodeExponent
// cap is deliberately not mirrored; see FuzzNewFromString.
func parseExp(b []byte, i int) (int64, int, error) {
	neg, i := readSign(b, i)

	exp, digits := int64(0), false
	for ; i < len(b); i++ {
		if b[i] < '0' || b[i] > '9' {
			break
		}
		if exp = exp*10 + int64(b[i]-'0'); exp > math.MaxInt32+1 {
			return 0, i, ErrRange
		}
		digits = true
	}
	if !digits {
		return 0, i, ErrSyntax
	}
	if neg {
		exp = -exp
	}
	if exp > math.MaxInt32 {
		return 0, i, ErrRange
	}

	return exp, i, nil
}

// minusSign is U+2212 MINUS SIGN, which shopspring reads as '-'.
const minusSign = "\u2212"

// readSign reads an optional sign at b[i], reporting whether it negates and
// the index past it. A minus is either an ASCII '-' or [minusSign].
func readSign(b []byte, i int) (bool, int) {
	switch {
	case i >= len(b):
		return false, i
	case b[i] == '-':
		return true, i + 1
	case b[i] == '+':
		return false, i + 1
	case string(b[i:min(i+len(minusSign), len(b))]) == minusSign:
		return true, i + len(minusSign)
	}

	return false, i
}
