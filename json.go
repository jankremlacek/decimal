package decimal

import (
	"bytes"
	"encoding/json"
)

// MarshalJSON implements [encoding/json.Marshaler], writing
// the value as a JSON string. A decimal is quoted on the wire
// because a JSON number is a float to most readers, and
// that would lose the precision this type exists to keep.
func (d Decimal) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, MaxDigits+5)
	b = append(b, '"')
	b = d.Append(b)

	return append(b, '"'), nil
}

// UnmarshalJSON implements [encoding/json.Unmarshaler], accepting
// the quoted and the bare numeric form. A JSON null leaves
// the receiver untouched. On error the receiver is untouched too.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		if bytes.IndexByte(b, '\\') < 0 {
			b = b[1 : len(b)-1]
		} else {
			// A JSON string may spell any character as an escape. Decoding
			// allocates, so only text that carries a backslash pays for it.
			var s string
			if err := json.Unmarshal(b, &s); err != nil {
				return err
			}
			b = []byte(s)
		}
	}
	parsed, err := parse(b)
	if err != nil {
		return err
	}
	*d = parsed

	return nil
}
