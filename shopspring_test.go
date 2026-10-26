// Portions of this file are ported from shopspring/decimal under its MIT License:
//
//	Copyright (c) 2015 Spring, Inc.
//	Copyright (c) 2013 Oguz Bilgic
//
// This file ports shopspring/decimal v1.6.0's own decimal_test.go,
// decimal_go113_test.go and decimal_go124_test.go fixtures, 1:1,
// against our implementation — every fixture of every method both
// packages share. Where our design diverges, the case carries our
// expected outcome with a comment naming the divergence. Upstream's
// arbitrary-precision fixtures (the exact/inexact float expansions,
// 250-digit magnitudes) exceed our 39-digit / 19-fraction bounds
// by construction; they appear here as range-rejection cases
// rather than being dropped. Upstream tests of the methods outside
// our subset (Float64, NewFromFloat32, DivRound, Shift, Equals,
// IntPart, Rat, Sqrt, Format, ...) have no counterpart here.
package decimal

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// didPanic is upstream's panic probe, verbatim.
func didPanic(f func()) bool {
	ret := false
	func() {
		defer func() {
			if message := recover(); message != nil {
				ret = true
			}
		}()
		f()
	}()

	return ret
}

// shopspringTable is upstream testTable's float/short pairs that fit our
// bounds; the out-of-range rows live in TestShopspring_OutOfRange. Negative
// counterparts are generated in the tests exactly as upstream's init does.
var shopspringTable = []struct {
	float float64
	short string
}{
	{3.141592653589793, "3.141592653589793"},
	{3, "3"},
	{1234567890123456, "1234567890123456"},
	{1234567890123456000, "1234567890123456000"},
	{1234.567890123456, "1234.567890123456"},
	{.1234567890123456, "0.1234567890123456"},
	{0, "0"},
	{.1111111111111110, "0.111111111111111"},
	{.1111111111111111, "0.1111111111111111"},
	{.1111111111111119, "0.1111111111111119"},
	{.000000000000000001, "0.000000000000000001"},
	{.000000000000000002, "0.000000000000000002"},
	{.000000000000000003, "0.000000000000000003"},
	{.000000000000000005, "0.000000000000000005"},
	{.000000000000000008, "0.000000000000000008"},
	{.1000000000000001, "0.1000000000000001"},
	{.1000000000000002, "0.1000000000000002"},
	{.1000000000000003, "0.1000000000000003"},
	{.1000000000000005, "0.1000000000000005"},
	{.1000000000000008, "0.1000000000000008"},
	{1e25, "10000000000000000000000000"},
	{1.5e14, "150000000000000"},
	{1.5e15, "1500000000000000"},
	{1.5e16, "15000000000000000"},
	{1.0001e25, "10001000000000000000000000"},
	{1.0001000000000000033e25, "10001000000000000000000000"},
	{2e25, "20000000000000000000000000"},
	{4e25, "40000000000000000000000000"},
	{8e25, "80000000000000000000000000"},
	// Upstream's math.MaxInt64 row: the expected text is FormatFloat's
	// shortest round-trip form of float64(MaxInt64), not the integer.
	{9.223372036854776e18, "9223372036854776000"},
	// Upstream's decimal_go113_test.go appends these two (golang/go#29491).
	{498484681984085570, "498484681984085570"},
	{5.8339553793802237e+23, "583395537938022370000000"},
}

// shopspringScientific is upstream testTableScientificNotation, verbatim.
var shopspringScientific = map[string]string{
	"1e9":        "1000000000",
	"2.41E-3":    "0.00241",
	"24.2E-4":    "0.00242",
	"243E-5":     "0.00243",
	"1e-5":       "0.00001",
	"245E3":      "245000",
	"1.2345E-1":  "0.12345",
	"0e5":        "0",
	"0e-5":       "0",
	"0.e0":       "0",
	".0e0":       "0",
	"123.456e0":  "123.456",
	"123.456e2":  "12345.6",
	"123.456e10": "1234560000000",
}

func TestShopspring_NewFromFloat(t *testing.T) {
	for _, x := range shopspringTable {
		if got := NewFromFloat(x.float).String(); got != x.short {
			t.Errorf("NewFromFloat(%v) = %s, want %s", x.float, got, x.short)
		}
		if x.float > 0 && x.short != "0" {
			if got := NewFromFloat(-x.float).String(); got != "-"+x.short {
				t.Errorf("NewFromFloat(%v) = %s, want -%s", -x.float, got, x.short)
			}
		}
		if got := MustFromString(x.short).InexactFloat64(); got != x.float {
			t.Errorf("Decimal.InexactFloat64(%s) = %v, want %v", x.short, got, x.float)
		}
	}

	shouldPanicOn := []float64{
		math.NaN(),
		math.Inf(1),
		math.Inf(-1),
	}
	for _, n := range shouldPanicOn {
		if !didPanic(func() { NewFromFloat(n) }) {
			t.Fatalf("NewFromFloat(%v) did not panic", n)
		}
	}
}

