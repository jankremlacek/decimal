package decimal

import (
	"errors"
	"math"
	"testing"
)

func TestDecimal_Round(t *testing.T) {
	tests := []struct {
		name   string
		v      string
		places int32
		want   string
	}{
		{
			name:   "tie rounds away",
			v:      "2.5",
			places: 0,
			want:   "3",
		},
		{
			name:   "negative tie rounds away",
			v:      "-2.5",
			places: 0,
			want:   "-3",
		},
		{
			name:   "tie at depth rounds away",
			v:      "2.45",
			places: 1,
			want:   "2.5",
		},
		{
			name:   "negative tie at depth rounds away",
			v:      "-2.45",
			places: 1,
			want:   "-2.5",
		},
		{
			name:   "below the tie rounds down",
			v:      "2.44",
			places: 1,
			want:   "2.4",
		},
		{
			name:   "rounding mints then sheds a zero",
			v:      "1.95",
			places: 1,
			want:   "2",
		},
		{
			name:   "half rounds up to one",
			v:      "0.5",
			places: 0,
			want:   "1",
		},
		{
			name:   "shy of half vanishes",
			v:      "0.049",
			places: 1,
			want:   "0",
		},
		{
			name:   "hundreds below the tie",
			v:      "12345",
			places: -2,
			want:   "12300",
		},
		{
			name:   "hundreds at the tie",
			v:      "12350",
			places: -2,
			want:   "12400",
		},
		{
			name:   "places beyond the fraction is a no-op",
			v:      "1.5",
			places: 4,
			want:   "1.5",
		},
		{
			name:   "zero",
			v:      "0",
			places: 2,
			want:   "0",
		},
		{
			name:   "wei far left of the point vanishes",
			v:      "0.0000000000000000001",
			places: -21,
			want:   "0",
		},
		{
			name:   "chunked shed over a wide coefficient",
			v:      "200000000000000000000000000000000000000",
			places: -19,
			want:   "200000000000000000000000000000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Round(tt.places).String(); got != tt.want {
				t.Errorf("Decimal.Round(%s, %d) = %s, want %s", tt.v, tt.places, got, tt.want)
			}
		})
	}

	t.Run("panics when rounding up leaves the range", func(t *testing.T) {
		d := MustFromString("340282366920938463463374607431768211455")
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.Round panic = %v, want ErrRange", err)
			}
		}()
		d.Round(-35)
	})
}

func TestDecimal_RoundFloor(t *testing.T) {
	tests := []struct {
		name   string
		v      string
		places int32
		want   string
	}{
		{
			name:   "positive cuts down",
			v:      "1.9",
			places: 0,
			want:   "1",
		},
		{
			name:   "negative moves away",
			v:      "-1.2",
			places: 0,
			want:   "-2",
		},
		{
			name:   "tens",
			v:      "145",
			places: -1,
			want:   "140",
		},
		{
			name:   "negative tens move away",
			v:      "-145",
			places: -1,
			want:   "-150",
		},
		{
			name:   "exact cut unchanged",
			v:      "1.5",
			places: 1,
			want:   "1.5",
		},
		{
			name:   "negative wei floors a whole step left",
			v:      "-0.0000000000000000001",
			places: -21,
			want:   "-1000000000000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).RoundFloor(tt.places).String(); got != tt.want {
				t.Errorf("Decimal.RoundFloor(%s, %d) = %s, want %s", tt.v, tt.places, got, tt.want)
			}
		})
	}

	t.Run("panics when flooring leaves the range", func(t *testing.T) {
		d := MustFromString("-340282366920938463463374607431768211455")
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.RoundFloor panic = %v, want ErrRange", err)
			}
		}()
		d.RoundFloor(-38)
	})

	t.Run("panics at the deepest places", func(t *testing.T) {
		d := MustFromString("-0.1")
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.RoundFloor panic = %v, want ErrRange", err)
			}
		}()
		d.RoundFloor(math.MinInt32)
	})
}

