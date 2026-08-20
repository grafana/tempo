package backendscheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/pkg/tempopb"
)

func idList(n int) [][]byte {
	ids := make([][]byte, n)
	for i := range ids {
		id := make([]byte, 16)
		id[0] = byte(i)
		id[1] = byte(i >> 8)
		id[2] = byte(i >> 16)
		ids[i] = id
	}
	return ids
}

// TestValidateRedactionRequestCapsTraceIDs bounds the explicit trace-ID list.
//
// Next() assigns the batch's whole list onto every job it hands out, so the dispatch cost is
// O(ids x blocks): a few million 16-byte IDs against a tenant with tens of thousands of blocks is
// terabytes of gRPC traffic out of the scheduler, which is a singleton. The gRPC message limit
// caps a single submission somewhere near 5.8M ids, which is a failure discovered under load
// rather than a stated boundary -- and far past the point where the cost is unreasonable.
func TestValidateRedactionRequestCapsTraceIDs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		n       int
		wantMsg string
	}{
		{name: "one id", n: 1},
		{name: "at the cap", n: maxRedactionTraceIDs},
		{name: "one over the cap", n: maxRedactionTraceIDs + 1, wantMsg: "too many trace_ids"},
		{name: "far over the cap", n: maxRedactionTraceIDs * 10, wantMsg: "too many trace_ids"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedactionRequest(&tempopb.SubmitRedactionRequest{TraceIds: idList(tc.n)}, nil)

			if tc.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Contains(t, err.Error(), tc.wantMsg)
			// The rejection has to name the way out, or an operator with a large list has no next
			// step but to split it up and pay the same dispatch cost across several batches.
			require.Contains(t, err.Error(), "query",
				"the error must point at the query selector, which costs nothing per job")
		})
	}
}

// TestValidateRedactionRequestCapDoesNotApplyToQuery pins that the cap is a property of shipping
// an ID list, not of redaction size. A query selects an unbounded number of traces and is resolved
// per block on the worker, so it carries no per-job payload at all.
func TestValidateRedactionRequestCapDoesNotApplyToQuery(t *testing.T) {
	err := validateRedactionRequest(
		&tempopb.SubmitRedactionRequest{},
		&tempopb.TraceQLSelector{Query: fmt.Sprintf(`{resource.namespace = %q}`, "checkout")},
	)
	require.NoError(t, err)
}