func TestShopspring_NewFromFloatShortestRoundTrip(t *testing.T) {
	// Upstream's NewFromFloat32 half has no counterpart, and
	// its Float64 is our InexactFloat64. A float whose shortest
	// form falls outside our representable set rejects instead:
	// the parse with ErrRange, and NewFromFloat by panicking.
	f64 := []float64{
		1, -1, 0.1, -0.1, 0.5, 2.5, 100, 123.456,
		1e15, 1e16, 1e17, 1e21, 1e22, 1e23, 9007199254740992,
		1e-100, 1e100, 1e-300, 1e300,
		math.MaxFloat64, -math.MaxFloat64,
		math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
		math.Pi, math.E,
	}
	for e := -1074; e <= 1023; e++ {
		f64 = append(f64, math.Ldexp(1, e))
	}
	for _, f := range slices.Clone(f64) {
		f64 = append(f64, math.Nextafter(f, math.Inf(1)), math.Nextafter(f, math.Inf(-1)))
	}

	for _, f := range f64 {
		if f == 0 || math.IsInf(f, 0) {
			continue
		}
		want, err := NewFromString(strconv.FormatFloat(f, 'f', -1, 64))
		if errors.Is(err, ErrRange) {
			if !didPanic(func() { NewFromFloat(f) }) {
				t.Errorf("NewFromFloat(%v) did not panic, want ErrRange panic", f)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NewFromString(%v) = %v", f, err)
		}
		got := NewFromFloat(f)
		if got != want {
			t.Errorf("NewFromFloat(%v) = %s, want %s", f, got, want)
		}
		if back := got.InexactFloat64(); back != f {
			t.Errorf("Decimal.InexactFloat64(%s) = %v, want %v", got, back, f)
		}
	}
}

func TestShopspring_NewFromFloatRandomRoundTrip(t *testing.T) {
	// Upstream's NewFromFloat32 half has no counterpart; out-of-range floats
	// reject as in TestShopspring_NewFromFloatShortestRoundTrip.
	rng := rand.New(rand.NewPCG(0xdead1337, 0xdead1337))
	for range 200000 {
		f := math.Float64frombits(rng.Uint64())
		if f == 0 || math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		want, err := NewFromString(strconv.FormatFloat(f, 'f', -1, 64))
		if errors.Is(err, ErrRange) {
			if !didPanic(func() { NewFromFloat(f) }) {
				t.Fatalf("NewFromFloat(%v) did not panic, want ErrRange panic", f)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NewFromString(%v) = %v", f, err)
		}
		if got := NewFromFloat(f); got != want {
			t.Fatalf("NewFromFloat(%v) = %s, want %s", f, got, want)
		}
	}
}

func TestShopspring_NewFromString(t *testing.T) {
	for _, x := range shopspringTable {
		d, err := NewFromString(x.short)
		if err != nil {
			t.Errorf("NewFromString(%q) = %v", x.short, err)
		} else if d.String() != x.short {
			t.Errorf("NewFromString(%q) = %s, want %s", x.short, d.String(), x.short)
		}
	}

	for e, s := range shopspringScientific {
		d, err := NewFromString(e)
		if err != nil {
			t.Errorf("NewFromString(%q) = %v", e, err)
		} else if d.String() != s {
			t.Errorf("NewFromString(%q) = %s, want %s", e, d.String(), s)
		}
		if e[0] != '-' && s != "0" {
			d, err := NewFromString("-" + e)
			if err != nil {
				t.Errorf("NewFromString(%q) = %v", "-"+e, err)
			} else if d.String() != "-"+s {
				t.Errorf("NewFromString(%q) = %s, want -%s", "-"+e, d.String(), s)
			}
		}
	}

	// Upstream's testMalformedDecimalStrings also pins
	// its own error text; only the rejection ports.
	for _, s := range []string{"1ee10", "123.45.66"} {
		if _, err := NewFromString(s); err == nil {
			t.Errorf("NewFromString(%q) = nil error, want error", s)
		}
	}
}

func TestShopspring_NewFromStringUnicodeMinus(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"\u2212123.45", "-123.45"},
		{"\u2212.5", "-0.5"},
		{"1e\u22122", "0.01"},
	}
	for _, tc := range cases {
		d, err := NewFromString(tc.in)
		if err != nil {
			t.Errorf("NewFromString(%q) = %v", tc.in, err)
			continue
		}
		if got := d.String(); got != tc.want {
			t.Errorf("NewFromString(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestShopspring_NewFromStringErrs(t *testing.T) {
	// Upstream's list verbatim; every entry is a rejection for us too.
	tests := []string{
		"",
		"qwert",
		"-",
		".",
		"-.",
		".-",
		"234-.56",
		"234-56",
		"2-",
		"..",
		"2..",
		"..2",
		".5.2",
		"8..2",
		"8.1.",
		"1e",
		"1-e",
		"1e9e",
		"1ee9",
		"1ee",
		"1eE",
		"1e-",
		"1e-.",
		"1e1.2",
		"123.456e1.3",
		"1e-1.2",
		"123.456e-1.3",
		"123.456Easdf",
		"123.456e-9223372036854775808",
		"123.456e-2147483648",
		"512.99 USD",
		"$99.99",
		"51,850.00",
		"20_000_000.00",
		"$20_000_000.00",
		".-5",
		".+5",
		".-1234567890123456789012",
	}
	for _, s := range tests {
		if _, err := NewFromString(s); err == nil {
			t.Errorf("NewFromString(%q) = nil error, want error", s)
		}
	}
}

func TestShopspring_MaxDecodeExponent(t *testing.T) {
	// Divergence: upstream accepts an exponent of ±MaxDecodeExponent
	// (10000) and rejects one past it, a cap we do not mirror
	// (see FuzzNewFromString). Every row lies outside our
	// representable set, so each rejects as out of range.
	// The binary half has no counterpart: the wire layout is ours.
	tests := []string{
		"1e10000",
		"1e-10000",
		"1e10001",
		"1e-10001",
		"1e2147483647",
		"1e-2147483648",
		"0.1e-10000",
	}
	for _, s := range tests {
		if _, err := NewFromString(s); !errors.Is(err, ErrRange) {
			t.Errorf("NewFromString(%q) = %v, want ErrRange", s, err)
		}
	}
}

func TestShopspring_NewFromStringDeepEquals(t *testing.T) {
	// Divergence: upstream distinguishes representations, so
	// "10" and "10.0" are not reflect.DeepEqual. We normalize on
	// construction — equal values share the one canonical representation,
	// and == is value equality — so the trailing-zero rows flip to true.
	tests := []struct {
		str1     string
		str2     string
		expected bool
	}{
		{"1", "1", true},
		{"1.0", "1.0", true},
		{"10", "10.0", true},  // upstream: false
		{"1.1", "1.10", true}, // upstream: false
		{"1.001", "1.01", false},
	}
	for _, tt := range tests {
		d1, err1 := NewFromString(tt.str1)
		d2, err2 := NewFromString(tt.str2)
		if err1 != nil || err2 != nil {
			t.Errorf("NewFromString(%q, %q) = %v, %v", tt.str1, tt.str2, err1, err2)
		}
		if got := d1 == d2; got != tt.expected {
			t.Errorf("NewFromString(%q) == NewFromString(%q) is %t, want %t", tt.str1, tt.str2, got, tt.expected)
		}
	}
}

func TestShopspring_RequireFromString(t *testing.T) {
	// Upstream's RequireFromString is our MustFromString.
	s := "1.23"
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("MustFromString(%q) panicked: %v", s, err)
		}
	}()
	if d := MustFromString(s); d.String() != s {
		t.Errorf("MustFromString(%q) = %s, want %s", s, d.String(), s)
	}
}

func TestShopspring_RequireFromStringErrs(t *testing.T) {
	if !didPanic(func() { MustFromString("qwert") }) {
		t.Errorf("MustFromString(qwert) did not panic")
	}
}

func TestShopspring_NewFromInt(t *testing.T) {
	tests := map[int64]string{
		0:                    "0",
		1:                    "1",
		323412345:            "323412345",
		9223372036854775807:  "9223372036854775807",
		-9223372036854775808: "-9223372036854775808",
	}
	for p, s := range tests {
		if p > 0 {
			tests[-p] = "-" + s
		}
	}
	for input, s := range tests {
		if got := NewFromInt(input).String(); got != s {
			t.Errorf("NewFromInt(%d) = %s, want %s", input, got, s)
		}
	}
}

func TestShopspring_JSON(t *testing.T) {
	// Divergence: upstream's MarshalJSONWithoutQuotes global has no
	// counterpart — we always quote — so its unquoted branch is absent.
	for _, x := range shopspringTable {
		s := x.short
		var doc struct {
			Amount Decimal `json:"amount"`
		}
		docStr := `{"amount":"` + s + `"}`
		if err := json.Unmarshal([]byte(docStr), &doc); err != nil {
			t.Errorf("json.Unmarshal(%s) = %v", docStr, err)
		} else if doc.Amount.String() != s {
			t.Errorf("json.Unmarshal(%s) decoded amount %s, want %s", docStr, doc.Amount.String(), s)
		}

		out, err := json.Marshal(&doc)
		if err != nil {
			t.Errorf("json.Marshal(%+v) = %v", doc, err)
		} else if string(out) != docStr {
			t.Errorf("json.Marshal(%+v) = %s, want %s", doc, string(out), docStr)
		}
	}
}

func TestShopspring_UnmarshalJSONNull(t *testing.T) {
	var doc struct {
		Amount Decimal `json:"amount"`
	}
	docStr := `{"amount": null}`
	if err := json.Unmarshal([]byte(docStr), &doc); err != nil {
		t.Errorf("json.Unmarshal(%s) = %v", docStr, err)
	} else if !doc.Amount.Equal(Zero) {
		t.Errorf("json.Unmarshal(%s) decoded amount %s, want 0", docStr, doc.Amount.String())
	}
}

func TestShopspring_BadJSON(t *testing.T) {
	for _, testCase := range []string{
		"]o_o[",
		"{",
		`{"amount":""`,
		`{"amount":""}`,
		`{"amount":"nope"}`,
		`0.333`,
	} {
		var doc struct {
			Amount Decimal `json:"amount"`
		}
		if err := json.Unmarshal([]byte(testCase), &doc); err == nil {
			t.Errorf("json.Unmarshal(%q) = nil error, want error", testCase)
		}
	}
}

func TestShopspring_Floor(t *testing.T) {
	assertFloor := func(input, expected Decimal) {
		if got := input.Floor(); !got.Equal(expected) {
			t.Errorf("Decimal.Floor(%s) = %s, want %s", input, got, expected)
		}
	}
	testsWithStrings := []struct {
		input    string
		expected string
	}{
		{"1.999", "1"},
		{"1", "1"},
		{"1.01", "1"},
		{"0", "0"},
		{"0.9", "0"},
		{"0.1", "0"},
		{"-0.9", "-1"},
		{"-0.1", "-1"},
		{"-1.00", "-1"},
		{"-1.01", "-2"},
		{"-1.999", "-2"},
	}
	for _, test := range testsWithStrings {
		assertFloor(MustFromString(test.input), MustFromString(test.expected))
	}

	testsWithDecimals := []struct {
		input    Decimal
		expected string
	}{
		{New(100, -1), "10"},
		{New(10, 0), "10"},
		{New(1, 1), "10"},
		{New(1999, -3), "1"},
		{New(101, -2), "1"},
		{New(1, 0), "1"},
		{New(0, 0), "0"},
		{New(9, -1), "0"},
		{New(1, -1), "0"},
		{New(-1, -1), "-1"},
		{New(-9, -1), "-1"},
		{New(-1, 0), "-1"},
		{New(-101, -2), "-2"},
		{New(-1999, -3), "-2"},
	}
	for _, test := range testsWithDecimals {
		assertFloor(test.input, MustFromString(test.expected))
	}
}

func TestShopspring_Ceil(t *testing.T) {
	assertCeil := func(input, expected Decimal) {
		if got := input.Ceil(); !got.Equal(expected) {
			t.Errorf("Decimal.Ceil(%s) = %s, want %s", input, got, expected)
		}
	}
	testsWithStrings := []struct {
		input    string
		expected string
	}{
		{"1.999", "2"},
		{"1", "1"},
		{"1.01", "2"},
		{"0", "0"},
		{"0.9", "1"},
		{"0.1", "1"},
		{"-0.9", "0"},
		{"-0.1", "0"},
		{"-1.00", "-1"},
		{"-1.01", "-1"},
		{"-1.999", "-1"},
	}
	for _, test := range testsWithStrings {
		assertCeil(MustFromString(test.input), MustFromString(test.expected))
	}

	testsWithDecimals := []struct {
		input    Decimal
		expected string
	}{
		{New(100, -1), "10"},
		{New(10, 0), "10"},
		{New(1, 1), "10"},
		{New(1999, -3), "2"},
		{New(101, -2), "2"},
		{New(1, 0), "1"},
		{New(0, 0), "0"},
		{New(9, -1), "1"},
		{New(1, -1), "1"},
		{New(-1, -1), "0"},
		{New(-9, -1), "0"},
		{New(-1, 0), "-1"},
		{New(-101, -2), "-1"},
		{New(-1999, -3), "-1"},
	}
	for _, test := range testsWithDecimals {
		assertCeil(test.input, MustFromString(test.expected))
	}
}

func TestShopspring_RoundAndStringFixed(t *testing.T) {
	type testData struct {
		input         string
		places        int32
		expected      string
		expectedFixed string
	}
	tests := []testData{
		{"1.454", 0, "1", ""},
		{"1.454", 1, "1.5", ""},
		{"1.454", 2, "1.45", ""},
		{"1.454", 3, "1.454", ""},
		{"1.454", 4, "1.454", "1.4540"},
		{"1.454", 5, "1.454", "1.45400"},
		{"1.554", 0, "2", ""},
		{"1.554", 1, "1.6", ""},
		{"1.554", 2, "1.55", ""},
		{"0.554", 0, "1", ""},
		{"0.454", 0, "0", ""},
		{"0.454", 5, "0.454", "0.45400"},
		{"0", 0, "0", ""},
		{"0", 1, "0", "0.0"},
		{"0", 2, "0", "0.00"},
		{"0", -1, "0", ""},
		{"5", 2, "5", "5.00"},
		{"5", 1, "5", "5.0"},
		{"5", 0, "5", ""},
		{"500", 2, "500", "500.00"},
		{"545", -1, "550", ""},
		{"545", -2, "500", ""},
		{"545", -3, "1000", ""},
		{"545", -4, "0", ""},
		{"499", -3, "0", ""},
		{"499", -4, "0", ""},
	}

	// Upstream's negative-number augmentation, verbatim.
	for _, test := range tests {
		expected := test.expected
		if expected != "0" {
			expected = "-" + expected
		}
		expectedStr := test.expectedFixed
		if strings.ContainsAny(expectedStr, "123456789") && expectedStr != "" {
			expectedStr = "-" + expectedStr
		}
		tests = append(tests,
			testData{"-" + test.input, test.places, expected, expectedStr})
	}

	for _, test := range tests {
		d := MustFromString(test.input)

		expected := MustFromString(test.expected)
		if got := d.Round(test.places); !got.Equal(expected) {
			t.Errorf("Decimal.Round(%s, %d) = %s, want %s", d, test.places, got, expected)
		}

		if test.expectedFixed == "" {
			test.expectedFixed = test.expected
		}
		if got := d.StringFixed(test.places); got != test.expectedFixed {
			t.Errorf("Decimal.StringFixed(%s, %d) = %s, want %s", d, test.places, got, test.expectedFixed)
		}
	}
}

func TestShopspring_RoundCeilAndStringFixed(t *testing.T) {
	tests := []struct {
		input         string
		places        int32
		expected      string
		expectedFixed string
	}{
		{"1.454", 0, "2", ""},
		{"1.454", 1, "1.5", ""},
		{"1.454", 2, "1.46", ""},
		{"1.454", 3, "1.454", ""},
		{"1.454", 4, "1.454", "1.4540"},
		{"1.454", 5, "1.454", "1.45400"},
		{"1.554", 0, "2", ""},
		{"1.554", 1, "1.6", ""},
		{"1.554", 2, "1.56", ""},
		{"0.554", 0, "1", ""},
		{"0.454", 0, "1", ""},
		{"0.454", 5, "0.454", "0.45400"},
		{"0", 0, "0", ""},
		{"0", 1, "0", "0.0"},
		{"0", 2, "0", "0.00"},
		{"0", -1, "0", ""},
		{"5", 2, "5", "5.00"},
		{"5", 1, "5", "5.0"},
		{"5", 0, "5", ""},
		{"500", 2, "500", "500.00"},
		{"500", -2, "500", ""},
		{"545", -1, "550", ""},
		{"545", -2, "600", ""},
		{"545", -3, "1000", ""},
		{"545", -4, "10000", ""},
		{"499", -3, "1000", ""},
		{"499", -4, "10000", ""},
		{"1.1001", 2, "1.11", ""},
		{"-1.1001", 2, "-1.10", ""},
		{"-1.454", 0, "-1", ""},
		{"-1.454", 1, "-1.4", ""},
		{"-1.454", 2, "-1.45", ""},
		{"-1.454", 3, "-1.454", ""},
		{"-1.454", 4, "-1.454", "-1.4540"},
		{"-1.454", 5, "-1.454", "-1.45400"},
		{"-1.554", 0, "-1", ""},
		{"-1.554", 1, "-1.5", ""},
		{"-1.554", 2, "-1.55", ""},
		{"-0.554", 0, "0", ""},
		{"-0.454", 0, "0", ""},
		{"-0.454", 5, "-0.454", "-0.45400"},
		{"-5", 2, "-5", "-5.00"},
		{"-5", 1, "-5", "-5.0"},
		{"-5", 0, "-5", ""},
		{"-500", 2, "-500", "-500.00"},
		{"-500", -2, "-500", ""},
		{"-545", -1, "-540", ""},
		{"-545", -2, "-500", ""},
		{"-545", -3, "0", ""},
		{"-545", -4, "0", ""},
		{"-499", -3, "0", ""},
		{"-499", -4, "0", ""},
	}

	for _, test := range tests {
		d := MustFromString(test.input)

		expected := MustFromString(test.expected)
		got := d.RoundCeil(test.places)
		if !got.Equal(expected) {
			t.Errorf("Decimal.RoundCeil(%s, %d) = %s, want %s", d, test.places, got, expected)
		}

		if test.expectedFixed == "" {
			test.expectedFixed = test.expected
		}
		if gotStr := got.StringFixed(test.places); gotStr != test.expectedFixed {
			t.Errorf("Decimal.StringFixed(%s, %d) = %s, want %s", got, test.places, gotStr, test.expectedFixed)
		}
	}
}

func TestShopspring_RoundFloorAndStringFixed(t *testing.T) {
	tests := []struct {
		input         string
		places        int32
		expected      string
		expectedFixed string
	}{
		{"1.454", 0, "1", ""},
		{"1.454", 1, "1.4", ""},
		{"1.454", 2, "1.45", ""},
		{"1.454", 3, "1.454", ""},
		{"1.454", 4, "1.454", "1.4540"},
		{"1.454", 5, "1.454", "1.45400"},
		{"1.554", 0, "1", ""},
		{"1.554", 1, "1.5", ""},
		{"1.554", 2, "1.55", ""},
		{"0.554", 0, "0", ""},
		{"0.454", 0, "0", ""},
		{"0.454", 5, "0.454", "0.45400"},
		{"0", 0, "0", ""},
		{"0", 1, "0", "0.0"},
		{"0", 2, "0", "0.00"},
		{"0", -1, "0", ""},
		{"5", 2, "5", "5.00"},
		{"5", 1, "5", "5.0"},
		{"5", 0, "5", ""},
		{"500", 2, "500", "500.00"},
		{"500", -2, "500", ""},
		{"545", -1, "540", ""},
		{"545", -2, "500", ""},
		{"545", -3, "0", ""},
		{"545", -4, "0", ""},
		{"499", -3, "0", ""},
		{"499", -4, "0", ""},
		{"1.1001", 2, "1.10", ""},
		{"-1.1001", 2, "-1.11", ""},
		{"-1.454", 0, "-2", ""},
		{"-1.454", 1, "-1.5", ""},
		{"-1.454", 2, "-1.46", ""},
		{"-1.454", 3, "-1.454", ""},
		{"-1.454", 4, "-1.454", "-1.4540"},
		{"-1.454", 5, "-1.454", "-1.45400"},
		{"-1.554", 0, "-2", ""},
		{"-1.554", 1, "-1.6", ""},
		{"-1.554", 2, "-1.56", ""},
		{"-0.554", 0, "-1", ""},
		{"-0.454", 0, "-1", ""},
		{"-0.454", 5, "-0.454", "-0.45400"},
		{"-5", 2, "-5", "-5.00"},
		{"-5", 1, "-5", "-5.0"},
		{"-5", 0, "-5", ""},
		{"-500", 2, "-500", "-500.00"},
		{"-500", -2, "-500", ""},
		{"-545", -1, "-550", ""},
		{"-545", -2, "-600", ""},
		{"-545", -3, "-1000", ""},
		{"-545", -4, "-10000", ""},
		{"-499", -3, "-1000", ""},
		{"-499", -4, "-10000", ""},
	}

	for _, test := range tests {
		d := MustFromString(test.input)

		expected := MustFromString(test.expected)
		got := d.RoundFloor(test.places)
		if !got.Equal(expected) {
			t.Errorf("Decimal.RoundFloor(%s, %d) = %s, want %s", d, test.places, got, expected)
		}

		if test.expectedFixed == "" {
			test.expectedFixed = test.expected
		}
		if gotStr := got.StringFixed(test.places); gotStr != test.expectedFixed {
			t.Errorf("Decimal.StringFixed(%s, %d) = %s, want %s", got, test.places, gotStr, test.expectedFixed)
		}
	}
}

func TestShopspring_TruncateNegativePrecision(t *testing.T) {
	// Divergence: upstream also pins each result's exponent, an artifact of
	// its unnormalized representation; only the value ports.
	tests := []struct {
		input  string
		places int32
		want   string
	}{
		{"5432", -2, "5400"},
		{"-5432", -2, "-5400"},
		{"5499", -2, "5400"},
		{"5500", -2, "5500"},
		{"5501", -2, "5500"},
		{"999", -3, "0"},
		{"1000", -3, "1000"},
		{"1001", -3, "1000"},
		{"-999", -3, "0"},
		{"-1000", -3, "-1000"},
		{"123.456", 2, "123.45"},
		{"123.456", 0, "123"},
		{"123.456", 5, "123.456"},
	}
	for _, tc := range tests {
		if got := MustFromString(tc.input).Truncate(tc.places).String(); got != tc.want {
			t.Errorf("Decimal.Truncate(%s, %d) = %s, want %s", tc.input, tc.places, got, tc.want)
		}
	}
}

func TestShopspring_RescaleToZeroRepresentation(t *testing.T) {
	// Divergence: upstream compares big.Int internals; our zero has
	// one representation, so == Zero carries the check. RoundDown
	// is outside the shared surface, and 5e-20 and 5e-40 lie
	// below our smallest nonzero, so construction itself panics.
	for _, k := range []int32{1, 19} {
		if got := New(5, -k).Truncate(0); got != Zero {
			t.Errorf("Decimal.Truncate(5e-%d, 0) = %#v, want Zero", k, got)
		}
	}
	for _, k := range []int32{20, 40} {
		if !didPanic(func() { New(5, -k) }) {
			t.Errorf("New(5, -%d) did not panic, want ErrRange panic", k)
		}
	}
}

func TestShopspring_RoundCeilFloorToZeroRepresentation(t *testing.T) {
	// Divergence: as in TestShopspring_RescaleToZeroRepresentation.
	for _, k := range []int32{1, 19} {
		if got := New(-5, -k).RoundCeil(0); got != Zero {
			t.Errorf("Decimal.RoundCeil(-5e-%d, 0) = %#v, want Zero", k, got)
		}
		if got := New(5, -k).RoundFloor(0); got != Zero {
			t.Errorf("Decimal.RoundFloor(5e-%d, 0) = %#v, want Zero", k, got)
		}
	}
	for _, k := range []int32{20, 40} {
		if !didPanic(func() { New(5, -k) }) {
			t.Errorf("New(5, -%d) did not panic, want ErrRange panic", k)
		}
	}
}

func TestShopspring_RoundingExponentNormalization(t *testing.T) {
	// Divergence: upstream pins the exponent each rounding
	// leaves; ours is normalized, so only the value ports.
	// RoundUp and RoundDown are outside the shared surface.
	tests := []struct {
		name   string
		round  func(Decimal, int32) Decimal
		input  string
		places int32
		want   string
	}{
		{"RoundFloor", Decimal.RoundFloor, "100.0", 0, "100"},
		{"RoundFloor", Decimal.RoundFloor, "100.00", 0, "100"},
		{"RoundFloor", Decimal.RoundFloor, "-100.0", 0, "-100"},
		{"RoundFloor", Decimal.RoundFloor, "500", -2, "500"},
		{"RoundFloor", Decimal.RoundFloor, "-100", 2, "-100"},
		{"RoundFloor", Decimal.RoundFloor, "5E3", -2, "5000"},
		{"RoundCeil", Decimal.RoundCeil, "100.0", 0, "100"},
		{"RoundCeil", Decimal.RoundCeil, "100.00", 0, "100"},
		{"RoundCeil", Decimal.RoundCeil, "-100.0", 0, "-100"},
		{"RoundCeil", Decimal.RoundCeil, "500", -2, "500"},
		{"RoundCeil", Decimal.RoundCeil, "-100", 2, "-100"},
		{"RoundCeil", Decimal.RoundCeil, "5E3", -2, "5000"},
	}
	for _, tc := range tests {
		if got := tc.round(MustFromString(tc.input), tc.places).String(); got != tc.want {
			t.Errorf("Decimal.%s(%s, %d) = %s, want %s", tc.name, tc.input, tc.places, got, tc.want)
		}
	}
}

func TestShopspring_Uninitialized(t *testing.T) {
	// Upstream's zero-value tour, restricted to the shared surface.
	a := Decimal{}
	b := Decimal{}

	decs := []struct {
		name string
		d    Decimal
	}{
		{"zero value", a},
		{"Abs", a.Abs()},
		{"Add", a.Add(b)},
		{"Sub", a.Sub(b)},
		{"Mul", a.Mul(b)},
		{"Div", a.Div(New(1, -1))},
		{"Round", a.Round(2)},
		{"Floor", a.Floor()},
		{"Ceil", a.Ceil()},
		{"Truncate", a.Truncate(2)},
	}
	for _, tt := range decs {
		if got := tt.d.String(); got != "0" {
			t.Errorf("zero-value Decimal.%s String() = %s, want 0", tt.name, got)
		}
		if got := tt.d.StringFixed(3); got != "0.000" {
			t.Errorf("zero-value Decimal.%s StringFixed(3) = %s, want 0.000", tt.name, got)
		}
	}

	if got := a.Cmp(b); got != 0 {
		t.Errorf("zero-value Decimal.Cmp(zero) = %d, want 0", got)
	}
	if got := a.Sign(); got != 0 {
		t.Errorf("zero-value Decimal.Sign() = %d, want 0", got)
	}
	if got := a.Exponent(); got != 0 {
		t.Errorf("zero-value Decimal.Exponent() = %d, want 0", got)
	}
	if got := a.InexactFloat64(); got != 0 {
		t.Errorf("zero-value Decimal.InexactFloat64() = %v, want 0", got)
	}
}

func TestShopspring_Add(t *testing.T) {
	type Inp struct {
		a string
		b string
	}
	inputs := map[Inp]string{
		{"2", "3"}:                     "5",
		{"2454495034", "3451204593"}:   "5905699627",
		{"24544.95034", ".3451204593"}: "24545.2954604593",
		{".1", ".1"}:                   "0.2",
		{".1", "-.1"}:                  "0",
		{"0", "1.001"}:                 "1.001",
	}
	for inp, res := range inputs {
		if got := MustFromString(inp.a).Add(MustFromString(inp.b)).String(); got != res {
			t.Errorf("Decimal.Add(%s, %s) = %s, want %s", inp.a, inp.b, got, res)
		}
	}
}

func TestShopspring_Sub(t *testing.T) {
	type Inp struct {
		a string
		b string
	}
	inputs := map[Inp]string{
		{"2", "3"}:                     "-1",
		{"12", "3"}:                    "9",
		{"-2", "9"}:                    "-11",
		{"2454495034", "3451204593"}:   "-996709559",
		{"24544.95034", ".3451204593"}: "24544.6052195407",
		{".1", "-.1"}:                  "0.2",
		{".1", ".1"}:                   "0",
		{"0", "1.001"}:                 "-1.001",
		{"1.001", "0"}:                 "1.001",
		{"2.3", ".3"}:                  "2",
	}
	for inp, res := range inputs {
		if got := MustFromString(inp.a).Sub(MustFromString(inp.b)).String(); got != res {
			t.Errorf("Decimal.Sub(%s, %s) = %s, want %s", inp.a, inp.b, got, res)
		}
	}
}

func TestShopspring_Neg(t *testing.T) {
	inputs := map[string]string{
		"0":     "0",
		"10":    "-10",
		"5.56":  "-5.56",
		"-10":   "10",
		"-5.56": "5.56",
	}
	for inp, res := range inputs {
		if got := MustFromString(inp).Neg().String(); got != res {
			t.Errorf("Decimal.Neg(%s) = %s, want %s", inp, got, res)
		}
	}
}

func TestShopspring_NegFromEmpty(t *testing.T) {
	if got := (Decimal{}).Neg().String(); got != "0" {
		t.Errorf("zero-value Decimal.Neg() = %s, want 0", got)
	}
}

func TestShopspring_Mul(t *testing.T) {
	type Inp struct {
		a string
		b string
	}
	inputs := map[Inp]string{
		{"2", "3"}:                     "6",
		{"2454495034", "3451204593"}:   "8470964534836491162",
		{"24544.95034", ".3451204593"}: "8470.964534836491162",
		{".1", ".1"}:                   "0.01",
		{"0", "1.001"}:                 "0",
	}
	for inp, res := range inputs {
		if got := MustFromString(inp.a).Mul(MustFromString(inp.b)).String(); got != res {
			t.Errorf("Decimal.Mul(%s, %s) = %s, want %s", inp.a, inp.b, got, res)
		}
	}

	// positive scale
	if got := New(1234, 5).Mul(New(45, -1)).String(); got != "555300000" {
		t.Errorf("Decimal.Mul(New(1234, 5), New(45, -1)) = %s, want 555300000", got)
	}
}

func TestShopspring_Div(t *testing.T) {
	type Inp struct {
		a string
		b string
	}
	inputs := map[Inp]string{
		{"6", "3"}:                            "2",
		{"10", "2"}:                           "5",
		{"2.2", "1.1"}:                        "2",
		{"-2.2", "-1.1"}:                      "2",
		{"12.88", "5.6"}:                      "2.3",
		{"1023427554493", "43432632"}:         "23563.5628642767953828", // rounded
		{"1", "434324545566634"}:              "0.0000000000000023",
		{"1", "3"}:                            "0.3333333333333333",
		{"2", "3"}:                            "0.6666666666666667", // rounded
		{"10000", "3"}:                        "3333.3333333333333333",
		{"10234274355545544493", "-3"}:        "-3411424785181848164.3333333333333333",
		{"-4612301402398.4753343454", "23.5"}: "-196268144782.9138440146978723",
	}
	for inp, expectedStr := range inputs {
		num, denom := MustFromString(inp.a), MustFromString(inp.b)
		got := num.Div(denom)
		if expected := MustFromString(expectedStr); !got.Equal(expected) {
			t.Errorf("Decimal.Div(%s, %s) = %s, want %s", num, denom, got, expected)
		}
	}

	type Inp2 struct {
		n    int64
		exp  int32
		n2   int64
		exp2 int32
	}
	// Upstream's positive-exponent grid with all four sign permutations.
	inputs2 := map[Inp2]string{
		{124, 10, 3, 1}: "41333333333.3333333333333333",
		{124, 10, 3, 0}: "413333333333.3333333333333333",
		{124, 10, 6, 1}: "20666666666.6666666666666667",
		{124, 10, 6, 0}: "206666666666.6666666666666667",
		{10, 10, 10, 1}: "1000000000",
	}
	for inp, expectedAbs := range inputs2 {
		for i := -1; i <= 1; i += 2 {
			for j := -1; j <= 1; j += 2 {
				num := New(inp.n*int64(i), inp.exp)
				denom := New(inp.n2*int64(j), inp.exp2)
				expected := expectedAbs
				if i != j {
					expected = "-" + expectedAbs
				}
				if got := num.Div(denom).String(); got != expected {
					t.Errorf("Decimal.Div(%s, %s) = %s, want %s", num, denom, got, expected)
				}
			}
		}
	}
}

func TestShopspring_Overflow(t *testing.T) {
	// Divergence: upstream constructs New(1, math.MinInt32) fine and
	// panics only inside Mul. Our bounds reject the value at construction,
	// so New itself is the panic site — earlier, same loudness.
	if !didPanic(func() { New(1, math.MinInt32) }) {
		t.Fatalf("New(1, math.MinInt32) did not panic")
	}
	if !didPanic(func() { New(1, math.MaxInt32) }) {
		t.Fatalf("New(1, math.MaxInt32) did not panic")
	}
}

func TestShopspring_ExtremeValues(t *testing.T) {
	// Divergence: upstream floors/ceils New(123, math.MinInt32) under
	// a one-second watchdog. That value sits far below our smallest
	// nonzero, so construction itself panics with ErrRange instead.
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, ErrRange) {
			t.Errorf("New(123, math.MinInt32) panic = %v, want ErrRange", err)
		}
	}()
	New(123, math.MinInt32)
}

