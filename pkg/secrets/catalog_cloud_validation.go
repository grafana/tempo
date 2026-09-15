package secrets

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/encoding/protowire"
)

// validFlyAccessToken follows the array representation in superfly/macaroon's
// macaroon.go, nonce.go and caveat_set.go at the catalog's pinned revision.
// This checks the outer container, not caveat semantics or authentication.
func validFlyAccessToken(candidate string) bool {
	prefix, body, ok := strings.Cut(candidate, "_")
	if !ok || (prefix != "fm1r" && prefix != "fm1a" && prefix != "fm2") || len(body) > 1002 {
		return false
	}
	var decoded [753]byte
	n, err := base64.StdEncoding.Decode(decoded[:], []byte(body))
	if err != nil {
		return false
	}
	data := flyMessagePack(decoded[:n])
	fields, ok := data.arrayLen()
	if !ok || fields != 4 {
		return false
	}
	nonceFields, ok := data.arrayLen()
	if !ok || (nonceFields != 2 && nonceFields != 3) {
		return false
	}
	if _, ok := data.bytes(); !ok { // The key ID is opaque, including nil/empty IDs.
		return false
	}
	if size, ok := data.bytes(); !ok || size != 16 {
		return false
	}
	if nonceFields == 3 {
		flag, ok := data.byte()
		if !ok || (flag != 0xc0 && flag != 0xc2 && flag != 0xc3) {
			return false
		}
	}
	if _, ok := data.bytes(); !ok { // The source decoder accepts strings or bytes for location.
		return false
	}
	caveatFields, ok := data.arrayLen()
	if !ok || caveatFields%2 != 0 || caveatFields > uint64(len(data)) {
		return false
	}
	for i := uint64(0); i < caveatFields; i += 2 {
		if !data.integer() || !data.value() {
			return false
		}
	}
	size, ok := data.bytes()
	return ok && size == 32 && len(data) == 0
}

// flyMessagePack consumes slices of a fixed-size decoded buffer. Each operation
// advances the cursor or fails; declared lengths never allocate or control work
// beyond the remaining bytes. MessagePack string/bin compatibility and nil scalar
// handling follow vmihailenco/msgpack v5.3.5, used by the pinned Fly decoder.
type flyMessagePack []byte

func (data *flyMessagePack) byte() (byte, bool) {
	if len(*data) == 0 {
		return 0, false
	}
	value := (*data)[0]
	*data = (*data)[1:]
	return value, true
}

func (data *flyMessagePack) take(size uint64) bool {
	if size > uint64(len(*data)) {
		return false
	}
	*data = (*data)[size:]
	return true
}

func (data *flyMessagePack) unsigned(size int) (uint64, bool) {
	if size > len(*data) {
		return 0, false
	}
	var value uint64
	for _, b := range (*data)[:size] {
		value = value<<8 | uint64(b)
	}
	*data = (*data)[size:]
	return value, true
}

func (data *flyMessagePack) arrayLen() (uint64, bool) {
	code, ok := data.byte()
	if !ok {
		return 0, false
	}
	switch {
	case code >= 0x90 && code <= 0x9f:
		return uint64(code & 0x0f), true
	case code == 0xdc:
		return data.unsigned(2)
	case code == 0xdd:
		return data.unsigned(4)
	default:
		return 0, false
	}
}

func (data *flyMessagePack) bytes() (uint64, bool) {
	code, ok := data.byte()
	if !ok {
		return 0, false
	}
	var size uint64
	switch {
	case code == 0xc0: // nil is accepted by the source string and byte decoders.
		return 0, true
	case code >= 0xa0 && code <= 0xbf:
		size = uint64(code & 0x1f)
	case code == 0xc4 || code == 0xd9:
		size, ok = data.unsigned(1)
	case code == 0xc5 || code == 0xda:
		size, ok = data.unsigned(2)
	case code == 0xc6 || code == 0xdb:
		size, ok = data.unsigned(4)
	default:
		return 0, false
	}
	return size, ok && data.take(size)
}

func (data *flyMessagePack) integer() bool {
	code, ok := data.byte()
	if !ok {
		return false
	}
	switch {
	case code <= 0x7f || code >= 0xe0 || code == 0xc0:
		return true
	case code >= 0xcc && code <= 0xcf:
		return data.take(uint64(1) << (code - 0xcc))
	case code >= 0xd0 && code <= 0xd3:
		return data.take(uint64(1) << (code - 0xd0))
	default:
		return false
	}
}

