package common

import (
	"strconv"
)

const (
	// NameObjects names the backend data object
	NameObjects = "data"
	// NameIndex names the backend index object
	NameIndex = "index"
	// nameBloomPrefix is the prefix used to build the bloom shards
	nameBloomPrefix = "bloom-"
)

// bloomName returns the backend bloom name for the given shard
func BloomName(shard int) string {
	return nameBloomPrefix + strconv.Itoa(shard)
}
