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
		{"SearchDataMap1", func() SearchDataMap { return &SearchDataMap1{} }},
		{"SearchDataMap2", func() SearchDataMap { return &SearchDataMap2{} }},
	}

	testCases := []struct {
		name    string
		values  int
		repeats int
	}{
		{"insert", 1, 0},
		{"insert", 10, 0},
		{"insert", 100, 0},
		{"repeat", 1, 10},
		{"repeat", 10, 10},
		{"repeat", 100, 10},
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

func BenchmarkSearchDataMapAddr1(b *testing.B) {
	s := &SearchDataMap1{}
	for i := 0; i < b.N; i++ {
		s.Add("key", "value1")
		s.Add("key", "value2")
		s.Add("key", "value3")
		s.Add("key", "value4")
		s.Add("key", "value5")
	}
}

func BenchmarkSearchDataMapAddr2(b *testing.B) {
	s := &SearchDataMap2{}
	for i := 0; i < b.N; i++ {
		s.Add("key", "value1")
		s.Add("key", "value2")
		s.Add("key", "value3")
		s.Add("key", "value4")
		s.Add("key", "value5")
	}
}
