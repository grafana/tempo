package validation

// ValidTraceID confirms that trace ids are 128 bits
func ValidTraceID(id []byte) bool {
	return len(id) == 16
}
