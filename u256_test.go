package decimal

import "testing"

func Test_mul128(t *testing.T) {
	tests := []struct {
		name string
		x    u128
		y    u128
		want u256
	}{
		{
			name: "single word",
			x:    u128{lo: 6},
			y:    u128{lo: 7},
			want: u256{w: [4]uint64{42}},
		},
		// Each partial product lands in a different limb.
		{
			name: "cross-limb carries",
			x:    u128{hi: 1, lo: 2},
			y:    u128{hi: 1, lo: 3},
			want: u256{w: [4]uint64{6, 5, 1, 0}},
		},
		// The widest coefficient squared reaches the top limb.
		{
			name: "widest square",
			x:    u128{hi: ^uint64(0), lo: ^uint64(0)},
			y:    u128{hi: ^uint64(0), lo: ^uint64(0)},
			want: u256{w: [4]uint64{1, 0, ^uint64(1), ^uint64(0)}},
		},
		// Both middle-column adds carry at once: the double carry the fuzzer
		// caught being folded into a single Add64 carry bit.
		{
			name: "double middle carry",
			x:    u128{hi: 1, lo: ^uint64(0)},
			y:    u128{hi: 1, lo: ^uint64(0)},
			want: u256{w: [4]uint64{1, ^uint64(3), 3, 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mul128(tt.x, tt.y); got != tt.want {
				t.Errorf("mul128(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_u256_divMod(t *testing.T) {
	tests := []struct {
		name    string
		x       u256
		y       u256
		wantQuo u256
		wantRem u256
	}{
		{
			name:    "word divisor",
			x:       u256{w: [4]uint64{123}},
			y:       u256{w: [4]uint64{10}},
			wantQuo: u256{w: [4]uint64{12}},
			wantRem: u256{w: [4]uint64{3}},
		},
		// A divisor of 2^64 is past 64 bits and takes the binary path.
		{
			name:    "wide divisor shifts limbs",
			x:       u256{w: [4]uint64{5, 7, 9, 0}},
			y:       u256{w: [4]uint64{0, 1}},
			wantQuo: u256{w: [4]uint64{7, 9}},
			wantRem: u256{w: [4]uint64{5}},
		},
		// The binary path leaving a remainder.
		{
			name:    "wide divisor with remainder",
			x:       u256{w: [4]uint64{5, 3}},
			y:       u256{w: [4]uint64{1, 1}},
			wantQuo: u256{w: [4]uint64{3}},
			wantRem: u256{w: [4]uint64{2}},
		},
		{
			name:    "zero dividend",
			x:       u256{},
			y:       u256{w: [4]uint64{0, 1}},
			wantQuo: u256{},
			wantRem: u256{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quo, rem := tt.x.divMod(tt.y)
			if quo != tt.wantQuo || rem != tt.wantRem {
				t.Errorf("u256.divMod(%v, %v) = %v r%v, want %v r%v", tt.x, tt.y, quo, rem, tt.wantQuo, tt.wantRem)
			}
		})
	}
}

func Test_u256_divMod64(t *testing.T) {
	// A dividend past 128 bits, out of reach of u128.divMod.
	x := u256{w: [4]uint64{0, 0, 1, 0}}
	quo, rem := x.divMod64(10)
	if got := quo.mul64(10).add(u256{w: [4]uint64{rem}}); got != x {
		t.Errorf("u256.divMod64(2^128, 10) does not recompose: %v*10 + %d = %v", quo, rem, got)
	}
	if rem != 6 {
		t.Errorf("u256.divMod64(2^128, 10) rem = %d, want 6", rem)
	}
}

func Test_u256_add_sub(t *testing.T) {
	t.Run("carry ripples across limbs", func(t *testing.T) {
		got := u256{w: [4]uint64{^uint64(0), ^uint64(0), 0, 0}}.add(u256{w: [4]uint64{1}})
		if got != (u256{w: [4]uint64{0, 0, 1, 0}}) {
			t.Errorf("u256.add carry = %v, want 2^128", got)
		}
	})
	t.Run("borrow ripples across limbs", func(t *testing.T) {
		got := u256{w: [4]uint64{0, 0, 1, 0}}.sub(u256{w: [4]uint64{1}})
		if got != (u256{w: [4]uint64{^uint64(0), ^uint64(0), 0, 0}}) {
			t.Errorf("u256.sub borrow = %v, want 2^128-1", got)
		}
	})
	t.Run("inc carries", func(t *testing.T) {
		got := u256{w: [4]uint64{^uint64(0)}}.inc()
		if got != (u256{w: [4]uint64{0, 1}}) {
			t.Errorf("u256.inc(2^64-1) = %v, want 2^64", got)
		}
	})
}

func Test_u256_cmp(t *testing.T) {
	tests := []struct {
		name string
		x    u256
		y    u256
		want int
	}{
		{
			name: "equal",
			x:    u256{w: [4]uint64{1, 2, 3, 4}},
			y:    u256{w: [4]uint64{1, 2, 3, 4}},
			want: 0,
		},
		{
			name: "top limb decides less",
			x:    u256{w: [4]uint64{9, 9, 9, 1}},
			y:    u256{w: [4]uint64{0, 0, 0, 2}},
			want: -1,
		},
		{
			name: "low limb decides greater",
			x:    u256{w: [4]uint64{2}},
			y:    u256{w: [4]uint64{1}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.cmp(tt.y); got != tt.want {
				t.Errorf("u256.cmp(%v, %v) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_u256_mulPow10(t *testing.T) {
	t.Run("identity at zero", func(t *testing.T) {
		x := u256{w: [4]uint64{42}}
		if got := x.mulPow10(0); got != x {
			t.Errorf("u256.mulPow10(42, 0) = %v, want 42", got)
		}
	})
	t.Run("two chunks recompose", func(t *testing.T) {
		// Past 10^19, mulPow10 takes a second step.
		got := u256{w: [4]uint64{1}}.mulPow10(25)
		quo, rem := got.divMod64(pow10[19])
		if rem != 0 {
			t.Fatalf("u256.mulPow10(1, 25) rem mod 10^19 = %d, want 0", rem)
		}
		quo, rem = quo.divMod64(pow10[6])
		if rem != 0 || quo != (u256{w: [4]uint64{1}}) {
			t.Errorf("u256.mulPow10(1, 25) does not recompose: %v r%d", quo, rem)
		}
	})
}

func Test_u256_highestBit(t *testing.T) {
	tests := []struct {
		name string
		x    u256
		want int
	}{
		{
			name: "zero",
			x:    u256{},
			want: -1,
		},
		{
			name: "bit zero",
			x:    u256{w: [4]uint64{1}},
			want: 0,
		},
		{
			name: "top limb",
			x:    u256{w: [4]uint64{0, 0, 0, 1 << 5}},
			want: 197,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.highestBit(); got != tt.want {
				t.Errorf("u256.highestBit(%v) = %d, want %d", tt.x, got, tt.want)
			}
		})
	}
}

func Test_u256_shl1(t *testing.T) {
	got := u256{w: [4]uint64{1 << 63, 1 << 63, 1 << 63, 0}}.shl1()
	if got != (u256{w: [4]uint64{0, 1, 1, 1}}) {
		t.Errorf("u256.shl1 cross-limb = %v, want carries into each limb", got)
	}
}
