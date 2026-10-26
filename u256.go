package decimal

import "math/bits"

// u256 is the widened intermediate of arithmetic: aligning
// to a finer precision, a 128×128 product, and a scaled
// dividend all fit here, so an operation computes its exact
// result first and only then must fit a [Decimal] again.
//
// The multiplying methods do not report spill: every
// caller stays provably inside 256 bits, the widest being
// a 128-bit coefficient times 10^35 in division, under 2^245.
type u256 struct {
	// w holds the value in 64-bit limbs, least significant first.
	w [4]uint64
}

func u256From(x u128) u256 {
	return u256{w: [4]uint64{x.lo, x.hi}}
}

// mul128 reports the full 256-bit product of two coefficients.
func mul128(x, y u128) u256 {
	h0, l0 := bits.Mul64(x.lo, y.lo)
	h1, l1 := bits.Mul64(x.lo, y.hi)
	h2, l2 := bits.Mul64(x.hi, y.lo)
	h3, l3 := bits.Mul64(x.hi, y.hi)

	w1, c1 := bits.Add64(h0, l1, 0)
	w1, c2 := bits.Add64(w1, l2, 0)

	// c1+c2 reaches two, which Add64's carry input must
	// never carry — it joins as a plain addend instead.
	w2, c3 := bits.Add64(h1, h2, 0)
	w2, c4 := bits.Add64(w2, l3, 0)
	w2, c5 := bits.Add64(w2, c1+c2, 0)

	return u256{w: [4]uint64{l0, w1, w2, h3 + c3 + c4 + c5}}
}

func (x u256) isZero() bool {
	return x.w[0]|x.w[1]|x.w[2]|x.w[3] == 0
}

// fitsU128 reports whether x folds back into a coefficient.
func (x u256) fitsU128() bool {
	return x.w[2]|x.w[3] == 0
}

// u128 reports the low 128 bits; the caller has checked fitsU128.
func (x u256) u128() u128 {
	return u128{hi: x.w[1], lo: x.w[0]}
}

// cmp reports -1, 0 or +1 as x sorts before, with, or after y.
func (x u256) cmp(y u256) int {
	for i := len(x.w) - 1; i >= 0; i-- {
		switch {
		case x.w[i] < y.w[i]:
			return -1
		case x.w[i] > y.w[i]:
			return 1
		}
	}

	return 0
}

func (x u256) add(y u256) u256 {
	var (
		r u256
		c uint64
	)
	for i := range x.w {
		r.w[i], c = bits.Add64(x.w[i], y.w[i], c)
	}

	return r
}

// sub reports x - y; the caller has established x >= y.
func (x u256) sub(y u256) u256 {
	var (
		r u256
		b uint64
	)
	for i := range x.w {
		r.w[i], b = bits.Sub64(x.w[i], y.w[i], b)
	}

	return r
}

func (x u256) inc() u256 {
	return x.add(u256{w: [4]uint64{1}})
}

// mulPow10 reports x * 10^n, stepping through uint64-sized scale factors.
func (x u256) mulPow10(n int32) u256 {
	for n > 0 {
		step := min(n, MaxFractionDigits)
		x = x.mul64(pow10[step])
		n -= step
	}

	return x
}

// divMod64 reports the quotient and remainder of x by y.
func (x u256) divMod64(y uint64) (u256, uint64) {
	var (
		q   u256
		rem uint64
	)
	for i := len(x.w) - 1; i >= 0; i-- {
		q.w[i], rem = bits.Div64(rem, x.w[i], y)
	}

	return q, rem
}

// divMod reports the quotient and remainder of x by y, which is
// nonzero. A divisor beyond 64 bits — a rarity meaning a twenty-plus
// digit coefficient — falls back to binary long division.
func (x u256) divMod(y u256) (u256, u256) {
	if y.w[1]|y.w[2]|y.w[3] == 0 {
		q, rem := x.divMod64(y.w[0])
		return q, u256{w: [4]uint64{rem}}
	}

	var q, r u256
	for i := x.highestBit(); i >= 0; i-- {
		r = r.shl1()
		r.w[0] |= x.w[i/64] >> (i % 64) & 1
		if r.cmp(y) >= 0 {
			r = r.sub(y)
			q.w[i/64] |= 1 << (i % 64)
		}
	}

	return q, r
}

func (x u256) shl1() u256 {
	return u256{w: [4]uint64{
		x.w[0] << 1,
		x.w[1]<<1 | x.w[0]>>63,
		x.w[2]<<1 | x.w[1]>>63,
		x.w[3]<<1 | x.w[2]>>63,
	}}
}

// internal methods - not to be used by consumers

// highestBit reports the index of the top set bit, or -1 for zero.
func (x u256) highestBit() int {
	for i := len(x.w) - 1; i >= 0; i-- {
		if x.w[i] != 0 {
			return 64*i + 63 - bits.LeadingZeros64(x.w[i])
		}
	}

	return -1
}

// mul64 reports x * m; [u256] says why it reports no spill.
func (x u256) mul64(m uint64) u256 {
	var (
		r     u256
		carry uint64
	)
	for i := range x.w {
		hi, lo := bits.Mul64(x.w[i], m)
		lo, c := bits.Add64(lo, carry, 0)
		r.w[i] = lo
		carry = hi + c
	}

	return r
}
