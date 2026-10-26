package decimal

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
)

// Value implements [driver.Valuer], rendering the canonical text.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements [sql.Scanner], accepting the driver
// shapes a numeric column arrives as: text raw or with
// one pair of surrounding quotes, an int64, or a float64.
// Unlike the constructors, a bad value fails the scan with
// an error — this is a database boundary, never a panic site.
func (d *Decimal) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return d.scanText([]byte(v))
	case []byte:
		return d.scanText(v)
	case int64:
		*d = NewFromInt(v)
		return nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("decimal.Scan(%v): %w", v, ErrSyntax)
		}
		var buf [32]byte
		parsed, err := parse(strconv.AppendFloat(buf[:0], v, 'f', -1, 64))
		if err != nil {
			return fmt.Errorf("decimal.Scan(%v): %w", v, err)
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("decimal.Scan: cannot scan %T", value)
	}
}

// scanText parses column text, stripping one pair
// of surrounding quotes a driver may leave on it.
func (d *Decimal) scanText(b []byte) error {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	parsed, err := parse(b)
	if err != nil {
		return fmt.Errorf("decimal.Scan(%q): %w", b, err)
	}
	*d = parsed

	return nil
}
