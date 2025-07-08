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
