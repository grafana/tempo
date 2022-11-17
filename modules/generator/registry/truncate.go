package registry

func truncateLength(values []string, length int) {
	if length > 0 {
		for i, value := range values {
			if len(value) > length {
				values[i] = value[:length]
			}
		}
	}
}
