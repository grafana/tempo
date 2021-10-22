package tempofb

import (
	"fmt"
	"testing"
)

func BenchmarkSearchDataMapAdd(b *testing.B) {
	intfs := []struct {
		name string
		f    func() SearchDataMap
	}{
		{"SearchDataMapSmall", func() SearchDataMap { return &SearchDataMapSmall{} }},
		{"SearchDataMapLarge", func() SearchDataMap { return &SearchDataMapLarge{} }},
	}

	testCases := []struct {
		name    string
		values  int
		repeats int
	}{
		{"insert", 1, 0},
		{"insert", 5, 0},
		{"insert", 10, 0},
		{"insert", 20, 0},
		{"insert", 100, 0},
		{"repeat", 10, 10},
		{"repeat", 10, 100},
		{"repeat", 100, 10},
		{"repeat", 100, 100},
	}

	for _, tc := range testCases {
		for _, intf := range intfs {
			b.Run(fmt.Sprint(tc.name, "/", tc.values, "x value/", tc.repeats, "x repeat", "/", intf.name), func(b *testing.B) {

				var k []string
				for i := 0; i < b.N; i++ {
					k = append(k, fmt.Sprintf("key%d", i))
				}

				var v []string
				for i := 0; i < tc.values; i++ {
					v = append(v, fmt.Sprintf("value%d", i))
				}

				s := intf.f()
				insert := func() {
					for i := 0; i < len(k); i++ {
						for j := 0; j < len(v); j++ {
							s.Add(k[i], v[j])
						}
					}
				}

				// insert
				b.ResetTimer()
				insert()

				// reinsert?
				if tc.repeats > 0 {
					b.ResetTimer()
					insert()
				}
			})
		}
	}

}
