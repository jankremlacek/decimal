package decimal

import "testing"

func TestDecimal_InexactFloat64(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want float64
	}{
		{
			name: "zero",
			v:    "0",
			want: 0,
		},
		{
			name: "tenth lands on the nearest float",
			v:    "0.1",
			want: 0.1,
		},
		{
			name: "negative",
			v:    "-2.5",
			want: -2.5,
		},
		{
			name: "price",
			v:    "64500.4",
			want: 64500.4,
		},
		{
			name: "wei",
			v:    "0.0000000000000000001",
			want: 1e-19,
		},
		{
			name: "widest coefficient",
			v:    "340282366920938463463374607431768211455",
			want: 3.402823669209385e38,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Exact equality on purpose: the conversion is correctly
			// rounded, so the nearest float64 is fully determined.
			if got := MustFromString(tt.v).InexactFloat64(); got != tt.want {
				t.Errorf("Decimal.InexactFloat64(%s) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
