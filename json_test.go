package decimal

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDecimal_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "fraction quoted",
			v:    "1.5",
			want: `"1.5"`,
		},
		{
			name: "negative quoted",
			v:    "-0.1",
			want: `"-0.1"`,
		},
		{
			name: "zero quoted",
			v:    "0",
			want: `"0"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MustFromString(tt.v).MarshalJSON()
			if err != nil {
				t.Fatalf("Decimal.MarshalJSON(%s) = %v", tt.v, err)
			}
			if string(got) != tt.want {
				t.Errorf("Decimal.MarshalJSON(%s) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "quoted string",
			in:   `"64500.4"`,
			want: "64500.4",
		},
		{
			name: "bare number",
			in:   `123.45`,
			want: "123.45",
		},
		{
			name: "quoted exponent form",
			in:   `"1.5e3"`,
			want: "1500",
		},
		{
			name: "bare exponent form",
			in:   `1.5e3`,
			want: "1500",
		},
		{
			name:    "quoted garbage",
			in:      `"abc"`,
			wantErr: true,
		},
		{
			name:    "unquoted garbage",
			in:      `abc`,
			wantErr: true,
		},
		{
			name:    "empty quotes",
			in:      `""`,
			wantErr: true,
		},
		{
			name:    "object",
			in:      `{}`,
			wantErr: true,
		},
		{
			name:    "unterminated quote",
			in:      `"5`,
			wantErr: true,
		},
		{
			name: "escaped characters",
			in:   `"\u0031\u002e5"`,
			want: "1.5",
		},
		{
			name:    "invalid escape",
			in:      `"1\x5"`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Decimal
			err := d.UnmarshalJSON([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Decimal.UnmarshalJSON(%s) = %v, wantErr %t", tt.in, err, tt.wantErr)
			}
			if err == nil && d.String() != tt.want {
				t.Errorf("Decimal.UnmarshalJSON(%s) = %s, want %s", tt.in, d, tt.want)
			}
		})
	}

	t.Run("null leaves the receiver untouched", func(t *testing.T) {
		d := MustFromString("7")
		if err := d.UnmarshalJSON([]byte("null")); err != nil {
			t.Fatalf("Decimal.UnmarshalJSON(null) = %v", err)
		}
		if d != MustFromString("7") {
			t.Errorf("Decimal.UnmarshalJSON(null) = %s, want 7 kept", d)
		}
	})
}

func TestDecimal_JSONRoundTrip(t *testing.T) {
	// Through encoding/json inside a struct, the way decimals travel on the wire.
	type line struct {
		Price Decimal `json:"price"`
		Qty   Decimal `json:"qty"`
	}
	original := line{Price: MustFromString("64500.4"), Qty: MustFromString("1.25")}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(line) = %v", err)
	}
	if !bytes.Contains(encoded, []byte(`"price":"64500.4"`)) {
		t.Errorf("json.Marshal(line) = %s, decimal did not serialize as a quoted string", encoded)
	}

	var decoded line
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(line) = %v", err)
	}
	if decoded != original {
		t.Errorf("line JSON round-trip = %+v, want %+v", decoded, original)
	}
}
