package util

import (
	"context"
	"testing"

	"github.com/grafana/dskit/user"
)

var (
	benchmarkExtractValidOrgID  string
	benchmarkExtractValidOrgID2 string
)

func BenchmarkExtractValidOrgID(b *testing.B) {
	ctx := user.InjectOrgID(context.Background(), "bench-tenant")

	b.ResetTimer()

	var result string
	for i := 0; i < b.N; i++ {
		id, err := ExtractValidOrgID(ctx)
		if err != nil {
			b.Fatal(err)
		}
		result = id
	}

	benchmarkExtractValidOrgID = result
}

func BenchmarkExtractValidOrgID2(b *testing.B) {
	ctx := user.InjectOrgID(context.Background(), "bench-tenant")

	b.ResetTimer()

	var result string
	for i := 0; i < b.N; i++ {
		id, err := ExtractValidOrgID2(ctx)
		if err != nil {
			b.Fatal(err)
		}
		result = id
	}

	benchmarkExtractValidOrgID2 = result
}

// ./pkg/util/compare_benchmarks.sh
//                      │ /tmp/BenchmarkExtractValidOrgID.txt │ /tmp/BenchmarkExtractValidOrgID2.txt │
//                      │               sec/op                │   sec/op     vs base                 │
// ExtractValidOrgID-11                           15.28n ± 1%   37.16n ± 1%  +143.31% (p=0.000 n=10)

//                      │ /tmp/BenchmarkExtractValidOrgID.txt │ /tmp/BenchmarkExtractValidOrgID2.txt │
//                      │                B/op                 │        B/op         vs base          │
// ExtractValidOrgID-11                             0.00 ± 0%           16.00 ± 0%  ? (p=0.000 n=10)

//                      │ /tmp/BenchmarkExtractValidOrgID.txt │ /tmp/BenchmarkExtractValidOrgID2.txt │
//                      │              allocs/op              │     allocs/op       vs base          │
// ExtractValidOrgID-11                            0.000 ± 0%           1.000 ± 0%  ? (p=0.000 n=10)

// #!/bin/bash

// # Script to compare BenchmarkExtractValidOrgID and BenchmarkExtractValidOrgID2
// # using benchstat to show performance delta

// set -e

// # Configuration
// BENCH_COUNT=10
// BENCH_TIME=1000000x
// PACKAGE="./pkg/util"
// OUTPUT_DIR="/tmp"

// # Run both benchmarks together
// go test -bench='^Benchmark(ExtractValidOrgID|ExtractValidOrgID2)$' \
//     -benchmem \
//     -count=${BENCH_COUNT} \
//     -benchtime=${BENCH_TIME} \
//     ${PACKAGE} > ${OUTPUT_DIR}/bench_both.txt 2>&1

// # Split results into separate files
// grep "BenchmarkExtractValidOrgID-" ${OUTPUT_DIR}/bench_both.txt | \
//     grep -v "BenchmarkExtractValidOrgID2" > ${OUTPUT_DIR}/BenchmarkExtractValidOrgID.txt

// grep "BenchmarkExtractValidOrgID2-" ${OUTPUT_DIR}/bench_both.txt | \
//     sed 's/BenchmarkExtractValidOrgID2-/BenchmarkExtractValidOrgID-/' > ${OUTPUT_DIR}/BenchmarkExtractValidOrgID2.txt

// # Run benchstat to compare
// benchstat ${OUTPUT_DIR}/BenchmarkExtractValidOrgID.txt ${OUTPUT_DIR}/BenchmarkExtractValidOrgID2.txt
