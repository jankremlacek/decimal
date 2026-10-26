package decimal

import (
	"bytes"
	"encoding/gob"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecimal_MarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want []byte
	}{
		{
			name: "zero is the bare header",
			v:    "0",
			want: []byte{0x00},
		},
		{
			name: "fraction",
			v:    "1.5",
			want: []byte{0x01, 0x0f},
		},
		{
			name: "negative sets the sign bit",
			v:    "-1.5",
			want: []byte{0x81, 0x0f},
		},
		{
			name: "single coefficient byte",
			v:    "255",
			want: []byte{0x00, 0xff},
		},
		{
			name: "minimal big-endian coefficient",
			v:    "256",
			want: []byte{0x00, 0x01, 0x00},
		},
		{
			name: "widest coefficient fills seventeen bytes",
			v:    "340282366920938463463374607431768211455",
			want: []byte{
				0x00,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MustFromString(tt.v).MarshalBinary()
			if err != nil {
				t.Fatalf("Decimal.MarshalBinary(%s) = %v", tt.v, err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Decimal.MarshalBinary(%s) mismatch:\n%s", tt.v, diff)
			}
		})
	}
}

func TestDecimal_UnmarshalBinary(t *testing.T) {
	t.Run("round-trips every shape", func(t *testing.T) {
		for _, in := range []string{
			"0", "1.5", "-1.5", "255", "-256", "64500.4",
			"0.0000000000000000001", "-0.0000000000000000001",
			"340282366920938463463374607431768211455",
			"-34028236692093846346.3374607431768211455",
		} {
			original := MustFromString(in)
			encoded, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("Decimal.MarshalBinary(%s) = %v", in, err)
			}
			var decoded Decimal
			if err := decoded.UnmarshalBinary(encoded); err != nil {
				t.Fatalf("Decimal.UnmarshalBinary(%s) = %v", in, err)
			}
			if decoded != original {
				t.Errorf("Decimal binary round-trip of %s = %s", in, decoded)
			}
		}
	})

	t.Run("foreign non-canonical bytes normalize", func(t *testing.T) {
		var d Decimal
		if err := d.UnmarshalBinary([]byte{0x02, 0x96}); err != nil {
			t.Fatalf("Decimal.UnmarshalBinary(1.50) = %v", err)
		}
		if d != MustFromString("1.5") {
			t.Errorf("Decimal.UnmarshalBinary(1.50) = %s, want canonical 1.5", d)
		}
	})

	t.Run("foreign negative zero normalizes", func(t *testing.T) {
		var d Decimal
		if err := d.UnmarshalBinary([]byte{0x81}); err != nil {
			t.Fatalf("Decimal.UnmarshalBinary(-0) = %v", err)
		}
		if d != Zero {
			t.Errorf("Decimal.UnmarshalBinary(-0) = %s, want Zero", d)
		}
	})

	tests := []struct {
		name    string
		in      []byte
		wantErr error
	}{
		{
			name:    "empty",
			in:      nil,
			wantErr: ErrSyntax,
		},
		{
			name:    "precision beyond the cap",
			in:      []byte{0x14, 0x01},
			wantErr: ErrRange,
		},
		{
			name: "coefficient beyond 128 bits",
			in: []byte{
				0x00,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
			},
			wantErr: ErrRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Decimal
			if err := d.UnmarshalBinary(tt.in); !errors.Is(err, tt.wantErr) {
				t.Errorf("Decimal.UnmarshalBinary(% x) = %v, want %v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestDecimal_Gob(t *testing.T) {
	// Through encoding/gob inside a struct, the way a caller persists decimals.
	type line struct {
		Price Decimal
		Qty   Decimal
	}
	src := line{Price: MustFromString("64500.4"), Qty: MustFromString("-1.25")}
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(src); err != nil {
		t.Fatalf("Encoding line to gob error: %v", err)
	}

	var dst line
	if err := gob.NewDecoder(buf).Decode(&dst); err != nil {
		t.Fatalf("Decoding line from gob error: %v", err)
	}

	if dst != src {
		t.Errorf("line gob round-trip = %+v, want %+v", dst, src)
	}
}
