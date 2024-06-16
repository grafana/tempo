package pipeline

import (
	"testing"
)

func TestTraceQueryFilter(t *testing.T) {
	//nextFn := func(query string) http.RoundTripper {
	//	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
	//		return &http.Response{}, nil
	//	})
	//}

	//tcs := []struct {
	//	query string
	//	regex string
	//}{
	//	{},
	//}

	//for _, tc := range tcs {
	//
	//}
}

func TestTraceQueryFilterMatch(t *testing.T) {

}

func TestTraceQueryFilterMultipleMatches(t *testing.T) {

}

func BenchmarkTraceQueryFilter(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
