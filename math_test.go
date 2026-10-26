package decimal

import (
	"errors"
	"testing"
)

func TestDecimal_Add(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "integers",
			a:    "2",
			b:    "3",
			want: "5",
		},
		{
			name: "mixed precision",
			a:    "0.1",
			b:    "0.2",
			want: "0.3",
		},
		{
			name: "cancellation to zero",
			a:    "1.5",
			b:    "-1.5",
			want: "0",
		},
		{
			name: "negative plus positive",
			a:    "-1.5",
			b:    "0.25",
			want: "-1.25",
		},
		{
			name: "both negative",
			a:    "-1.5",
			b:    "-0.25",
			want: "-1.75",
		},
		{
			name: "wei granularity",
			a:    "0.0000000000000000001",
			b:    "0.0000000000000000001",
			want: "0.0000000000000000002",
		},
		{
			name: "carry across the point",
			a:    "9.999",
			b:    "0.001",
			want: "10",
		},
		{
			name: "alignment past 128 bits with a fitting result",
			a:    "40000000000000000000000000000000000000",
			b:    "-6000000000000000000000000000000000000.5",
			want: "33999999999999999999999999999999999999.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Add(MustFromString(tt.b)).String(); got != tt.want {
				t.Errorf("Decimal.Add(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDecimal_Add_Panics(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "sum past 128 bits",
			a:    "300000000000000000000000000000000000000",
			b:    "100000000000000000000000000000000000000",
		},
		{
			name: "sum needs forty significant digits",
			a:    "100000000000000000000000000000000000000",
			b:    "0.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, ErrRange) {
					t.Errorf("Decimal.Add(%s, %s) panic = %v, want ErrRange", tt.a, tt.b, err)
				}
			}()
			MustFromString(tt.a).Add(MustFromString(tt.b))
		})
	}
}

func TestDecimal_Sub(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "mixed precision",
			a:    "0.3",
			b:    "0.1",
			want: "0.2",
		},
		{
			name: "through zero",
			a:    "1",
			b:    "2.5",
			want: "-1.5",
		},
		{
			name: "double negation",
			a:    "1",
			b:    "-1",
			want: "2",
		},
		{
			name: "self cancellation",
			a:    "-0.7",
			b:    "-0.7",
			want: "0",
		},
		{
			name: "cancellation through the widened alignment",
			a:    "40000000000000000000000000000000000000",
			b:    "6000000000000000000000000000000000000.5",
			want: "33999999999999999999999999999999999999.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Sub(MustFromString(tt.b)).String(); got != tt.want {
				t.Errorf("Decimal.Sub(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
			}
		})
	}

	t.Run("panics past 128 bits", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.Sub panic = %v, want ErrRange", err)
			}
		}()
		MustFromString("-300000000000000000000000000000000000000").Sub(MustFromString("100000000000000000000000000000000000000"))
	})
}

func TestDecimal_Mul(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "trailing zero shed",
			a:    "1.5",
			b:    "2",
			want: "3",
		},
		{
			name: "signs cross",
			a:    "-0.5",
			b:    "4",
			want: "-2",
		},
		{
			name: "both negative",
			a:    "-0.5",
			b:    "-4",
			want: "2",
		},
		{
			name: "fraction squared",
			a:    "0.1",
			b:    "0.1",
			want: "0.01",
		},
		{
			name: "zero annihilates",
			a:    "0",
			b:    "64500.4",
			want: "0",
		},
		{
			name: "wei times its scale",
			a:    "0.000000000000000001",
			b:    "1000000000000000000",
			want: "1",
		},
		{
			name: "reciprocal scales",
			a:    "1000000",
			b:    "0.000001",
			want: "1",
		},
		{
			name: "double middle carry from the fuzzer",
			a:    "1000000000000000000000000000000000000",
			b:    "9.0000000000000000001",
			want: "9000000000000000000100000000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Mul(MustFromString(tt.b)).String(); got != tt.want {
				t.Errorf("Decimal.Mul(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDecimal_Mul_Panics(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "nonzero fraction past the cap",
			a:    "0.0000000000000000001",
			b:    "0.0000000000000000001",
		},
		{
			name: "product past 128 bits",
			a:    "100000000000000000000",
			b:    "100000000000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, ErrRange) {
					t.Errorf("Decimal.Mul(%s, %s) panic = %v, want ErrRange", tt.a, tt.b, err)
				}
			}()
			MustFromString(tt.a).Mul(MustFromString(tt.b))
		})
	}
}

