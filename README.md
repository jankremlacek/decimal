# decimal

[![Go Reference](https://pkg.go.dev/badge/github.com/jankremlacek/decimal.svg)](https://pkg.go.dev/github.com/jankremlacek/decimal)

A fixed-point decimal for Go, built for financial values and as a
mostly drop-in replacement for [shopspring/decimal](https://github.com/shopspring/decimal).

A `Decimal` is a 128-bit coefficient over a power of ten, held by value: no
pointers, no allocation, nothing for the garbage collector to scan. Every value
is normalized, so `==` is value equality and a `Decimal` is a valid map key.

## Why

shopspring/decimal is exact and battle-tested, but every value wraps a
`*big.Int`: parsing, adding or multiplying allocates on the heap, and every
live value is a pointer the garbage collector has to chase. On a hot path, such
as a crypto market handling a stream of prices and sizes, that is allocation
churn and GC work on every message.

This package trades unbounded precision for a fixed 24-byte value type that
never touches `math/big`:

- **No heap allocation** to parse, compare, round, add, subtract, multiply or
  divide, or to render with `Append`. `String` and the `Marshal*` methods
  allocate once, for the result they return; error paths allocate their error.
- **Nothing for the GC to scan.** A `Decimal` holds no pointers, so it can sit
  in a local, a struct field or a million-element slice, and the collector
  skips it.
- **Wide enough for crypto:** 39 significant digits and up to 19 fraction
  digits, so wei-denominated amounts (10⁻¹⁸) fit exactly.

On a per-trade workload (parse price and size, compute notional and fee,
accumulate both, render the notional) it runs **1.5× faster with 0 allocations
against shopspring's 14**. With a million values live on the heap, a full GC
cycle costs **18× less** and the live heap is less than half the size. See
[Performance](#performance).

```bash
go get github.com/jankremlacek/decimal
```

Supports the two latest Go releases, currently 1.26 and 1.27.
The module has no runtime dependencies.

## Usage

```go
package main

import (
  "fmt"

  "github.com/jankremlacek/decimal"
)

func main() {
  price := decimal.MustFromString("64500.4")
  qty := decimal.New(125, -2) // 1.25

  total := price.Mul(qty)
  fmt.Println(total)                // 80625.5
  fmt.Println(total.StringFixed(2)) // 80625.50

  share := total.Div(decimal.NewFromInt(3))
  fmt.Println(share) // 26875.1666666666666667

  d, err := decimal.NewFromString("1.5e-8")
  if err != nil {
    panic(err)
  }
  fmt.Println(d, d.Cmp(decimal.Zero)) // 0.000000015 1
}
```

## Model

**Range.** The representable set is exactly `±coefficient / 10^prec`, where
`coefficient` is at most 2¹²⁸−1 and `prec` is at most 19 (`MaxFractionDigits`;
wei, 10⁻¹⁸, fits with a digit to spare). Both bounds bind together, so magnitude
and precision trade against each other:

| Shape                   | Largest magnitude                            |
| ----------------------- | -------------------------------------------- |
| no fraction             | `340282366920938463463374607431768211455`    |
| full 19-digit fraction  | `34028236692093846346.3374607431768211455`   |
| smallest nonzero        | `0.0000000000000000001`                      |

A result that does not fit is an error or a panic, never a silent loss and never
a slow fallback; the package does not use `math/big`.

**Arithmetic is exact-or-panic.** `Add`, `Sub` and `Mul` return the exact result
or panic with `ErrRange`. Only `Div` rounds: every quotient rounds half away from
zero at sixteen fraction digits, exactly as shopspring does, so a quotient that
fits sixteen digits is exact. A zero divisor panics with `ErrDivByZero`.

**Untrusted input returns errors.** The constructors (`New`, `NewFromFloat`,
`MustFromString`) panic on out-of-range input, which suits literals and
constants. `NewFromString`, the text/JSON/binary/gob decoders and `sql.Scanner`
return an error instead and never panic.

**Immutability.** Every method works on a copy and never mutates its receiver or
arguments. The decoders are the deliberate exception: on successful non-null
input they replace the receiver with a freshly parsed canonical value.

## Serialization

| Form          | Encoding                                                                                       |
| ------------- | ---------------------------------------------------------------------------------------------- |
| `String`/text | Canonical decimal text: no exponent, no trailing fraction zeros. `Append` renders allocation-free. |
| JSON          | Canonical text, **quoted** — a reader that parses a bare number into `float64` loses precision. Decoding accepts quoted and bare numbers; `null` leaves the receiver untouched. |
| Binary, gob   | Compact layout: one header byte (precision plus sign bit), then the coefficient as minimal big-endian bytes — 1 byte for zero, 17 at full width. Callers that persist these bytes own their versioning. |
| SQL           | `Value` writes canonical text; `Scan` accepts text, `[]byte`, `int64` and `float64`. A column delivered as text round-trips exactly; one delivered as `float64` carries only what that float held. Postgres `NUMERIC` is far wider than the representable set, so a value outside it fails the scan. |

Decoded values are normalized on the way in, so they are canonical.

## Differences from shopspring/decimal

The parser holds shopspring's accept/reject line, and division rounds the way
shopspring's does; the [differential fuzzers](fuzz_test.go) enforce both. Where
behaviour differs, it is on purpose:

- **Bounded range.** shopspring is arbitrary precision; values here must fit the
  range above. Out-of-range results panic (arithmetic) or return `ErrRange`
  (parsing and decoding).
- **Exponent cap.** shopspring rejects an effective exponent past its
  `MaxDecodeExponent`. This parser's cost depends only on the text length, so
  it accepts e.g. a zero or an in-range value with a long zero tail past that cap.
- **JSON** is always quoted; there is no `MarshalJSONWithoutQuotes` switch.
- **Smaller API.** Available: `New`, `NewFromInt`, `NewFromFloat`,
  `NewFromString`, `MustFromString` (shopspring's `RequireFromString`), `Zero`,
  `Min`, `Max`, `Sum`; methods `Add`, `Sub`, `Mul`, `Div`, `Neg`, `Abs`, `Cmp`,
  `Equal`, `GreaterThan[OrEqual]`, `LessThan[OrEqual]`, `Sign`, `IsZero`,
  `IsPositive`, `IsNegative`, `IsInteger`, `Exponent`, `Round`, `RoundFloor`,
  `RoundCeil`, `RoundFloorSigFigs`, `Truncate`, `Floor`, `Ceil`, `String`,
  `StringFixed`, `Append`, `TextLen`, `InexactFloat64`, and the text, JSON,
  binary, gob and SQL interfaces. Not available: `Float64`, `NewFromFloat32`,
  `DivRound`, `QuoRem`, `Shift`, `Equals`, `IntPart`, `BigInt`, `Rat`, `Pow`,
  `Sqrt`, `Format`, `NullDecimal` and similar.

[`shopspring_test.go`](shopspring_test.go) ports shopspring v1.6.0's own test
fixtures for every method both packages share; each divergence is marked there.

## Performance

`go test -bench VsShopspring -benchmem` runs both packages side by side.
Measured on an Apple M4 Max with Go 1.27.

**Workload and GC**

| Benchmark                                 | decimal                     | shopspring                       |
| ----------------------------------------- | --------------------------- | -------------------------------- |
| `Fill`: one trade, end to end             | 139 ns, 0 allocs            | 205 ns, 14 allocs (416 B)        |
| `GC`: one full cycle, 1M live values      | 0.13 ms, 23 MiB live        | 2.4 ms, 54 MiB live              |

`Fill` renders with the allocation-free `Append`; shopspring has no equivalent,
so it renders with `String`.

**Single operations** (ns/op, allocs/op)

| Operation                               | decimal      | shopspring     | Speed          |
| --------------------------------------- | ------------ | -------------- | -------------- |
| `NewFromString("64500.4")`              | 14.3, 0      | 28.3, 2        | 2.0× faster    |
| `NewFromString` wei (`0.000…001`)       | 15.5, 0      | 166, 4         | 10.7× faster   |
| `NewFromString` 39 digits               | 75.5, 0      | 274, 5         | 3.6× faster    |
| `String` price                          | 17.0, 1      | 60.0, 2        | 3.5× faster    |
| `MarshalJSON` price                     | 18.3, 1      | 64.2, 3        | 3.5× faster    |
| `Add` same precision                    | 15.9, 0      | 19.3, 2        | 1.2× faster    |
| `Add` across precisions                 | 22.9, 0      | 28.7, 2        | 1.3× faster    |
| `Cmp` across precisions                 | 7.8, 0       | 18.3, 1        | 2.3× faster    |
| `Mul` price × size                      | 30.8, 0      | 20.7, 2        | 1.5× slower    |
| `Div` same precision                    | 233, 0       | 47.4, 4        | 4.9× slower    |
| `Div` 38 by 29 digits                   | 1101, 0      | 157, 8         | 7.0× slower    |

`Mul` and `Div` are the exceptions. They compute in portable 256-bit
arithmetic, where shopspring uses `math/big`'s assembly. Both stay
allocation-free, so in a running service, where each shopspring allocation
also feeds the next GC cycle, the gap is narrower than the single-operation
numbers show.

Speed and zero allocation are the package's reason to exist, but nothing in
the test suite enforces them. Check the benchmarks when you touch a hot path.

## Contributing

- `make vet test` passes, and statement coverage stays at 100%.
- Immutability, serialization round-trips and shopspring parity have no compiler
  support. Each is held by a table test; a change that touches one extends its
  table.
- A behaviour change keeps both differential fuzzers green under `make test-fuzz`.
  An ordinary test run only replays the pinned seeds in `testdata/fuzz`; it does
  not fuzz. A fuzzer-found defect is fixed and its input pinned as a regression
  seed.
- Values are built only through the constructors and internal normalizers,
  never a raw struct literal, so the canonical form holds.
- Production code must not import shopspring/decimal; it is a test-only oracle.

Run `make help` for the available targets.

## License

[MIT](LICENSE). `shopspring_test.go` ports test fixtures from
[shopspring/decimal](https://github.com/shopspring/decimal), used under its MIT
License; the notice is kept in that file.
