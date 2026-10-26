package decimal

import "testing"

// benchTexts spans the common shapes: small integers, prices,
// wei-scale amounts, and the widest representable value.
var benchTexts = []struct {
	name string
	text string
}{
	{name: "int", text: "42"},
	{name: "price", text: "64500.4"},
	{name: "wei", text: "0.000000000000000001"},
	{name: "full", text: "34028236692093846346.3374607431768211455"},
}

// benchPairs drives the arithmetic benchmarks; wide exercises the binary
// long-division fallback of twenty-plus digit divisor coefficients.
var benchPairs = []struct {
	name string
	a    string
	b    string
}{
	{name: "same_prec", a: "64500.4", b: "0.1"},
	{name: "cross_prec", a: "64500.4", b: "0.000000000000000001"},
	{name: "wide", a: "12345678901234567890123456789012345678", b: "98765432109876543210987654321"},
}

var (
	sinkDecimal Decimal
	sinkBytes   []byte
	sinkString  string
	sinkInt     int
)

func BenchmarkNewFromString(b *testing.B) {
	for _, tt := range benchTexts {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				d, err := NewFromString(tt.text)
				if err != nil {
					b.Fatalf("NewFromString(%s) = %v", tt.text, err)
				}
				sinkDecimal = d
			}
		})
	}
}

func BenchmarkDecimal_String(b *testing.B) {
	for _, tt := range benchTexts {
		b.Run(tt.name, func(b *testing.B) {
			d := MustFromString(tt.text)
			b.ReportAllocs()
			for b.Loop() {
				sinkString = d.String()
			}
		})
	}
}

func BenchmarkDecimal_Append(b *testing.B) {
	// The allocation-free render path.
	for _, tt := range benchTexts {
		b.Run(tt.name, func(b *testing.B) {
			d := MustFromString(tt.text)
			buf := make([]byte, 0, MaxDigits+3)
			b.ReportAllocs()
			for b.Loop() {
				buf = d.Append(buf[:0])
			}
			sinkBytes = buf
		})
	}
}

func BenchmarkDecimal_MarshalJSON(b *testing.B) {
	for _, tt := range benchTexts {
		b.Run(tt.name, func(b *testing.B) {
			d := MustFromString(tt.text)
			b.ReportAllocs()
			for b.Loop() {
				j, err := d.MarshalJSON()
				if err != nil {
					b.Fatalf("Decimal.MarshalJSON(%s) = %v", tt.text, err)
				}
				sinkBytes = j
			}
		})
	}
}

func BenchmarkDecimal_UnmarshalJSON(b *testing.B) {
	for _, tt := range benchTexts {
		payload := []byte(`"` + tt.text + `"`)
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var d Decimal
				if err := d.UnmarshalJSON(payload); err != nil {
					b.Fatalf("Decimal.UnmarshalJSON(%s) = %v", payload, err)
				}
				sinkDecimal = d
			}
		})
	}
}

func BenchmarkDecimal_MarshalBinary(b *testing.B) {
	for _, tt := range benchTexts {
		b.Run(tt.name, func(b *testing.B) {
			d := MustFromString(tt.text)
			b.ReportAllocs()
			for b.Loop() {
				bin, err := d.MarshalBinary()
				if err != nil {
					b.Fatalf("Decimal.MarshalBinary(%s) = %v", tt.text, err)
				}
				sinkBytes = bin
			}
		})
	}
}

func BenchmarkDecimal_Add(b *testing.B) {
	for _, tt := range benchPairs {
		b.Run(tt.name, func(b *testing.B) {
			x, y := MustFromString(tt.a), MustFromString(tt.b)
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Add(y)
			}
		})
	}
}

func BenchmarkDecimal_Mul(b *testing.B) {
	// The full-width pair overflows our fraction cap,
	// so multiplication benches everyday shapes only.
	pairs := []struct {
		name string
		a    string
		b    string
	}{
		{name: "price_qty", a: "64500.4", b: "1.25"},
		{name: "rate", a: "0.0001", b: "1250000"},
	}
	for _, tt := range pairs {
		b.Run(tt.name, func(b *testing.B) {
			x, y := MustFromString(tt.a), MustFromString(tt.b)
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Mul(y)
			}
		})
	}
}

func BenchmarkDecimal_Div(b *testing.B) {
	for _, tt := range benchPairs {
		b.Run(tt.name, func(b *testing.B) {
			x, y := MustFromString(tt.a), MustFromString(tt.b)
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Div(y)
			}
		})
	}
}

func BenchmarkDecimal_Cmp(b *testing.B) {
	for _, tt := range benchPairs {
		b.Run(tt.name, func(b *testing.B) {
			x, y := MustFromString(tt.a), MustFromString(tt.b)
			b.ReportAllocs()
			for b.Loop() {
				sinkInt = x.Cmp(y)
			}
		})
	}
}
