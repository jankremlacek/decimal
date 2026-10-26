package decimal

import "testing"

func Test_u128_divMod(t *testing.T) {
	tests := []struct {
		name    string
		x       u128
		y       uint64
		wantQuo u128
		wantRem uint64
	}{
		{
			name:    "single word",
			x:       u128{lo: 123},
			y:       10,
			wantQuo: u128{lo: 12},
			wantRem: 3,
		},
		// The high word alone exceeds the divisor, taking the wide branch.
		{
			name:    "high word beyond the divisor",
			x:       u128{hi: 20, lo: 5},
			y:       10,
			wantQuo: u128{hi: 2},
			wantRem: 5,
		},
		{
			name:    "high word below the divisor",
			x:       u128{hi: 3, lo: 0},
			y:       1e19,
			wantQuo: u128{lo: 5},
			wantRem: 5340232221128654848,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quo, rem := tt.x.divMod(tt.y)
			if quo != tt.wantQuo || rem != tt.wantRem {
				t.Errorf("u128.divMod(%v, %d) = %v r%d, want %v r%d", tt.x, tt.y, quo, rem, tt.wantQuo, tt.wantRem)
			}
		})
	}
}

func Test_u128_mulAdd(t *testing.T) {
	tests := []struct {
		name      string
		x         u128
		m         uint64
		a         uint64
		want      u128
		wantSpill bool
	}{
		{
			name: "digit append",
			x:    u128{lo: 12},
			m:    10,
			a:    3,
			want: u128{lo: 123},
		},
		{
			name: "carry into the high word",
			x:    u128{lo: 1 << 63},
			m:    4,
			a:    1,
			want: u128{hi: 2, lo: 1},
		},
		{
			name:      "spill past 128 bits",
			x:         u128{hi: ^uint64(0), lo: ^uint64(0)},
			m:         10,
			a:         0,
			wantSpill: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, spill := tt.x.mulAdd(tt.m, tt.a)
			if spill != tt.wantSpill {
				t.Fatalf("u128.mulAdd(%v, %d, %d) spill = %t, want %t", tt.x, tt.m, tt.a, spill, tt.wantSpill)
			}
			if !spill && got != tt.want {
				t.Errorf("u128.mulAdd(%v, %d, %d) = %v, want %v", tt.x, tt.m, tt.a, got, tt.want)
			}
		})
	}
}

func Test_u128_scale(t *testing.T) {
	t.Run("multi-chunk scale", func(t *testing.T) {
		// Past 10^19, scale takes a second step.
		scaled, ok := u128{lo: 1}.scale(25)
		if !ok {
			t.Fatalf("u128.scale(1, 25) overflowed")
		}
		if got := string(scaled.appendDigits(nil)); got != "10000000000000000000000000" {
			t.Errorf("u128.scale(1, 25) = %s, want 10^25", got)
		}
	})
	t.Run("overflow", func(t *testing.T) {
		if _, ok := (u128{lo: 1}).scale(39); ok {
			t.Errorf("u128.scale(1, 39) = ok, want overflow")
		}
	})
}

func Test_u128_cmp(t *testing.T) {
	tests := []struct {
		name string
		x    u128
		y    u128
		want int
	}{
		{
			name: "equal",
			x:    u128{hi: 1, lo: 2},
			y:    u128{hi: 1, lo: 2},
			want: 0,
		},
		{
			name: "high word decides less",
			x:    u128{hi: 1},
			y:    u128{hi: 2},
			want: -1,
		},
		{
			name: "high word decides greater",
			x:    u128{hi: 3},
			y:    u128{hi: 2},
			want: 1,
		},
		{
			name: "low word decides less",
			x:    u128{hi: 1, lo: 1},
			y:    u128{hi: 1, lo: 2},
			want: -1,
		},
		{
			name: "low word decides greater",
			x:    u128{hi: 1, lo: 3},
			y:    u128{hi: 1, lo: 2},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.cmp(tt.y); got != tt.want {
				t.Errorf("u128.cmp(%v, %v) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_u128_inc(t *testing.T) {
	if got := (u128{lo: 41}).inc(); got != (u128{lo: 42}) {
		t.Errorf("u128.inc(41) = %v, want 42", got)
	}
	if got := (u128{lo: ^uint64(0)}).inc(); got != (u128{hi: 1}) {
		t.Errorf("u128.inc(2^64-1) = %v, want carry into the high word", got)
	}
}

func Test_u128_appendDigits(t *testing.T) {
	tests := []struct {
		name string
		x    u128
		want string
	}{
		{
			name: "zero",
			x:    u128{},
			want: "0",
		},
		{
			name: "single word",
			x:    u128{lo: 64500},
			want: "64500",
		},
		{
			name: "wide with zero-padded low chunk",
			x:    u128{hi: 5, lo: 0},
			want: "92233720368547758080",
		},
		{
			name: "widest",
			x:    u128{hi: ^uint64(0), lo: ^uint64(0)},
			want: "340282366920938463463374607431768211455",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.x.appendDigits(nil)); got != tt.want {
				t.Errorf("u128.appendDigits(%v) = %s, want %s", tt.x, got, tt.want)
			}
		})
	}
}

func Test_u128_digits(t *testing.T) {
	tests := []struct {
		name string
		x    u128
		want int
	}{
		{
			name: "zero",
			x:    u128{},
			want: 1,
		},
		{
			name: "single digit",
			x:    u128{lo: 9},
			want: 1,
		},
		{
			name: "first two-digit value",
			x:    u128{lo: 10},
			want: 2,
		},
		{
			name: "below a power of ten",
			x:    u128{lo: 999},
			want: 3,
		},
		{
			name: "at a power of ten",
			x:    u128{lo: 1000},
			want: 4,
		},
		{
			name: "widest single word",
			x:    u128{lo: ^uint64(0)},
			want: 20,
		},
		{
			name: "wide with zero-padded low chunk",
			x:    u128{hi: 5, lo: 0},
			want: 20,
		},
		{
			name: "widest",
			x:    u128{hi: ^uint64(0), lo: ^uint64(0)},
			want: 39,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.x.digits()

			if got != tt.want {
				t.Errorf("u128.digits(%v) = %d, want %d", tt.x, got, tt.want)
			}
			if written := len(tt.x.appendDigits(nil)); got != written {
				t.Errorf("u128.digits(%v) = %d, appendDigits wrote %d", tt.x, got, written)
			}
		})
	}
}

func Test_u128_bigEndian(t *testing.T) {
	t.Run("round-trips minimally", func(t *testing.T) {
		for _, x := range []u128{{}, {lo: 1}, {lo: 256}, {hi: 1}, {hi: ^uint64(0), lo: ^uint64(0)}} {
			b := x.appendBigEndian(nil)
			back, ok := u128FromBigEndian(b)
			if !ok || back != x {
				t.Errorf("u128 big-endian round-trip of %v = %v (%d bytes)", x, back, len(b))
			}
		}
	})
	t.Run("zero writes nothing", func(t *testing.T) {
		if b := (u128{}).appendBigEndian(nil); len(b) != 0 {
			t.Errorf("u128.appendBigEndian(0) = % x, want empty", b)
		}
	})
	t.Run("rejects more than sixteen bytes", func(t *testing.T) {
		if _, ok := u128FromBigEndian(make([]byte, 17)); ok {
			t.Errorf("u128FromBigEndian(17 bytes) = ok, want rejection")
		}
	})
}