func TestDecimal_RoundFloorSigFigs(t *testing.T) {
	tests := []struct {
		name    string
		v       string
		sigFigs int32
		want    string
	}{
		{
			name:    "below a power of ten",
			v:       "99999.5",
			sigFigs: 5,
			want:    "99999",
		},
		{
			name:    "above a power of ten",
			v:       "100000.5",
			sigFigs: 5,
			want:    "100000",
		},
		{
			name:    "negative floors across a power of ten",
			v:       "-99999.5",
			sigFigs: 5,
			want:    "-100000",
		},
		{
			name:    "seven figures cut the fraction",
			v:       "112345.67",
			sigFigs: 7,
			want:    "112345.6",
		},
		{
			name:    "five figures round left of the point",
			v:       "112345.67",
			sigFigs: 5,
			want:    "112340",
		},
		{
			name:    "sub-unit",
			v:       "0.0012345",
			sigFigs: 3,
			want:    "0.00123",
		},
		{
			name:    "fewer figures than asked unchanged",
			v:       "1.5",
			sigFigs: 5,
			want:    "1.5",
		},
		{
			name:    "coefficient bound",
			v:       "340282366920938463463374607431768211455",
			sigFigs: 5,
			want:    "340280000000000000000000000000000000000",
		},
		{
			name:    "fifty-eight figures unchanged",
			v:       "340282366920938463463374607431768211455",
			sigFigs: 58,
			want:    "340282366920938463463374607431768211455",
		},
		{
			name:    "max int32 figures unchanged",
			v:       "-34028236692093846346.3374607431768211455",
			sigFigs: math.MaxInt32,
			want:    "-34028236692093846346.3374607431768211455",
		},
		{
			name:    "zero",
			v:       "0",
			sigFigs: 5,
			want:    "0",
		},
		{
			name:    "one figure",
			v:       "112345.67",
			sigFigs: 1,
			want:    "100000",
		},
		{
			name:    "negative one figure moves away",
			v:       "-112345.67",
			sigFigs: 1,
			want:    "-200000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).RoundFloorSigFigs(tt.sigFigs).String(); got != tt.want {
				t.Errorf("Decimal.RoundFloorSigFigs(%s, %d) = %s, want %s", tt.v, tt.sigFigs, got, tt.want)
			}
		})
	}

	t.Run("panics on zero figures", func(t *testing.T) {
		defer func() {
			if p := recover(); p == nil {
				t.Errorf("Decimal.RoundFloorSigFigs(1.5, 0) did not panic")
			}
		}()
		MustFromString("1.5").RoundFloorSigFigs(0)
	})

	t.Run("panics on negative figures", func(t *testing.T) {
		defer func() {
			if p := recover(); p == nil {
				t.Errorf("Decimal.RoundFloorSigFigs(1.5, -1) did not panic")
			}
		}()
		MustFromString("1.5").RoundFloorSigFigs(-1)
	})

	t.Run("panics when flooring leaves the range", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.RoundFloorSigFigs panic = %v, want ErrRange", err)
			}
		}()
		MustFromString("-340282366920938463463374607431768211455").RoundFloorSigFigs(1)
	})
}

func TestDecimal_RoundCeil(t *testing.T) {
	tests := []struct {
		name   string
		v      string
		places int32
		want   string
	}{
		{
			name:   "positive moves away",
			v:      "1.0001",
			places: 0,
			want:   "2",
		},
		{
			name:   "negative cuts up",
			v:      "-1.2",
			places: 0,
			want:   "-1",
		},
		{
			name:   "tens",
			v:      "145",
			places: -1,
			want:   "150",
		},
		{
			name:   "exact cut unchanged",
			v:      "1.5",
			places: 1,
			want:   "1.5",
		},
		{
			name:   "wei ceils a whole step left",
			v:      "0.0000000000000000001",
			places: -21,
			want:   "1000000000000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).RoundCeil(tt.places).String(); got != tt.want {
				t.Errorf("Decimal.RoundCeil(%s, %d) = %s, want %s", tt.v, tt.places, got, tt.want)
			}
		})
	}

	t.Run("panics when ceiling leaves the range", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.RoundCeil panic = %v, want ErrRange", err)
			}
		}()
		MustFromString("340000000000000000000000000000000000000").RoundCeil(-39)
	})

	t.Run("panics at the deepest places", func(t *testing.T) {
		d := MustFromString("0.1")
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrRange) {
				t.Errorf("Decimal.RoundCeil panic = %v, want ErrRange", err)
			}
		}()
		d.RoundCeil(math.MinInt32)
	})
}

func TestDecimal_Truncate(t *testing.T) {
	tests := []struct {
		name   string
		v      string
		places int32
		want   string
	}{
		{
			name:   "cuts without rounding",
			v:      "1.999",
			places: 1,
			want:   "1.9",
		},
		{
			name:   "negative cuts toward zero",
			v:      "-1.999",
			places: 1,
			want:   "-1.9",
		},
		{
			name:   "places beyond the fraction is a no-op",
			v:      "1.999",
			places: 8,
			want:   "1.999",
		},
		{
			name:   "negative places cuts left of the point",
			v:      "12345",
			places: -2,
			want:   "12300",
		},
		{
			name:   "negative places cuts a negative toward zero",
			v:      "-12399",
			places: -2,
			want:   "-12300",
		},
		{
			name:   "negative places past the magnitude truncates to zero",
			v:      "999",
			places: -3,
			want:   "0",
		},
		{
			name:   "half truncates to zero",
			v:      "0.5",
			places: 0,
			want:   "0",
		},
		{
			name:   "wei truncates to zero",
			v:      "0.0000000000000000001",
			places: 0,
			want:   "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Truncate(tt.places).String(); got != tt.want {
				t.Errorf("Decimal.Truncate(%s, %d) = %s, want %s", tt.v, tt.places, got, tt.want)
			}
		})
	}
}

func TestDecimal_Floor(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "positive cuts down",
			v:    "1.2",
			want: "1",
		},
		{
			name: "negative moves away",
			v:    "-1.2",
			want: "-2",
		},
		{
			name: "integer unchanged",
			v:    "-3",
			want: "-3",
		},
		{
			name: "zero",
			v:    "0",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Floor().String(); got != tt.want {
				t.Errorf("Decimal.Floor(%s) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}

func TestDecimal_Ceil(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "positive moves away",
			v:    "1.2",
			want: "2",
		},
		{
			name: "negative cuts up",
			v:    "-1.2",
			want: "-1",
		},
		{
			name: "integer unchanged",
			v:    "3",
			want: "3",
		},
		{
			name: "zero",
			v:    "0",
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustFromString(tt.v).Ceil().String(); got != tt.want {
				t.Errorf("Decimal.Ceil(%s) = %s, want %s", tt.v, got, tt.want)
			}
		})
	}
}
