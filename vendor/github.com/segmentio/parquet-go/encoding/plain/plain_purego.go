//go:build purego || !amd64

package plain

func validateByteArray(b []byte) status {
	for i := 0; i < len(b); {
		r := len(b) - i
		if r < ByteArrayLengthSize {
			return errTooShort
		}
		n := ByteArrayLength(b[i:])
		i += ByteArrayLengthSize
		r -= ByteArrayLengthSize
		if n > r {
			return errTooShort
		}
		if n > MaxByteArrayLength {
			return errTooLarge
		}
		i += n
	}
	return ok
}
