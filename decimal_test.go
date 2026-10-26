package decimal

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		v    int64
		exp  int32
		want string
	}{
		{
			name: "thousandths",
			v:    1234,
			exp:  -3,
			want: "1.234",
		},
		{
			name: "negative thousandths",
			v:    -1234,
			exp:  -3,
			want: "-1.234",
		},
		{
			name: "tenths",
			v:    15,
			exp:  -1,
			want: "1.5",
		},
		{
			name: "zero swallows the exponent",
			v:    0,
			exp:  5,
			want: "0",
		},
		{
			name: "zero swallows the highest exponent",
			v:    0,
			exp:  math.MaxInt32,
			want: "0",
		},
		{
			name: "zero swallows the deepest exponent",
			v:    0,
			exp:  math.MinInt32,
			want: "0",
		},
		{
			name: "positive exponent scales up",
			v:    42,
			exp:  3,
			want: "42000",
		},
		{
			name: "trailing zeros shed",
			v:    100,
			exp:  -2,
			want: "1",
		},
		{
			name: "exponent below the cap sheds back into it",
			v:    1000,
			exp:  -22,
			want: "0.0000000000000000001",
		},
		{
			name: "int64 min",
			v:    math.MinInt64,
			exp:  0,
			want: "-9223372036854775808",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.v, tt.exp).String(); got != tt.want {
				t.Errorf("New(%d, %d) = %s, want %s", tt.v, tt.exp, got, tt.want)
			}
		})
	}
}

func TestNew_Panics(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		exp   int32
	}{
		{name: "scales past 128 bits", value: 1, exp: 40},
		{name: "fraction below the cap", value: 1, exp: -40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, ErrRange) {
					t.Errorf("New(%d, %d) panic = %v, want ErrRange", tt.value, tt.exp, err)
				}
			}()
			New(tt.value, tt.exp)
		})
	}
}

