package decimal

// signBit carries the sign in the header byte, above the precision.
const signBit = 0x80

// MarshalBinary implements [encoding.BinaryMarshaler].
//
// The layout is one header byte holding the precision with the sign
// in its top bit, then the coefficient as minimal big-endian
// bytes. Zero writes no coefficient bytes, so a decimal costs
// one byte at zero and seventeen at the full 128-bit width.
func (d Decimal) MarshalBinary() ([]byte, error) {
	head := d.prec
	if d.neg {
		head |= signBit
	}

	return d.coef.appendBigEndian(append(make([]byte, 0, 17), head)), nil
}

// UnmarshalBinary implements [encoding.BinaryUnmarshaler].
// Foreign bytes spelling a non-canonical value normalize
// on the way in. It returns [ErrSyntax] for empty data and
// [ErrRange] for a precision or coefficient that does not fit.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return ErrSyntax
	}
	prec := data[0] &^ signBit
	if prec > MaxFractionDigits {
		return ErrRange
	}
	coef, ok := u128FromBigEndian(data[1:])
	if !ok {
		return ErrRange
	}
	*d = canonical(data[0]&signBit != 0, coef, int32(prec))

	return nil
}

// GobEncode implements [encoding/gob.GobEncoder]
// with the [Decimal.MarshalBinary] layout.
func (d Decimal) GobEncode() ([]byte, error) {
	return d.MarshalBinary()
}

// GobDecode implements [encoding/gob.GobDecoder] as [Decimal.UnmarshalBinary].
func (d *Decimal) GobDecode(data []byte) error {
	return d.UnmarshalBinary(data)
}