func TestShopspring_OutOfRange(t *testing.T) {
	// Divergence: upstream's arbitrary precision holds these testTable rows;
	// they exceed our 128-bit coefficient or nineteen-digit fraction, so
	// NewFromFloat panics with ErrRange and parsing rejects.
	for _, f := range []float64{1e250, 2e250, -1e250, -2e250, 1.29067116156722e-309} {
		if !didPanic(func() { NewFromFloat(f) }) {
			t.Errorf("NewFromFloat(%v) did not panic, want ErrRange panic", f)
		}
	}
	huge := "1" + strings.Repeat("0", 250)
	if _, err := NewFromString(huge); !errors.Is(err, ErrRange) {
		t.Errorf("NewFromString(1 with 250 zeros) = %v, want ErrRange", err)
	}
}

func TestShopspring_Min(t *testing.T) {
	// first element in each row is the expected answer, rest are inputs
	testCases := [][]float64{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1, 1, 2},
		{-2, 1, 2, -2},
		{-3, 0, 2, -2, -3},
	}
	for _, test := range testCases {
		expected, input := test[0], test[1:]
		expectedDecimal := NewFromFloat(expected)
		decimalInput := []Decimal{}
		for _, inp := range input {
			decimalInput = append(decimalInput, NewFromFloat(inp))
		}
		got := Min(decimalInput[0], decimalInput[1:]...)
		if !got.Equal(expectedDecimal) {
			t.Errorf("Min(%v) = %s, want %s", decimalInput, got, expectedDecimal)
		}
	}
}

