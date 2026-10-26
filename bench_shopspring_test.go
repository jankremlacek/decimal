package decimal

import (
	"runtime"
	"testing"

	shopspring "github.com/shopspring/decimal"
)

// The BenchmarkVsShopspring_* benchmarks run the same
// inputs through this package and shopspring/decimal side
// by side, so one -bench run reads as a direct comparison.

var sinkShopspring shopspring.Decimal

func BenchmarkVsShopspring_NewFromString(b *testing.B) {
	for _, tt := range benchTexts {
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal, _ = NewFromString(tt.text)
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkShopspring, _ = shopspring.NewFromString(tt.text)
			}
		})
	}
}

func BenchmarkVsShopspring_String(b *testing.B) {
	for _, tt := range benchTexts {
		d, ref := MustFromString(tt.text), shopspring.RequireFromString(tt.text)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkString = d.String()
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkString = ref.String()
			}
		})
	}
}

func BenchmarkVsShopspring_MarshalJSON(b *testing.B) {
	for _, tt := range benchTexts {
		d, ref := MustFromString(tt.text), shopspring.RequireFromString(tt.text)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkBytes, _ = d.MarshalJSON()
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkBytes, _ = ref.MarshalJSON()
			}
		})
	}
}

func BenchmarkVsShopspring_Add(b *testing.B) {
	for _, tt := range benchPairs {
		x, y := MustFromString(tt.a), MustFromString(tt.b)
		rx, ry := shopspring.RequireFromString(tt.a), shopspring.RequireFromString(tt.b)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Add(y)
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkShopspring = rx.Add(ry)
			}
		})
	}
}

func BenchmarkVsShopspring_Mul(b *testing.B) {
	// The same everyday shapes as BenchmarkDecimal_Mul.
	pairs := []struct {
		name string
		a    string
		b    string
	}{
		{name: "price_qty", a: "64500.4", b: "1.25"},
		{name: "rate", a: "0.0001", b: "1250000"},
	}
	for _, tt := range pairs {
		x, y := MustFromString(tt.a), MustFromString(tt.b)
		rx, ry := shopspring.RequireFromString(tt.a), shopspring.RequireFromString(tt.b)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Mul(y)
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkShopspring = rx.Mul(ry)
			}
		})
	}
}

func BenchmarkVsShopspring_Div(b *testing.B) {
	for _, tt := range benchPairs {
		x, y := MustFromString(tt.a), MustFromString(tt.b)
		rx, ry := shopspring.RequireFromString(tt.a), shopspring.RequireFromString(tt.b)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkDecimal = x.Div(y)
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkShopspring = rx.Div(ry)
			}
		})
	}
}

func BenchmarkVsShopspring_Cmp(b *testing.B) {
	for _, tt := range benchPairs {
		x, y := MustFromString(tt.a), MustFromString(tt.b)
		rx, ry := shopspring.RequireFromString(tt.a), shopspring.RequireFromString(tt.b)
		b.Run(tt.name+"/ours", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkInt = x.Cmp(y)
			}
		})
		b.Run(tt.name+"/shopspring", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkInt = rx.Cmp(ry)
			}
		})
	}
}

// fill is a per-trade workload: parse the price and size off the wire,
// price the notional and its fee, accumulate both, and render the notional.
var fill = struct{ price, size, feeRate string }{"64500.4", "0.000125", "0.00025"}

func BenchmarkVsShopspring_Fill(b *testing.B) {
	b.Run("ours", func(b *testing.B) {
		rate := MustFromString(fill.feeRate)
		buf := make([]byte, 0, MaxDigits+3)
		var total, fees Decimal
		b.ReportAllocs()
		for b.Loop() {
			price, _ := NewFromString(fill.price)
			size, _ := NewFromString(fill.size)
			notional := price.Mul(size)
			total = total.Add(notional)
			fees = fees.Add(notional.Mul(rate))
			buf = notional.Append(buf[:0])
		}
		sinkDecimal, sinkBytes = total.Add(fees), buf
	})
	b.Run("shopspring", func(b *testing.B) {
		rate := shopspring.RequireFromString(fill.feeRate)
		var total, fees shopspring.Decimal
		b.ReportAllocs()
		for b.Loop() {
			price, _ := shopspring.NewFromString(fill.price)
			size, _ := shopspring.NewFromString(fill.size)
			notional := price.Mul(size)
			total = total.Add(notional)
			fees = fees.Add(notional.Mul(rate))
			sinkString = notional.String()
		}
		sinkShopspring = total.Add(fees)
	})
}

// BenchmarkVsShopspring_GC times one full collection with a million live
// values on the heap. Ours are pointer-free, so the collector skips them;
// each shopspring value points at a big.Int and its word slice.
func BenchmarkVsShopspring_GC(b *testing.B) {
	const n = 1_000_000
	b.Run("ours", func(b *testing.B) {
		vals := make([]Decimal, n)
		for i := range vals {
			vals[i] = New(int64(i)+645004, -1)
		}
		benchGC(b, func() { sinkDecimal = vals[n-1] })
	})
	b.Run("shopspring", func(b *testing.B) {
		vals := make([]shopspring.Decimal, n)
		for i := range vals {
			vals[i] = shopspring.New(int64(i)+645004, -1)
		}
		benchGC(b, func() { sinkShopspring = vals[n-1] })
	})
}

// benchGC reports the cost of runtime.GC and the live heap it walked;
// keepAlive must touch the values so they stay live through the loop.
func benchGC(b *testing.B, keepAlive func()) {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	for b.Loop() {
		runtime.GC()
	}
	keepAlive()
	b.ReportMetric(float64(ms.HeapAlloc)/(1<<20), "MiB-live")
}