// value skips one complete MessagePack value, including nested caveat containers.
// Counting pending values instead of recursing bounds stack space independently
// of nesting depth. Unknown caveat types remain representable, not authorized.
func (data *flyMessagePack) value() bool {
	pending := uint64(1)
	for pending > 0 {
		code, ok := data.byte()
		if !ok {
			return false
		}
		pending--
		var size, children uint64
		switch {
		case code <= 0x7f || code >= 0xe0 || code == 0xc0 || code == 0xc2 || code == 0xc3:
		case code >= 0x80 && code <= 0x8f:
			children = 2 * uint64(code&0x0f)
		case code >= 0x90 && code <= 0x9f:
			children = uint64(code & 0x0f)
		case code >= 0xa0 && code <= 0xbf:
			size = uint64(code & 0x1f)
		case code >= 0xc4 && code <= 0xc9:
			// bin8/16/32 or ext8/16/32; extensions also carry a type byte.
			width := (code - 0xc4) % 3
			size, ok = data.unsigned(1 << width)
			if code >= 0xc7 {
				size++
			}
		case code == 0xca:
			size = 4
		case code == 0xcb:
			size = 8
		case code >= 0xcc && code <= 0xd3:
			size = uint64(1) << ((code - 0xcc) % 4)
		case code >= 0xd4 && code <= 0xd8:
			size = 1 + uint64(1)<<(code-0xd4)
		case code >= 0xd9 && code <= 0xdb:
			size, ok = data.unsigned(1 << (code - 0xd9))
		case code >= 0xdc && code <= 0xdf:
			children, ok = data.unsigned(2 << ((code - 0xdc) % 2))
			if code >= 0xde {
				children *= 2
			}
		default: // 0xc1 is reserved, not a MessagePack value.
			return false
		}
		if !ok || !data.take(size) || children > uint64(len(*data)) {
			return false
		}
		pending += children
		if pending > uint64(len(*data)) {
			return false
		}
	}
	return true
}

// validVaultServiceToken validates source-established modern service forms.
// Historical single-letter prefixes are intentionally outside native scope.
func validVaultServiceToken(candidate string) bool {
	prefix, body, ok := strings.Cut(candidate, ".")
	if !ok || prefix != "hvs" {
		return false
	}
	body, namespace, hasNamespace := strings.Cut(body, ".")
	if hasNamespace && (len(namespace) < 1 || len(namespace) > 5 || !vaultBase62(namespace)) {
		return false
	}
	if len(body) == 24 && vaultBase62(body) {
		return true
	}
	if len(body) < 24 || len(body) > 1000 {
		return false
	}
	var decoded [750]byte
	n, err := base64.RawURLEncoding.Decode(decoded[:], []byte(body))
	return err == nil && validVaultSignedToken(decoded[:n])
}

func vaultBase62(value string) bool {
	for i := range len(value) {
		b := value[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') {
			return false
		}
	}
	return true
}

// The protobuf parser preserves field reordering, last-value-wins scalar/bytes
// semantics and unknown fields. A fixed 750-byte input bounds all wire parsing,
// including the library's unknown-group traversal. HMAC bytes are never verified.
func validVaultSignedToken(data []byte) bool {
	var version uint64
	var signature, token []byte
	for len(data) > 0 {
		number, wireType, n := protowire.ConsumeTag(data)
		if n < 0 || !number.IsValid() {
			return false
		}
		data = data[n:]
		switch number {
		case 1:
			if wireType != protowire.VarintType {
				return false
			}
			version, n = protowire.ConsumeVarint(data)
		case 2, 3:
			if wireType != protowire.BytesType {
				return false
			}
			var value []byte
			value, n = protowire.ConsumeBytes(data)
			if number == 2 {
				signature = value
			} else {
				token = value
			}
		default:
			n = protowire.ConsumeFieldValue(number, wireType, data)
		}
		if n < 0 {
			return false
		}
		data = data[n:]
	}
	return version == 1 && len(signature) == 32 && validVaultToken(token)
}

func validVaultToken(data []byte) bool {
	var random []byte
	for len(data) > 0 {
		number, wireType, n := protowire.ConsumeTag(data)
		if n < 0 || !number.IsValid() {
			return false
		}
		data = data[n:]
		switch number {
		case 1:
			if wireType != protowire.BytesType {
				return false
			}
			random, n = protowire.ConsumeBytes(data)
			if n >= 0 && !utf8.Valid(random) {
				return false
			}
		case 2, 3:
			if wireType != protowire.VarintType {
				return false
			}
			_, n = protowire.ConsumeVarint(data)
		default:
			n = protowire.ConsumeFieldValue(number, wireType, data)
		}
		if n < 0 {
			return false
		}
		data = data[n:]
	}
	return len(random) > 0
}