func TestShopspring_Max(t *testing.T) {
	// first element in each row is the expected answer, rest are inputs
	testCases := [][]float64{
		{0, 0},
		{1, 1},
		{-1, -1},
		{2, 1, 2},
		{2, 1, 2, -2},
		{3, 0, 3, -2},
		{-2, -3, -2},
	}
	for _, test := range testCases {
		expected, input := test[0], test[1:]
		expectedDecimal := NewFromFloat(expected)
		decimalInput := []Decimal{}
		for _, inp := range input {
			decimalInput = append(decimalInput, NewFromFloat(inp))
		}
		got := Max(decimalInput[0], decimalInput[1:]...)
		if !got.Equal(expectedDecimal) {
			t.Errorf("Max(%v) = %s, want %s", decimalInput, got, expectedDecimal)
		}
	}
}

func TestShopspring_Abs1(t *testing.T) {
	a := New(-1234, -4)
	b := New(1234, -4)
	if got := a.Abs(); got.Cmp(b) != 0 {
		t.Errorf("Decimal.Abs(%s) = %s, want %s", a, got, b)
	}
}

func TestShopspring_Abs2(t *testing.T) {
	a := New(-1234, -4)
	b := New(1234, -4)
	if got := b.Abs(); got.Cmp(a) == 0 {
		t.Errorf("Decimal.Abs(%s) = %s, want it to differ from %s", b, got, a)
	}
}

