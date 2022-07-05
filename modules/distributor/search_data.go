package distributor

import "github.com/grafana/tempo/pkg/model/trace"

// extractSearchDataAll returns flatbuffer search data for every trace.
func extractSearchDataAll(traces []*rebatchedTrace, extractTag trace.ExtractTagFunc) [][]byte {
	headers := make([][]byte, len(traces))

	for i, t := range traces {
		headers[i] = trace.ExtractSearchData(t.trace, t.id, extractTag)
	}

	return headers
}
