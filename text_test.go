package decimal

import (
	"errors"
	"testing"
)

func TestDecimal_String(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "zero",
			v:    "0",
			want: "0",
		},
		{
			name: "integer",
			v:    "1000",
			want: "1000",
		},
		{
			name: "fraction",
			v:    "123.456",
			want: "123.456",
		},
		{
			name: "negative trimmed",
			v:    "-0.10",
			want: "-0.1",
		},
		{
			name: "leading fraction zeros",
			v:    "0.001",
			want: "0.001",
		},
		{
			name: "interior zeros",
			v:    "1.002",
			want: "1.002",
		},
		{
			name: "coefficient shorter than the fraction",
			v:    "0.0000000000000000001",
			want: "0.0000000000000000001",
		},
		{
			name: "widest coefficient",
			v:    "340282366920938463463374607431768211455",
			want: "340282366920938463463374607431768211455",
		},
		{
			name: "widest full-precision value",
			v:    "-34028236692093846346.3374607431768211455",
			want: "-34028236692093846346.3374607431768211455",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).String(); got != tt.want {
				t.Errorf("Decimal.String() of %q = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_Append(t *testing.T) {
	got := MustFromString("64500.4").Append([]byte("price="))
	if string(got) != "price=64500.4" {
		t.Errorf("Decimal.Append(price=) = %s, want price=64500.4", got)
	}
}

func TestDecimal_TextLen(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want int
	}{
		{
			name: "zero",
			v:    "0",
			want: 1,
		},
		{
			name: "integer",
			v:    "1000",
			want: 4,
		},
		{
			name: "negative integer",
			v:    "-1000",
			want: 5,
		},
		{
			name: "fraction",
			v:    "123.456",
			want: 7,
		},
		{
			name: "coefficient shorter than the fraction",
			v:    "0.0000000000000000001",
			want: 21,
		},
		{
			name: "trailing fraction zeros shed",
			v:    "-0.10",
			want: 4,
		},
		{
			name: "widest coefficient",
			v:    "340282366920938463463374607431768211455",
			want: 39,
		},
		{
			name: "widest full-precision value",
			v:    "-34028236692093846346.3374607431768211455",
			want: 41,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MustFromString(tt.v)

			got := d.TextLen()

			if got != tt.want {
				t.Errorf("Decimal.TextLen() of %q = %d, want %d", tt.v, got, tt.want)
			}
			if written := len(d.Append(nil)); got != written {
				t.Errorf("Decimal.TextLen() of %q = %d, Append wrote %d", tt.v, got, written)
			}
		})
	}
}

func TestDecimal_StringFixed(t *testing.T) {
	tests := []struct {
		name   string
		v      string
		places int32
		want   string
	}{
		{
			name:   "pads fraction",
			v:      "1.5",
			places: 4,
			want:   "1.5000",
		},
		{
			name:   "zero",
			v:      "0",
			places: 2,
			want:   "0.00",
		},
		{
			name:   "rounds half away",
			v:      "-1.995",
			places: 2,
			want:   "-2.00",
		},
		{
			name:   "cuts fraction",
			v:      "1.234567",
			places: 2,
			want:   "1.23",
		},
		{
			name:   "rounds up into the integer",
			v:      "3.6",
			places: 0,
			want:   "4",
		},
		{
			name:   "negative places round left of the point",
			v:      "545",
			places: -1,
			want:   "550",
		},
		{
			name:   "small negative rounds away",
			v:      "-0.005",
			places: 2,
			want:   "-0.01",
		},
		{
			name:   "pads beyond the fraction",
			v:      "0.001",
			places: 5,
			want:   "0.00100",
		},
		{
			name:   "integer keeps no point at zero places",
			v:      "42",
			places: 0,
			want:   "42",
		},
		{
			name:   "wide integer",
			v:      "123456789012345678901234567890123456789",
			places: 0,
			want:   "123456789012345678901234567890123456789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).StringFixed(tt.places); got != tt.want {
				t.Errorf("Decimal.StringFixed(%s, %d) = %q, want %q", tt.v, tt.places, got, tt.want)
			}
		})
	}
}

func TestDecimal_MarshalText(t *testing.T) {
	got, err := MustFromString("-1.5").MarshalText()
	if err != nil {
		t.Fatalf("Decimal.MarshalText() = %v", err)
	}
	if string(got) != "-1.5" {
		t.Errorf("Decimal.MarshalText() = %s, want -1.5", got)
	}
}

func TestDecimal_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{
			name: "fraction",
			in:   "1.500",
			want: "1.5",
		},
		{
			name: "exponent form",
			in:   "1.5e3",
			want: "1500",
		},
		{
			name:    "malformed",
			in:      "abc",
			wantErr: ErrSyntax,
		},
		{
			name:    "out of range",
			in:      "1e40",
			wantErr: ErrRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Decimal
			err := d.UnmarshalText([]byte(tt.in))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Decimal.UnmarshalText(%q) = %v, want %v", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Decimal.UnmarshalText(%q) = %v", tt.in, err)
			}
			if got := d.String(); got != tt.want {
				t.Errorf("Decimal.UnmarshalText(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