func TestShopspring_Equalities(t *testing.T) {
	// Upstream's Equals alias is deprecated and outside
	// our subset; its one assertion is dropped.
	a := New(1234, 3)
	b := New(1234, 3)
	c := New(1234, 4)

	if !a.Equal(b) {
		t.Errorf("Decimal.Equal(%s, %s) = false, want true", a, b)
	}
	if a.Equal(c) {
		t.Errorf("Decimal.Equal(%s, %s) = true, want false", a, c)
	}

	if !c.GreaterThan(b) {
		t.Errorf("Decimal.GreaterThan(%s, %s) = false, want true", c, b)
	}
	if b.GreaterThan(c) {
		t.Errorf("Decimal.GreaterThan(%s, %s) = true, want false", b, c)
	}
	if !a.GreaterThanOrEqual(b) {
		t.Errorf("Decimal.GreaterThanOrEqual(%s, %s) = false, want true", a, b)
	}
	if !c.GreaterThanOrEqual(b) {
		t.Errorf("Decimal.GreaterThanOrEqual(%s, %s) = false, want true", c, b)
	}
	if b.GreaterThanOrEqual(c) {
		t.Errorf("Decimal.GreaterThanOrEqual(%s, %s) = true, want false", b, c)
	}
	if !b.LessThan(c) {
		t.Errorf("Decimal.LessThan(%s, %s) = false, want true", b, c)
	}
	if c.LessThan(b) {
		t.Errorf("Decimal.LessThan(%s, %s) = true, want false", c, b)
	}
	if !a.LessThanOrEqual(b) {
		t.Errorf("Decimal.LessThanOrEqual(%s, %s) = false, want true", a, b)
	}
	if !b.LessThanOrEqual(c) {
		t.Errorf("Decimal.LessThanOrEqual(%s, %s) = false, want true", b, c)
	}
	if c.LessThanOrEqual(b) {
		t.Errorf("Decimal.LessThanOrEqual(%s, %s) = true, want false", c, b)
	}
}