func TestNewFromInt(t *testing.T) {
	tests := []struct {
		name string
		v    int64
		want string
	}{
		{
			name: "zero",
			v:    0,
			want: "0",
		},
		{
			name: "positive",
			v:    42,
			want: "42",
		},
		{
			name: "negative",
			v:    -42,
			want: "-42",
		},
		{
			name: "int64 max",
			v:    math.MaxInt64,
			want: "9223372036854775807",
		},
		{
			name: "int64 min",
			v:    math.MinInt64,
			want: "-9223372036854775808",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFromInt(tt.v).String(); got != tt.want {
				t.Errorf("NewFromInt(%d) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestNewFromFloat(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		want string
	}{
		{
			name: "zero",
			v:    0,
			want: "0",
		},
		{
			name: "price",
			v:    100.5,
			want: "100.5",
		},
		{
			name: "negative tenth",
			v:    -0.1,
			want: "-0.1",
		},
		{
			name: "small fraction",
			v:    0.000123,
			want: "0.000123",
		},
		{
			name: "wei",
			v:    5e-18,
			want: "0.000000000000000005",
		},
		{
			name: "large integer",
			v:    1e18,
			want: "1000000000000000000",
		},
		{
			name: "shortest round-trip text",
			v:    0.30000000000000004,
			want: "0.30000000000000004",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFromFloat(tt.v).String(); got != tt.want {
				t.Errorf("NewFromFloat(%v) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestNewFromFloat_Panics(t *testing.T) {
	tests := []struct {
		name    string
		v       float64
		wantErr error
	}{
		{
			name:    "NaN",
			v:       math.NaN(),
			wantErr: ErrSyntax,
		},
		{
			name:    "positive infinity",
			v:       math.Inf(1),
			wantErr: ErrSyntax,
		},
		{
			name:    "negative infinity",
			v:       math.Inf(-1),
			wantErr: ErrSyntax,
		},
		{
			name:    "fraction below the cap",
			v:       1e-25,
			wantErr: ErrRange,
		},
		{
			// An ordinary magnitude whose shortest round-tripping text still
			// needs more fraction digits than a Decimal holds — the shape
			// a computed float takes, unlike the obviously tiny cases.
			name:    "shortest text wider than the cap",
			v:       5.551115123125783e-17,
			wantErr: ErrRange,
		},
		{
			name:    "magnitude past 128 bits",
			v:       1e39,
			wantErr: ErrRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, tt.wantErr) {
					t.Errorf("NewFromFloat(%v) panic = %v, want %v", tt.v, err, tt.wantErr)
				}
			}()
			NewFromFloat(tt.v)
		})
	}
}

func TestNewFromString(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "integer",
			v:    "42",
			want: "42",
		},
		{
			name: "fraction",
			v:    "1.5",
			want: "1.5",
		},
		{
			name: "negative",
			v:    "-0.10",
			want: "-0.1",
		},
		{
			name: "plus sign",
			v:    "+5",
			want: "5",
		},
		{
			name: "bare leading dot",
			v:    ".5",
			want: "0.5",
		},
		{
			name: "signed leading dot",
			v:    "-.5",
			want: "-0.5",
		},
		{
			name: "unicode minus",
			v:    "\u2212123.45",
			want: "-123.45",
		},
		{
			name: "unicode minus before a leading dot",
			v:    "\u2212.5",
			want: "-0.5",
		},
		{
			name: "unicode minus zero",
			v:    "\u22120",
			want: "0",
		},
		{
			name: "bare trailing dot",
			v:    "5.",
			want: "5",
		},
		{
			name: "trailing dot with exponent",
			v:    "5.e3",
			want: "5000",
		},
		{
			name: "leading zeros",
			v:    "00.100",
			want: "0.1",
		},
		{
			name: "zero at depth",
			v:    "0.000",
			want: "0",
		},
		{
			name: "negative zero",
			v:    "-0",
			want: "0",
		},
		{
			name: "lowercase exponent",
			v:    "1.5e3",
			want: "1500",
		},
		{
			name: "uppercase signed exponent",
			v:    "1.5E+3",
			want: "1500",
		},
		{
			name: "negative exponent",
			v:    "1.2e-5",
			want: "0.000012",
		},
		{
			name: "unicode minus exponent",
			v:    "1.2e\u22125",
			want: "0.000012",
		},
		{
			name: "exponent onto the fraction cap",
			v:    "12e-19",
			want: "0.0000000000000000012",
		},
		{
			name: "twenty fraction digits shed to nineteen",
			v:    "0.00000000000000000010",
			want: "0.0000000000000000001",
		},
		{
			name: "widest coefficient",
			v:    "340282366920938463463374607431768211455",
			want: "340282366920938463463374607431768211455",
		},
		{
			name: "widest full-precision value",
			v:    "34028236692093846346.3374607431768211455",
			want: "34028236692093846346.3374607431768211455",
		},
		{
			name: "wide zero tail folds away",
			v:    "7000.00000000000000000000000000000000000",
			want: "7000",
		},
		{
			name: "integer zero tail meets its exponent",
			v:    "7" + strings.Repeat("0", 40) + "e-4",
			want: "7" + strings.Repeat("0", 36),
		},
		{
			name: "zero tail past shopspring's exponent cap folds away",
			v:    "7." + strings.Repeat("0", 10001),
			want: "7",
		},
		{
			name: "zero swallows any exponent",
			v:    "0e2000000",
			want: "0",
		},
		{
			name: "zero swallows the int32 exponent floor",
			v:    "0e-2147483648",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewFromString(tt.v)
			if err != nil {
				t.Fatalf("NewFromString(%q) = %v", tt.v, err)
			}
			if got := d.String(); got != tt.want {
				t.Errorf("NewFromString(%q) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestNewFromString_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		v       string
		wantErr error
	}{
		{
			name:    "empty",
			v:       "",
			wantErr: ErrSyntax,
		},
		{
			name:    "bare sign",
			v:       "-",
			wantErr: ErrSyntax,
		},
		{
			name:    "bare plus",
			v:       "+",
			wantErr: ErrSyntax,
		},
		{
			name:    "bare unicode minus",
			v:       "\u2212",
			wantErr: ErrSyntax,
		},
		{
			name:    "truncated unicode minus",
			v:       "\xe2\x885",
			wantErr: ErrSyntax,
		},
		{
			name:    "two minus signs",
			v:       "\u2212-5",
			wantErr: ErrSyntax,
		},
		{
			name:    "unicode minus after the dot",
			v:       ".\u22125",
			wantErr: ErrSyntax,
		},
		{
			name:    "unicode minus between digits",
			v:       "2\u22125",
			wantErr: ErrSyntax,
		},
		{
			name:    "bare dot",
			v:       ".",
			wantErr: ErrSyntax,
		},
		{
			name:    "sign after the dot",
			v:       ".+0",
			wantErr: ErrSyntax,
		},
		{
			name:    "negative sign after the dot",
			v:       ".-5",
			wantErr: ErrSyntax,
		},
		{
			name:    "two dots",
			v:       "1.2.3",
			wantErr: ErrSyntax,
		},
		{
			name:    "underscore",
			v:       "1_0",
			wantErr: ErrSyntax,
		},
		{
			name:    "leading space",
			v:       " 1",
			wantErr: ErrSyntax,
		},
		{
			name:    "trailing space",
			v:       "1 ",
			wantErr: ErrSyntax,
		},
		{
			name:    "trailing garbage",
			v:       "1.5x",
			wantErr: ErrSyntax,
		},
		{
			name:    "trailing garbage after exponent",
			v:       "1e5x",
			wantErr: ErrSyntax,
		},
		{
			name:    "letters",
			v:       "abc",
			wantErr: ErrSyntax,
		},
		{
			name:    "hex",
			v:       "0x10",
			wantErr: ErrSyntax,
		},
		{
			name:    "bare exponent",
			v:       "e5",
			wantErr: ErrSyntax,
		},
		{
			name:    "empty exponent",
			v:       "1.5e",
			wantErr: ErrSyntax,
		},
		{
			name:    "signed empty exponent",
			v:       "1e+",
			wantErr: ErrSyntax,
		},
		{
			name:    "unicode minus empty exponent",
			v:       "1e\u2212",
			wantErr: ErrSyntax,
		},
		{
			name:    "forty digit coefficient",
			v:       "9999999999999999999999999999999999999999",
			wantErr: ErrRange,
		},
		{
			name:    "forty fraction digits",
			v:       "0.0000000000000000000000000000000000000009",
			wantErr: ErrRange,
		},
		{
			name:    "twenty fraction digits",
			v:       "0.00000000000000000001",
			wantErr: ErrRange,
		},
		{
			name:    "exponent past 128 bits",
			v:       "1e40",
			wantErr: ErrRange,
		},
		{
			name:    "exponent below the cap",
			v:       "1e-20",
			wantErr: ErrRange,
		},
		{
			name:    "exponent far past the range",
			v:       "1e9999999",
			wantErr: ErrRange,
		},
		{
			name:    "negative exponent far past the range",
			v:       "1e-9999999",
			wantErr: ErrRange,
		},
		{
			name:    "exponent past int32",
			v:       "1e2147483648",
			wantErr: ErrRange,
		},
		{
			name:    "exponent digits past int32",
			v:       "1e99999999999",
			wantErr: ErrRange,
		},
		{
			name:    "interior zeros span past the width",
			v:       "1" + strings.Repeat("0", 38) + "1",
			wantErr: ErrRange,
		},
		{
			name:    "interior zeros spill on the fold itself",
			v:       "5" + strings.Repeat("0", 38) + "1",
			wantErr: ErrRange,
		},
		{
			name:    "interior zero run past the width",
			v:       "7." + strings.Repeat("0", 40) + "7",
			wantErr: ErrRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewFromString(tt.v); !errors.Is(err, tt.wantErr) {
				t.Errorf("NewFromString(%q) = %v, want %v", tt.v, err, tt.wantErr)
			}
		})
	}
}

func TestMustFromString(t *testing.T) {
	t.Run("well formed", func(t *testing.T) {
		if got := MustFromString("1.5").String(); got != "1.5" {
			t.Errorf("MustFromString(1.5) = %s, want 1.5", got)
		}
	})
	t.Run("panics on malformed", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrSyntax) {
				t.Errorf("MustFromString(abc) panic = %v, want ErrSyntax", err)
			}
		}()
		MustFromString("abc")
	})
}

// TestMaxFractionDigits ties the exported constant to the parser's actual
// limit: a fraction that wide parses, one digit wider does not.
func TestMaxFractionDigits(t *testing.T) {
	tests := []struct {
		name    string
		v       string
		wantErr bool
	}{
		{
			name:    "at the limit",
			v:       "0.0000000000000000001",
			wantErr: false,
		},
		{
			name:    "one past the limit",
			v:       "0.00000000000000000001",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFromString(tt.v)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewFromString(%s) error = nil, want error", tt.v)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewFromString(%s) error: %v", tt.v, err)
			}
			if want := int32(-MaxFractionDigits); got.Exponent() != want {
				t.Errorf("NewFromString(%s).Exponent() = %d, want %d", tt.v, got.Exponent(), want)
			}
		})
	}
}

