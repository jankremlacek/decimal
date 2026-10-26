package decimal

import (
	"encoding/binary"
	"math/bits"
	"strconv"
)

// pow10 holds every power of ten a uint64 holds, the step size of scaling.
var pow10 = [MaxFractionDigits + 1]uint64{
	1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
}

// u128 is an unsigned 128-bit coefficient magnitude.
type u128 struct {
	hi uint64
	lo uint64
}

func (x u128) isZero() bool { return x.hi == 0 && x.lo == 0 }

// cmp reports -1, 0 or +1 as x sorts before, with, or after y.
func (x u128) cmp(y u128) int {
	switch {
	case x.hi != y.hi:
		if x.hi < y.hi {
			return -1
		}
		return 1
	case x.lo < y.lo:
		return -1
	case x.lo > y.lo:
		return 1
	default:
		return 0
	}
}

// inc reports x + 1; the caller keeps x below the 128-bit ceiling.
func (x u128) inc() u128 {
	x.lo++
	if x.lo == 0 {
		x.hi++
	}

	return x
}

// mulAdd reports x*m + a and whether the result left 128 bits.
func (x u128) mulAdd(m, a uint64) (u128, bool) {
	loHi, loLo := bits.Mul64(x.lo, m)
	hiHi, hiLo := bits.Mul64(x.hi, m)
	hi, spill := bits.Add64(loHi, hiLo, 0)
	lo, carry := bits.Add64(loLo, a, 0)
	hi, carried := bits.Add64(hi, carry, 0)

	return u128{hi: hi, lo: lo}, hiHi != 0 || spill != 0 || carried != 0
}

// divMod reports the quotient and remainder of x by y.
func (x u128) divMod(y uint64) (u128, uint64) {
	if x.hi < y {
		lo, rem := bits.Div64(x.hi, x.lo, y)
		return u128{lo: lo}, rem
	}
	hi, rem := bits.Div64(0, x.hi, y)
	lo, rem := bits.Div64(rem, x.lo, y)

	return u128{hi: hi, lo: lo}, rem
}

// scale reports x * 10^n and whether it fits 128 bits.
func (x u128) scale(n int32) (u128, bool) {
	// Zero never spills, so no n is worth stepping through.
	if x.isZero() {
		return x, true
	}
	for n > 0 {
		step := min(n, MaxFractionDigits)
		scaled, spilled := x.mulAdd(pow10[step], 0)
		if spilled {
			return u128{}, false
		}
		x, n = scaled, n-step
	}

	return x, true
}

// appendDigits writes x in base ten without leading zeros.
func (x u128) appendDigits(b []byte) []byte {
	if x.hi == 0 {
		return strconv.AppendUint(b, x.lo, 10)
	}
	// Peel the low nineteen digits, the widest chunk a uint64 holds.
	quo, rem := x.divMod(pow10[MaxFractionDigits])
	b = quo.appendDigits(b)
	var buf [20]byte
	digits := strconv.AppendUint(buf[:0], rem, 10)
	for range MaxFractionDigits - len(digits) {
		b = append(b, '0')
	}

	return append(b, digits...)
}

// digits reports how many base-ten digits [u128.appendDigits] writes.
func (x u128) digits() int {
	if x.hi != 0 {
		quo, _ := x.divMod(pow10[MaxFractionDigits])

		return quo.digits() + MaxFractionDigits
	}
	if x.lo < 10 {
		return 1
	}

	d := (bits.Len64(x.lo) + 1) * 1233 >> 12
	if x.lo < pow10[d] {
		return d
	}

	return d + 1
}

// appendBigEndian writes x as minimal big-endian bytes; zero writes nothing.
func (x u128) appendBigEndian(b []byte) []byte {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], x.hi)
	binary.BigEndian.PutUint64(buf[8:], x.lo)

	i := 0
	for i < len(buf) && buf[i] == 0 {
		i++
	}

	return append(b, buf[i:]...)
}

// u128FromBigEndian reads the magnitude encoding appendBigEndian writes.
func u128FromBigEndian(b []byte) (u128, bool) {
	if len(b) > 16 {
		return u128{}, false
	}
	var buf [16]byte
	copy(buf[16-len(b):], b)

	return u128{
		hi: binary.BigEndian.Uint64(buf[:8]),
		lo: binary.BigEndian.Uint64(buf[8:]),
	}, true
}