func TestShopspring_ScalesNotEqual(t *testing.T) {
	a := New(1234, 2)
	b := New(1234, 3)
	if a.Equal(b) {
		t.Errorf("Decimal.Equal(%s, %s) = true, want false", a, b)
	}
}

func TestShopspring_Cmp1(t *testing.T) {
	a := New(123, 3)
	b := New(-1234, 2)
	if got := a.Cmp(b); got != 1 {
		t.Errorf("Decimal.Cmp(%s, %s) = %d, want 1", a, b, got)
	}
}

func TestShopspring_Cmp2(t *testing.T) {
	a := New(123, 3)
	b := New(1234, 2)
	if got := a.Cmp(b); got != -1 {
		t.Errorf("Decimal.Cmp(%s, %s) = %d, want -1", a, b, got)
	}
}

func TestShopspring_Sign(t *testing.T) {
	if got := Zero.Sign(); got != 0 {
		t.Errorf("Decimal.Sign(0) = %d, want 0", got)
	}
	if got := New(1, 0).Sign(); got != 1 {
		t.Errorf("Decimal.Sign(1) = %d, want 1", got)
	}
	if got := New(-1, 0).Sign(); got != -1 {
		t.Errorf("Decimal.Sign(-1) = %d, want -1", got)
	}
}

