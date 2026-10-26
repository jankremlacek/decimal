package decimal

import (
	"errors"
	"strings"
	"testing"

	shopspring "github.com/shopspring/decimal"
)

// FuzzNewFromString holds the parser to shopspring's accept/reject
// line: any text shopspring rejects we reject; any text it accepts
// we either parse to the identical value or refuse as out of
// range — and then its canonical form must not parse either. Every
// accepted value must survive all four wire forms unchanged.
func FuzzNewFromString(f *testing.F) {
	for _, seed := range []string{
		"0", "-0.000", "1.5", "1.500", ".5", "5.", "+5", "-.5", "5.e3",
		"1.5e3", "1.5E+3", "1.2e-5", "12e-19", "00.100", "0.30000000000000004",
		"340282366920938463463374607431768211455",
		"34028236692093846346.3374607431768211455",
		"9999999999999999999999999999999999999999",
		"1e-25", "1e40", "1e2000000", "1000000000000000e-34", "1000000000000001e-34",
		"", "-", ".", "+", "1.2.3", "1_0",
		" 1", "1 ", "abc", "0x10", "e5", "1.5e", "1e+", "NaN",
		"\u2212123.45", "\u2212.5", "1e\u22122", "\u2212", "\xe2\x885",
		"\u2212-5", ".\u22125", "2\u22125", "1e\u2212",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		ref, refErr := shopspring.NewFromString(s)
		d, err := NewFromString(s)
		if refErr != nil {
			// The one deliberate leniency divergence: shopspring
			// rejects an effective exponent past its MaxDecodeExponent
			// to bound its big.Int cost. Our parse costs the text's
			// length alone, so a zero, or an in-range value with
			// a long zero tail, past that cap is ours to accept.
			if err == nil && !strings.Contains(refErr.Error(), "exceeds MaxDecodeExponent") {
				t.Fatalf("NewFromString(%q) = %s, shopspring rejects", s, d)
			}
			return
		}
		if err != nil {
			if !errors.Is(err, ErrRange) {
				t.Fatalf("NewFromString(%q) = %v, shopspring accepts", s, err)
			}
			// Prove the value out of range without rendering it — the text
			// costs shopspring |exp| digits. Past MaxDigits the magnitude
			// exceeds the coefficient width; with nd coefficient digits
			// the magnitude stays under 10^(nd+exp), so at or below
			// -MaxFractionDigits it undercuts our smallest nonzero.
			nd := int64(ref.NumDigits())
			if e := int64(ref.Exponent()); e > MaxDigits || nd+e <= -MaxFractionDigits {
				return
			}
			if _, err := NewFromString(ref.String()); err == nil {
				t.Fatalf("NewFromString(%q) rejected as out of range, yet canonical %s parses", s, ref)
			}
			return
		}
		// Zero compares as zero: shopspring renders a zero with
		// a huge positive exponent by computing 10^exp first.
		if ref.IsZero() {
			if !d.IsZero() {
				t.Fatalf("NewFromString(%q) = %s, shopspring reads zero", s, d)
			}
		} else if got, want := d.String(), ref.String(); got != want {
			t.Fatalf("NewFromString(%q) = %s, shopspring reads %s", s, got, want)
		}
		if re := MustFromString(d.String()); re != d {
			t.Fatalf("NewFromString(%q) = %s is not canonical", s, d)
		}

		text, err := d.MarshalText()
		if err != nil {
			t.Fatalf("Decimal.MarshalText(%s) = %v", d, err)
		}
		var fromText Decimal
		if err := fromText.UnmarshalText(text); err != nil || fromText != d {
			t.Fatalf("Decimal text round-trip of %s = %s, %v", d, fromText, err)
		}
		if got := d.TextLen(); got != len(text) {
			t.Fatalf("Decimal.TextLen(%s) = %d, want %d", d, got, len(text))
		}

		jsonBytes, err := d.MarshalJSON()
		if err != nil {
			t.Fatalf("Decimal.MarshalJSON(%s) = %v", d, err)
		}
		var fromJSON Decimal
		if err := fromJSON.UnmarshalJSON(jsonBytes); err != nil || fromJSON != d {
			t.Fatalf("Decimal JSON round-trip of %s = %s, %v", d, fromJSON, err)
		}

		binary, err := d.MarshalBinary()
		if err != nil {
			t.Fatalf("Decimal.MarshalBinary(%s) = %v", d, err)
		}
		var fromBinary Decimal
		if err := fromBinary.UnmarshalBinary(binary); err != nil || fromBinary != d {
			t.Fatalf("Decimal binary round-trip of %s = %s, %v", d, fromBinary, err)
		}
	})
}