func TestDecimal_Exponent(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want int32
	}{
		{
			name: "integer",
			v:    "1000",
			want: 0,
		},
		{
			name: "trailing zero shed first",
			v:    "1.50",
			want: -1,
		},
		{
			name: "full fraction",
			v:    "0.0000000000000000001",
			want: -19,
		},
		{
			name: "zero",
			v:    "0",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Exponent(); got != tt.want {
				t.Errorf("Decimal.Exponent(%s) = %d, want %d", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_ZeroValue(t *testing.T) {
	var d Decimal
	if !d.IsZero() {
		t.Errorf("Decimal zero value IsZero() = false, want true")
	}
	if d != Zero {
		t.Errorf("Decimal zero value != Zero")
	}
	if got := d.String(); got != "0" {
		t.Errorf("Decimal zero value String() = %s, want 0", got)
	}
}

func TestDecimal_Canonical(t *testing.T) {
	// Equal values must land on the identical representation, whatever
	// text spelled them: that is what makes == value equality.
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "trailing zeros",
			a:    "1.5",
			b:    "1.500",
		},
		{
			name: "negative zero at depth",
			a:    "0",
			b:    "-0.000",
		},
		{
			name: "exponent form",
			a:    "1000",
			b:    "1e3",
		},
		{
			name: "fraction exponent form",
			a:    "0.1",
			b:    "1e-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if MustFromString(tt.a) != MustFromString(tt.b) {
				t.Errorf("MustFromString(%q) != MustFromString(%q), want identical representations", tt.a, tt.b)
			}
		})
	}
}

