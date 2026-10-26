package decimal

import "fmt"

// roundMode selects where a shed nonzero fraction sends the magnitude.
type roundMode int

const (
	roundHalfAway roundMode = iota // to nearest, ties away from zero
	roundTrunc                     // toward zero
	roundFloor                     // toward negative infinity
	roundCeil                      // toward positive infinity
)

// Round returns d rounded half away from zero to places fraction digits.
// Negative places rounds left of the point: Round(-2) rounds to hundreds.
// It panics with [ErrRange] when that carries the result out of range.
func (d Decimal) Round(places int32) Decimal {
	return d.round(places, roundHalfAway, "Round")
}

// RoundFloor returns d rounded toward negative infinity to places fraction
// digits. Negative places and range behave as in [Decimal.Round].
func (d Decimal) RoundFloor(places int32) Decimal {
	return d.round(places, roundFloor, "RoundFloor")
}

// RoundFloorSigFigs returns d rounded toward negative infinity to sigFigs
// significant figures. Zero is zero. Panics on sigFigs < 1.
//
// A floor across a power of ten may carry one figure
// more than asked: -99999.5 at five figures is -100000.
// A floor that leaves the range panics with [ErrRange].
func (d Decimal) RoundFloorSigFigs(sigFigs int32) Decimal {
	if sigFigs < 1 {
		panic(fmt.Errorf("decimal.RoundFloorSigFigs(%s, %d): sigFigs below 1", d, sigFigs))
	}

	// Returning early keeps sigFigs within the coefficient's 39
	// digits, so the places arithmetic below cannot overflow.
	digits := int32(d.coef.digits())
	if sigFigs >= digits {
		return d
	}

	// adjusted is floor(log10 |d|), the power of ten of the leading digit.
	adjusted := digits - 1 + d.Exponent()

	return d.round(sigFigs-1-adjusted, roundFloor, "RoundFloorSigFigs")
}

// RoundCeil returns d rounded toward positive infinity to places fraction
// digits. Negative places and range behave as in [Decimal.Round].
func (d Decimal) RoundCeil(places int32) Decimal {
	return d.round(places, roundCeil, "RoundCeil")
}

// Truncate returns d with the fraction cut to places digits, no rounding.
// Negative places cuts left of the point: Truncate(-2) of 5432 is 5400.
func (d Decimal) Truncate(places int32) Decimal {
	return d.round(places, roundTrunc, "Truncate")
}

// Floor returns the largest integer not above d.
func (d Decimal) Floor() Decimal {
	return d.round(0, roundFloor, "Floor")
}

// Ceil returns the smallest integer not below d.
func (d Decimal) Ceil() Decimal {
	return d.round(0, roundCeil, "Ceil")
}

// round sheds the fraction digits beyond places and applies
// mode to what fell off. Only rounding left of the point can
// leave the range, panicking with [ErrRange] under the op name.
func (d Decimal) round(places int32, mode roundMode, op string) Decimal {
	shed := int64(d.prec) - int64(places)
	if shed <= 0 || d.coef.isZero() {
		return d
	}

	// Shed in uint64-sized chunks, least significant first. The final
	// chunk alone decides a half-away tie: a top chunk below its midpoint
	// keeps the whole suffix below halfway, and one at the midpoint rounds
	// up whether or not the tail is zero, since ties go away from zero.
	// Any nonzero chunk makes the cut inexact for the directed modes.
	var (
		coef             = d.coef
		sticky           bool
		lastRem, lastMid uint64
		fullyShed        = true
	)
	for shed > 0 {
		if coef.isZero() {
			fullyShed = false
			break
		}
		step := min(shed, MaxFractionDigits)
		quo, rem := coef.divMod(pow10[step])
		sticky = sticky || rem != 0
		lastRem, lastMid = rem, pow10[step]/2
		coef, shed = quo, shed-step
	}

	var up bool
	switch mode {
	case roundHalfAway:
		up = fullyShed && lastRem >= lastMid
	case roundFloor:
		up = d.neg && sticky
	case roundCeil:
		up = !d.neg && sticky
	}
	if up {
		coef = coef.inc()
	}
	if coef.isZero() {
		return Decimal{}
	}
	if places <= 0 {
		// A nonzero coefficient times 10^40 or more always leaves 128 bits;
		// rejecting it first also keeps -places from overflowing at MinInt32.
		if places < -MaxDigits {
			panic(fmt.Errorf("decimal.%s(%s, %d): %w", op, d, places, ErrRange))
		}
		scaled, ok := coef.scale(-places)
		if !ok {
			panic(fmt.Errorf("decimal.%s(%s, %d): %w", op, d, places, ErrRange))
		}

		return Decimal{coef: scaled, neg: d.neg}
	}

	// Rounding up can mint trailing zeros; canonical sheds them.
	return canonical(d.neg, coef, places)
}
