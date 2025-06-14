package ingest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
)

func TestHandleKafkaError(t *testing.T) {
	tests := []struct {
		err               error
		expectedRefresh   bool
		expectedRetriable bool
	}{
		{nil, false, false},
		{errors.New("Some error"), false, false},
		{errors.New("unknown broker"), false, true},
		{kerr.NotLeaderForPartition, true, true},
		{kerr.ReplicaNotAvailable, true, true},
		{kerr.UnknownLeaderEpoch, true, true},
		{kerr.LeaderNotAvailable, true, true},
		{kerr.BrokerNotAvailable, true, true},
		{kerr.UnknownTopicOrPartition, true, true},
		{kerr.NetworkException, true, true},
		{kerr.NotCoordinator, true, true},
		{kerr.IllegalSaslState, false, false},
	}

	for _, test := range tests {
		refreshCalled := false
		refreshFunc := func() {
			refreshCalled = true
		}

		retriable := HandleKafkaError(test.err, refreshFunc)
		require.Equal(t, test.expectedRefresh, refreshCalled, "HandleKafkaError(%v) refresh function call mismatch", test.err)
		require.Equal(t, test.expectedRetriable, retriable, "HandleKafkaError(%v) retriable mismatch", test.err)
	}
}
