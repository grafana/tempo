package util

// MapSizeWithinLimit evaluates the total size of all keys in the map against the limit
func MapSizeWithinLimit(uniqueMap map[string]struct{}, limit int) bool {
	var mapSize int
	for key := range uniqueMap {
		mapSize += len(key)
	}

	return mapSize < limit
}