func TestShopspring_Sum(t *testing.T) {
	vals := make([]Decimal, 10)
	var i = int64(0)
	for key := range vals {
		vals[key] = New(i, 0)
		i++
	}
	sum := Sum(vals[0], vals[1:]...)
	if !sum.Equal(New(45, 0)) {
		t.Errorf("Sum(0..9) = %s, want 45", sum)
	}

	// Divergence: upstream compares big.Int internals;
	// our zero has one representation, so == Zero carries
	// the check. Its Shift rows are outside the shared surface.
	if got := Sum(New(5, -1), New(-5, -1), Zero); got != Zero {
		t.Errorf("Sum(0.5, -0.5, 0) = %#v, want Zero", got)
	}
}

func TestShopspring_String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.22", "1.22"},
		{"1.00", "1"},
		{"153.192", "153.192"},
		{"999.999", "999.999"},
		{"0.0000000001", "0.0000000001"},
		{"0.0000000000", "0"},
	}
	for _, tc := range tests {
		if got := MustFromString(tc.input).String(); got != tc.want {
			t.Errorf("Decimal.String(%s) = %s, want %s", tc.input, got, tc.want)
		}
	}
}

func TestShopspring_JSONOmitZeroTag(t *testing.T) {
	// Upstream's go1.24-gated decimal_go124_test.go; our go.mod is past it.
	type Nested struct {
		Amount Decimal `json:"amount,omitzero"`
	}
	type Parent struct {
		Nested Nested `json:"nested,omitzero"`
	}
	tests := []struct {
		name   string
		parent Parent
		want   string
	}{
		{"Decimal{} empty value", Parent{Nested: Nested{Amount: Decimal{}}}, "{}"},
		{"Zero constant", Parent{Nested: Nested{Amount: Zero}}, "{}"},
	}
	for _, tc := range tests {
		b, err := json.Marshal(tc.parent)
		if err != nil {
			t.Fatalf("json.Marshal(%s) = %v", tc.name, err)
		}
		if string(b) != tc.want {
			t.Errorf("json.Marshal(%s) = %s, want %s", tc.name, b, tc.want)
		}
	}
}

func TestShopspring_Binary(t *testing.T) {
	// The wire layout is ours by design, so upstream's round-trip is what
	// ports: every table value must survive MarshalBinary intact.
	for _, x := range shopspringTable {
		d := MustFromString(x.short)
		b, err := d.MarshalBinary()
		if err != nil {
			t.Errorf("Decimal.MarshalBinary(%s) = %v", d, err)
		}
		var d2 Decimal
		if err := d2.UnmarshalBinary(b); err != nil {
			t.Errorf("Decimal.UnmarshalBinary(% x) = %v", b, err)
		}
		if !d2.Equal(d) {
			t.Errorf("Decimal binary round-trip of %s = %s", d, d2)
		}
	}
}
