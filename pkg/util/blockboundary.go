package util

import (
	"encoding/binary"
	"math"
)

// CreateBlockBoundaries splits the range of blockIDs into queryShards parts
func CreateBlockBoundaries(queryShards int) [][]byte {
	if queryShards == 0 {
		return nil
	}

	// create sharded queries
	blockBoundaries := make([][]byte, queryShards+1)
	for i := 0; i < queryShards+1; i++ {
		blockBoundaries[i] = make([]byte, 16)
	}

	// bucketSz is the min size for the bucket
	bucketSz := (math.MaxUint64 / uint64(queryShards))
	// numLarger is the number of buckets that have to be bumped by 1
	numLarger := (math.MaxUint64 % uint64(queryShards))
	boundary := uint64(0)
	for i := 0; i < queryShards; i++ {
		binary.BigEndian.PutUint64(blockBoundaries[i][:8], boundary)
		binary.BigEndian.PutUint64(blockBoundaries[i][8:], 0)

		boundary += bucketSz
		if numLarger != 0 {
			numLarger--
			boundary++
		}
	}

	binary.BigEndian.PutUint64(blockBoundaries[queryShards][:8], math.MaxUint64)
	binary.BigEndian.PutUint64(blockBoundaries[queryShards][8:], math.MaxUint64)

	return blockBoundaries
}
