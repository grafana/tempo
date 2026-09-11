package ingest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
)

func TestHandleKafkaError(t *testing.T) {
	tests := []struct {
		err             error
		expectedRefresh bool
	}{
		{nil, false},
		{errors.New("Some error"), false},
		{errors.New("unknown broker"), true},
		{kerr.NotLeaderForPartition, true},
		{kerr.ReplicaNotAvailable, true},
		{kerr.UnknownLeaderEpoch, true},
		{kerr.LeaderNotAvailable, true},
		{kerr.BrokerNotAvailable, true},
		{kerr.UnknownTopicOrPartition, true},
		{kerr.NetworkException, true},
		{kerr.NotCoordinator, true},
		{kerr.IllegalSaslState, false},
	}

	for _, test := range tests {
		refreshCalled := false
		refreshFunc := func() {
			refreshCalled = true
		}

		HandleKafkaError(test.err, refreshFunc)
		require.Equal(t, test.expectedRefresh, refreshCalled, "HandleKafkaError(%v) refresh function call mismatch", test.err)
	}
}

func TestIngesterPartitionID(t *testing.T) {
	tests := []struct {
		ingesterID  string
		expected    int32
		expectedErr bool
	}{
		{ingesterID: "ingester-0", expected: 0},
		{ingesterID: "ingester-1", expected: 1},
		{ingesterID: "ingester-zone-a-2", expected: 2},
		{ingesterID: "ingester-2147483647", expected: 2147483647},
		// Past what a partition ID can hold: this used to wrap round to
		// -2147483648 and address a different partition.
		{ingesterID: "ingester-2147483648", expectedErr: true},
		{ingesterID: "ingester-9223372036854775808", expectedErr: true},
		{ingesterID: "ingester", expectedErr: true},
		{ingesterID: "ingester-abc", expectedErr: true},
	}

	for _, test := range tests {
		t.Run(test.ingesterID, func(t *testing.T) {
			actual, err := IngesterPartitionID(test.ingesterID)
			if test.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}