func TestDecimal_Immutability(t *testing.T) {
	// call takes *Decimal so that a method regressed to a pointer
	// receiver would mutate the checked value and fail here;
	// a value receiver leaves it untouched. Each in is chosen so
	// its op changes the value, making a stray mutation observable.
	tests := []struct {
		name string
		v    string
		call func(d *Decimal) Decimal
	}{
		{
			name: "Add",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Add(MustFromString("2.5"))
			},
		},
		{
			name: "Sub",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Sub(MustFromString("2.5"))
			},
		},
		{
			name: "Mul",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Mul(MustFromString("2.5"))
			},
		},
		{
			name: "Div",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Div(MustFromString("2.5"))
			},
		},
		{
			name: "Neg",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Neg()
			},
		},
		{
			name: "Abs",
			v:    "-1.5",
			call: func(d *Decimal) Decimal {
				return d.Abs()
			},
		},
		{
			name: "Round",
			v:    "1.55",
			call: func(d *Decimal) Decimal {
				return d.Round(1)
			},
		},
		{
			name: "RoundFloor",
			v:    "1.55",
			call: func(d *Decimal) Decimal {
				return d.RoundFloor(1)
			},
		},
		{
			name: "RoundFloorSigFigs",
			v:    "1.55",
			call: func(d *Decimal) Decimal {
				return d.RoundFloorSigFigs(2)
			},
		},
		{
			name: "RoundCeil",
			v:    "1.51",
			call: func(d *Decimal) Decimal {
				return d.RoundCeil(1)
			},
		},
		{
			name: "Truncate",
			v:    "1.55",
			call: func(d *Decimal) Decimal {
				return d.Truncate(1)
			},
		},
		{
			name: "Floor",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Floor()
			},
		},
		{
			name: "Ceil",
			v:    "1.5",
			call: func(d *Decimal) Decimal {
				return d.Ceil()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MustFromString(tt.v)
			before := d
			_ = tt.call(&d)
			if d != before {
				t.Errorf("Decimal.%s mutated its receiver: d = %s, want %s", tt.name, d, before)
			}
		})
	}
}
