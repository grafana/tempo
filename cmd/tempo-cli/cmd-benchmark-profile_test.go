package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

func TestParseTraceIDCount(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: "0", want: 0},
		{in: "10000", want: 10000},
		{in: "all", want: benchmark.TraceIDsAll},
		{in: "ALL", want: benchmark.TraceIDsAll},
		{in: " all ", want: benchmark.TraceIDsAll},
		{in: "-1", wantErr: true},
		{in: "lots", wantErr: true},
		{in: "", wantErr: true},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseTraceIDCount(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
