package validation

import "bytes"

// ValidTraceID confirms that trace ids are 128 bits
func ValidTraceID(id []byte) bool {
	return len(id) == 16
}

func ValidSpanID(id []byte) bool {
	return len(id) == 8 && !bytes.Equal(id, []byte{0, 0, 0, 0, 0, 0, 0, 0})
}

// SmallestPositiveNonZeroIntPerTenant is returning the minimal positive and
// non-zero value of the supplied limit function for all given tenants. In many
// limits a value of 0 means unlimted so the method will return 0 only if all
// inputs have a limit of 0 or an empty tenant list is given.
func SmallestPositiveNonZeroIntPerTenant(tenantIDs []string, f func(string) int) int {
	var result *int
	for _, tenantID := range tenantIDs {
		v := f(tenantID)
		if v > 0 && (result == nil || v < *result) {
			result = &v
		}
	}
	if result == nil {
		return 0
	}
	return *result
}
