package decimal

import "strconv"

// InexactFloat64 reports the nearest float64,
// the name conceding what the conversion may cost.
func (d Decimal) InexactFloat64() float64 {
	var buf [MaxDigits + 3]byte
	// The canonical text spells a number well inside float64 range, so the parse cannot fail.
	f, _ := strconv.ParseFloat(string(d.Append(buf[:0])), 64)

	return f
}