// FuzzArithmetic holds every operation to shopspring's answer.
// An ErrRange panic is legitimate only when shopspring's own
// result does not fit our bounds; anything else is a defect.
func FuzzArithmetic(f *testing.F) {
	for _, seed := range [][2]string{
		{"0.1", "0.2"},
		{"1.5", "-1.5"},
		{"64500.4", "0.01"},
		{"0.0000000000000000001", "0.0000000000000000001"},
		{"100000000000000000000000000000000000000", "34000000000000000000000000000000000000.5"},
		{"12345678901234567890123456789012345678", "98765432109876543210987654321"},
		{"1", "3"},
		{"355", "113"},
		{"0", "0"},
	} {
		f.Add(seed[0], seed[1], int32(0))
		f.Add(seed[0], seed[1], int32(-21))
		f.Add(seed[0], seed[1], int32(7))
	}
	// 10^15's neighbourhood is where a float64 log10 digit count runs one
	// short (shopspring/decimal#420); 14 maps to 15 of its 16 figures.
	f.Add("1000000000000001", "1", int32(14))
	f.Fuzz(func(t *testing.T, a, b string, places int32) {
		x, errX := NewFromString(a)
		y, errY := NewFromString(b)
		_, errRX := shopspring.NewFromString(a)
		_, errRY := shopspring.NewFromString(b)
		if errX != nil || errY != nil || errRX != nil || errRY != nil {
			return
		}
		// Build the oracle operands from our canonical text rather
		// than the raw input: the values are identical, but
		// a spelling like 0E10000 leaves shopspring an exponent
		// it rescales by 10^exp inside every operation.
		rx := shopspring.RequireFromString(x.String())
		ry := shopspring.RequireFromString(y.String())

		fits := func(s shopspring.Decimal) bool {
			_, err := NewFromString(s.String())
			return err == nil
		}
		capture := func(op func() Decimal) (d Decimal, ok bool) {
			defer func() {
				if p := recover(); p != nil {
					err, is := p.(error)
					if !is || !errors.Is(err, ErrRange) {
						panic(p)
					}
					ok = false
				}
			}()
			return op(), true
		}
		check := func(name string, want shopspring.Decimal, op func() Decimal) {
			got, ok := capture(op)
			if !ok {
				if fits(want) {
					t.Fatalf("Decimal.%s(%s, %s) panicked, yet %s fits", name, a, b, want)
				}
				return
			}
			if got.String() != want.String() {
				t.Fatalf("Decimal.%s(%s, %s) = %s, shopspring computes %s", name, a, b, got, want)
			}
			if MustFromString(got.String()) != got {
				t.Fatalf("Decimal.%s(%s, %s) = %s is not canonical", name, a, b, got)
			}
		}

		check("Add", rx.Add(ry), func() Decimal { return x.Add(y) })
		check("Sub", rx.Sub(ry), func() Decimal { return x.Sub(y) })
		check("Mul", rx.Mul(ry), func() Decimal { return x.Mul(y) })
		if !y.IsZero() {
			check("Div", rx.Div(ry), func() Decimal { return x.Div(y) })
		}

		if got, want := x.Cmp(y), rx.Cmp(ry); got != want {
			t.Fatalf("Decimal.Cmp(%s, %s) = %d, shopspring computes %d", a, b, got, want)
		}
		if x.Equal(y) != rx.Equal(ry) {
			t.Fatalf("Decimal.Equal(%s, %s) = %t, shopspring disagrees", a, b, x.Equal(y))
		}
		if x.Neg().String() != rx.Neg().String() || x.Abs().String() != rx.Abs().String() {
			t.Fatalf("Decimal.Neg/Abs(%s) diverges from shopspring", a)
		}
		if got, want := x.InexactFloat64(), rx.InexactFloat64(); got != want {
			t.Fatalf("Decimal.InexactFloat64(%s) = %v, shopspring computes %v", a, got, want)
		}
		if Min(x, y).String() != shopspring.Min(rx, ry).String() || Max(x, y).String() != shopspring.Max(rx, ry).String() {
			t.Fatalf("Min/Max(%s, %s) diverges from shopspring", a, b)
		}

		// The oracle's leading-digit exponent is shopspring's
		// own. sigFigs spans 1..40, past the widest
		// coefficient, so the unchanged path is reached too.
		sigFigs := 1 + int32(uint32(places)%40)
		refPlaces := sigFigs - 1 - (int32(rx.NumDigits()) - 1 + rx.Exponent())
		check("RoundFloorSigFigs", rx.RoundFloor(refPlaces), func() Decimal {
			return x.RoundFloorSigFigs(sigFigs)
		})

		// The rounding family shares one bounded places argument.
		places %= 25
		check("Round", rx.Round(places), func() Decimal {
			return x.Round(places)
		})
		check("RoundFloor", rx.RoundFloor(places), func() Decimal {
			return x.RoundFloor(places)
		})
		check("RoundCeil", rx.RoundCeil(places), func() Decimal {
			return x.RoundCeil(places)
		})
		check("Truncate", rx.Truncate(places), func() Decimal {
			return x.Truncate(places)
		})
		if x.Floor().String() != rx.Floor().String() || x.Ceil().String() != rx.Ceil().String() {
			t.Fatalf("Decimal.Floor/Ceil(%s) diverges from shopspring", a)
		}
		// StringFixed is Round plus rendering, so it may only panic where Round legitimately did.
		if _, ok := capture(func() Decimal { return x.Round(places) }); ok {
			if got, want := x.StringFixed(places), rx.StringFixed(places); got != want {
				t.Fatalf("Decimal.StringFixed(%s, %d) = %q, shopspring renders %q", a, places, got, want)
			}
		}
	})
}