func TestDecimal_Div(t *testing.T) {
	// The rounded quotients are shopspring's own outputs, pinned as literals.
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "third rounds down",
			a:    "1",
			b:    "3",
			want: "0.3333333333333333",
		},
		{
			name: "two thirds rounds up",
			a:    "2",
			b:    "3",
			want: "0.6666666666666667",
		},
		{
			name: "exact quotient",
			a:    "10",
			b:    "4",
			want: "2.5",
		},
		{
			name: "seventh",
			a:    "1",
			b:    "7",
			want: "0.1428571428571429",
		},
		{
			name: "negative dividend",
			a:    "-1",
			b:    "7",
			want: "-0.1428571428571429",
		},
		{
			name: "negative divisor",
			a:    "7",
			b:    "-2",
			want: "-3.5",
		},
		{
			name: "both negative",
			a:    "-7",
			b:    "-2",
			want: "3.5",
		},
		{
			name: "pi approximant",
			a:    "355",
			b:    "113",
			want: "3.1415929203539823",
		},
		{
			name: "tick quantization",
			a:    "64500.4",
			b:    "0.01",
			want: "6450040",
		},
		{
			name: "zero dividend",
			a:    "0",
			b:    "5",
			want: "0",
		},
		{
			name: "sub-half of the sixteenth digit vanishes",
			a:    "0.00000000000000001",
			b:    "2",
			want: "0",
		},
		{
			name: "wide divisor takes the binary path",
			a:    "12345678901234567890123456789012345678",
			b:    "98765432109876543210987654321",
			want: "124999998.8609375000142383",
		},
		{
			name: "tiny quotient under a wide divisor",
			a:    "1",
			b:    "123456789012345678901234567890",
			want: "0",
		},
		{
			name: "widest dividend by three",
			a:    "340282366920938463463374607431768211455",
			b:    "3",
			want: "113427455640312821154458202477256070485",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.a).Div(MustFromString(tt.b)).String(); got != tt.want {
				t.Errorf("Decimal.Div(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
			}
		})
	}

	t.Run("panics on a zero divisor", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrDivByZero) {
				t.Errorf("Decimal.Div(1, 0) panic = %v, want ErrDivByZero", err)
			}
		}()
		MustFromString("1").Div(Zero)
	})

	t.Run("panics on a quotient past 128 bits", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.Div panic = %v, want ErrRange", err)
			}
		}()
		MustFromString("340000000000000000000000000000000000000").Div(MustFromString("0.0000000000000000001"))
	})
}

func TestDecimal_Neg(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "positive flips",
			v:    "1.5",
			want: "-1.5",
		},
		{
			name: "negative flips",
			v:    "-1.5",
			want: "1.5",
		},
		{
			name: "zero stays unsigned",
			v:    "0",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Neg().String(); got != tt.want {
				t.Errorf("Decimal.Neg(%s) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_Abs(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "negative drops the sign",
			v:    "-1.5",
			want: "1.5",
		},
		{
			name: "positive unchanged",
			v:    "1.5",
			want: "1.5",
		},
		{
			name: "zero unchanged",
			v:    "0",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Abs().String(); got != tt.want {
				t.Errorf("Decimal.Abs(%s) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	if got := Min(MustFromString("2"), MustFromString("-3"), MustFromString("1")); got != MustFromString("-3") {
		t.Errorf("Min(2, -3, 1) = %s, want -3", got)
	}
	if got := Min(MustFromString("2.5")); got != MustFromString("2.5") {
		t.Errorf("Min(2.5) = %s, want 2.5", got)
	}
}

func TestMax(t *testing.T) {
	if got := Max(MustFromString("2"), MustFromString("-3"), MustFromString("1")); got != MustFromString("2") {
		t.Errorf("Max(2, -3, 1) = %s, want 2", got)
	}
	if got := Max(MustFromString("-2.5")); got != MustFromString("-2.5") {
		t.Errorf("Max(-2.5) = %s, want -2.5", got)
	}
}

func TestSum(t *testing.T) {
	if got := Sum(MustFromString("1"), MustFromString("2"), MustFromString("3.5")); got != MustFromString("6.5") {
		t.Errorf("Sum(1, 2, 3.5) = %s, want 6.5", got)
	}
	if got := Sum(MustFromString("-1.5")); got != MustFromString("-1.5") {
		t.Errorf("Sum(-1.5) = %s, want -1.5", got)
	}
}
