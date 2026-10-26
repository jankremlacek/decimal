package decimal

import (
	"errors"
	"math"
	"testing"
)

func TestDecimal_Value(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "fraction",
			v:    "64500.4",
			want: "64500.4",
		},
		{
			name: "negative normalizes",
			v:    "-1.500",
			want: "-1.5",
		},
		{
			name: "zero",
			v:    "0",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MustFromString(tt.v).Value()
			if err != nil {
				t.Fatalf("Decimal.Value(%s) = %v", tt.v, err)
			}
			if got != tt.want {
				t.Errorf("Decimal.Value(%s) = %v, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    string
		wantErr error
	}{
		{
			name:  "string",
			value: "64500.4",
			want:  "64500.4",
		},
		{
			name:  "quoted string",
			value: `"1.5"`,
			want:  "1.5",
		},
		{
			name:  "bytes",
			value: []byte("-0.10"),
			want:  "-0.1",
		},
		{
			name:  "quoted bytes",
			value: []byte(`"-2.5"`),
			want:  "-2.5",
		},
		{
			name:  "int64",
			value: int64(-42),
			want:  "-42",
		},
		{
			name:  "float64",
			value: 100.5,
			want:  "100.5",
		},
		{
			name:    "malformed text",
			value:   "nope",
			wantErr: ErrSyntax,
		},
		{
			name:    "out-of-range text",
			value:   "1e40",
			wantErr: ErrRange,
		},
		{
			name:    "NaN",
			value:   math.NaN(),
			wantErr: ErrSyntax,
		},
		{
			name:    "infinity",
			value:   math.Inf(1),
			wantErr: ErrSyntax,
		},
		{
			name:    "out-of-range float",
			value:   1e300,
			wantErr: ErrRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Decimal
			err := d.Scan(tt.value)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Decimal.Scan(%v) = %v, want %v", tt.value, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Decimal.Scan(%v) = %v", tt.value, err)
			}
			if got := d.String(); got != tt.want {
				t.Errorf("Decimal.Scan(%v) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}

	t.Run("unsupported type", func(t *testing.T) {
		var d Decimal
		if err := d.Scan(true); err == nil {
			t.Errorf("Decimal.Scan(bool) = nil error, want error")
		}
	})

	t.Run("nil", func(t *testing.T) {
		// A NULL column needs a nullable wrapper; the bare type rejects it, as shopspring's does.
		var d Decimal
		if err := d.Scan(nil); err == nil {
			t.Errorf("Decimal.Scan(nil) = nil error, want error")
		}
	})

	t.Run("round-trips through Value", func(t *testing.T) {
		original := MustFromString("-64500.4")
		value, err := original.Value()
		if err != nil {
			t.Fatalf("Decimal.Value(-64500.4) = %v", err)
		}
		var d Decimal
		if err := d.Scan(value); err != nil {
			t.Fatalf("Decimal.Scan(%v) = %v", value, err)
		}
		if d != original {
			t.Errorf("Decimal Value/Scan round-trip = %s, want %s", d, original)
		}
	})
}
