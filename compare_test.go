package decimal

import "testing"

func TestDecimal_Sign(t *testing.T) {
	tests := []struct {
		name           string
		v              string
		wantSign       int
		wantIsZero     bool
		wantIsPositive bool
		wantIsNegative bool
	}{
		{
			name:           "positive",
			v:              "1.5",
			wantSign:       1,
			wantIsPositive: true,
		},
		{
			name:           "negative",
			v:              "-1.5",
			wantSign:       -1,
			wantIsNegative: true,
		},
		{
			name:       "zero",
			v:          "0",
			wantSign:   0,
			wantIsZero: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MustFromString(tt.v)
			if got := d.Sign(); got != tt.wantSign {
				t.Errorf("Decimal.Sign(%s) = %d, want %d", tt.v, got, tt.wantSign)
			}
			if got := d.IsZero(); got != tt.wantIsZero {
				t.Errorf("Decimal.IsZero(%s) = %t, want %t", tt.v, got, tt.wantIsZero)
			}
			if got := d.IsPositive(); got != tt.wantIsPositive {
				t.Errorf("Decimal.IsPositive(%s) = %t, want %t", tt.v, got, tt.wantIsPositive)
			}
			if got := d.IsNegative(); got != tt.wantIsNegative {
				t.Errorf("Decimal.IsNegative(%s) = %t, want %t", tt.v, got, tt.wantIsNegative)
			}
		})
	}
}

func TestDecimal_IsInteger(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want bool
	}{
		{
			name: "integer",
			v:    "1234",
			want: true,
		},
		{
			name: "zero",
			v:    "0",
			want: true,
		},
		{
			name: "negative integer",
			v:    "-1234",
			want: true,
		},
		{
			name: "trailing zeros shed to an integer",
			v:    "1234.000",
			want: true,
		},
		{
			name: "integer with trailing zeros",
			v:    "1200",
			want: true,
		},
		{
			name: "fraction",
			v:    "1234.5",
			want: false,
		},
		{
			name: "negative fraction",
			v:    "-0.5",
			want: false,
		},
		{
			name: "widest fraction",
			v:    "0.0000000000000000001",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).IsInteger(); got != tt.want {
				t.Errorf("Decimal.IsInteger(%s) = %t, want %t", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_Equal(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "same text",
			a:    "1.5",
			b:    "1.5",
			want: true,
		},
		{
			name: "trailing zeros normalize equal",
			a:    "1.5",
			b:    "1.500",
			want: true,
		},
		{
			name: "zero forms",
			a:    "0",
			b:    "-0.000",
			want: true,
		},
		{
			name: "close but distinct",
			a:    "1.5",
			b:    "1.51",
			want: false,
		},
		{
			name: "sign differs",
			a:    "1.5",
			b:    "-1.5",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Equal(MustFromString(tt.b)); got != tt.want {
				t.Errorf("Decimal.Equal(%s, %s) = %t, want %t", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDecimal_Cmp(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "equal",
			a:    "1.5",
			b:    "1.500",
			want: 0,
		},
		{
			name: "zero against zero",
			a:    "0",
			b:    "-0",
			want: 0,
		},
		{
			name: "positive order",
			a:    "1.05",
			b:    "1.5",
			want: -1,
		},
		{
			name: "same precision order",
			a:    "64500.4",
			b:    "64500.3",
			want: 1,
		},
		{
			name: "negative beats nothing",
			a:    "-0.1",
			b:    "0",
			want: -1,
		},
		{
			name: "positive beats negative",
			a:    "0.1",
			b:    "-100",
			want: 1,
		},
		{
			name: "negative order inverts magnitude",
			a:    "-2.5",
			b:    "-2.4",
			want: -1,
		},
		{
			name: "wei apart",
			a:    "1.0000000000000000001",
			b:    "1",
			want: 1,
		},
		{
			name: "lift past 128 bits decides",
			a:    "34028236692093846346.3374607431768211455",
			b:    "34028236692093846347",
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Cmp(MustFromString(tt.b)); got != tt.want {
				t.Errorf("Decimal.Cmp(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDecimal_Comparisons(t *testing.T) {
	tests := []struct {
		name    string
		a       string
		b       string
		wantGT  bool
		wantGTE bool
		wantLT  bool
		wantLTE bool
	}{
		{
			name:    "less",
			a:       "1",
			b:       "2",
			wantGT:  false,
			wantGTE: false,
			wantLT:  true,
			wantLTE: true,
		},
		{
			name:    "equal",
			a:       "2",
			b:       "2.0",
			wantGT:  false,
			wantGTE: true,
			wantLT:  false,
			wantLTE: true,
		},
		{
			name:    "greater",
			a:       "3",
			b:       "2",
			wantGT:  true,
			wantGTE: true,
			wantLT:  false,
			wantLTE: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := MustFromString(tt.a), MustFromString(tt.b)
			if got := a.GreaterThan(b); got != tt.wantGT {
				t.Errorf("Decimal.GreaterThan(%s, %s) = %t, want %t", tt.a, tt.b, got, tt.wantGT)
			}
			if got := a.GreaterThanOrEqual(b); got != tt.wantGTE {
				t.Errorf("Decimal.GreaterThanOrEqual(%s, %s) = %t, want %t", tt.a, tt.b, got, tt.wantGTE)
			}
			if got := a.LessThan(b); got != tt.wantLT {
				t.Errorf("Decimal.LessThan(%s, %s) = %t, want %t", tt.a, tt.b, got, tt.wantLT)
			}
			if got := a.LessThanOrEqual(b); got != tt.wantLTE {
				t.Errorf("Decimal.LessThanOrEqual(%s, %s) = %t, want %t", tt.a, tt.b, got, tt.wantLTE)
			}
		})
	}
}
