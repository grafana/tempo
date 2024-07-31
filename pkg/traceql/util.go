package traceql

import (
	"github.com/grafana/tempo/pkg/tempopb"
)

func MakeCollectTagValueFunc(collect func(tempopb.TagValue) bool) func(v Static) bool {
	return func(v Static) bool {
		tv := tempopb.TagValue{}

		switch v.Type {
		case TypeString:
			tv.Type = "string"
			tv.Value = v.EncodeToString(false) // avoid formatting

		case TypeBoolean:
			tv.Type = "bool"
			tv.Value = v.String()

		case TypeInt:
			tv.Type = "int"
			tv.Value = v.String()

		case TypeFloat:
			tv.Type = "float"
			tv.Value = v.String()

		case TypeDuration:
			tv.Type = duration
			tv.Value = v.String()

		case TypeStatus:
			tv.Type = "keyword"
			tv.Value = v.String()
		}

		return collect(tv)
	}
}

// bucketSet is a simple set of buckets that can be used to track the number of exemplars
type bucketSet struct {
	sz, maxTotal, maxBucket int
	buckets                 []int
}

func newBucketSet(size int) *bucketSet {
	return &bucketSet{
		sz:        size,
		maxTotal:  maxExemplars,
		maxBucket: maxExemplarsPerBucket,
		buckets:   make([]int, size+1), // +1 for total count
	}
}

func (b *bucketSet) len() int {
	return b.buckets[b.sz]
}

func (b *bucketSet) testTotal() bool {
	return b.len() >= b.maxTotal
}

func (b *bucketSet) inRange(i int) bool {
	return i >= 0 && i < b.sz
}

func (b *bucketSet) addAndTest(i int) bool {
	if !b.inRange(i) || b.testTotal() {
		return true
	}

	if b.buckets[i] >= b.maxBucket {
		return true
	}

	b.buckets[i]++
	b.buckets[b.sz]++
	return false
}
