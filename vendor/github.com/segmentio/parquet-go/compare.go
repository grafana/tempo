package parquet

import (
	"encoding/binary"

	"github.com/segmentio/parquet-go/deprecated"
)

func compareBool(v1, v2 bool) int {
	switch {
	case !v1 && v2:
		return -1
	case v1 && !v2:
		return +1
	default:
		return 0
	}
}

func compareInt32(v1, v2 int32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareInt64(v1, v2 int64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareInt96(v1, v2 deprecated.Int96) int {
	switch {
	case v1.Less(v2):
		return -1
	case v2.Less(v1):
		return +1
	default:
		return 0
	}
}

func compareFloat32(v1, v2 float32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareFloat64(v1, v2 float64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareUint32(v1, v2 uint32) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareUint64(v1, v2 uint64) int {
	switch {
	case v1 < v2:
		return -1
	case v1 > v2:
		return +1
	default:
		return 0
	}
}

func compareBE128(v1, v2 *[16]byte) int {
	x := binary.BigEndian.Uint64(v1[:8])
	y := binary.BigEndian.Uint64(v2[:8])
	switch {
	case x < y:
		return -1
	case x > y:
		return +1
	}
	x = binary.BigEndian.Uint64(v1[8:])
	y = binary.BigEndian.Uint64(v2[8:])
	switch {
	case x < y:
		return -1
	case x > y:
		return +1
	default:
		return 0
	}
}

func lessBE128(v1, v2 *[16]byte) bool {
	x := binary.BigEndian.Uint64(v1[:8])
	y := binary.BigEndian.Uint64(v2[:8])
	switch {
	case x < y:
		return true
	case x > y:
		return false
	}
	x = binary.BigEndian.Uint64(v1[8:])
	y = binary.BigEndian.Uint64(v2[8:])
	return x < y
}
