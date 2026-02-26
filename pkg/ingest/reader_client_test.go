package ingest_test

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.uber.org/atomic"
)

const (
	leaveTestTopic    = "leave-test-topic"
	leaveTestGroup    = "leave-test-group"
	leaveTestInstance = "leave-test-instance-a"
)

// TestLeaveConsumerGroupByInstanceID_NoOp verifies that an empty instanceID is a
// no-op: no request is sent and nil is returned.
func TestLeaveConsumerGroupByInstanceID_NoOp(t *testing.T) {
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, leaveTestTopic))
	require.NoError(t, err)
	t.Cleanup(fake.Close)
	addr := fake.ListenAddrs()[0]

	var leaveGroupCalled atomic.Int32
	fake.ControlKey(int16(kmsg.LeaveGroup), func(req kmsg.Request) (kmsg.Response, error, bool) {
		leaveGroupCalled.Inc()
		return nil, nil, false
	})

	client, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.DisableClientMetrics())
	require.NoError(t, err)
	defer client.Close()

	err = ingest.LeaveConsumerGroupByInstanceID(t.Context(), client, leaveTestGroup, "", log.NewNopLogger())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, leaveGroupCalled.Load(), "LeaveGroup must not be sent for empty instanceID")
}

// TestLeaveConsumerGroupByInstanceID_SendsRequestWithInstanceID verifies that a
// LeaveGroup request is sent to the broker and contains the correct instance ID.
// The broker intercept returns a synthetic success response because no consumer
// group exists in this minimal test; the purpose is only to verify the wire format.
func TestLeaveConsumerGroupByInstanceID_SendsRequestWithInstanceID(t *testing.T) {
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(2, leaveTestTopic))
	require.NoError(t, err)
	t.Cleanup(fake.Close)
	addr := fake.ListenAddrs()[0]

	receivedInstanceIDs := make(chan string, 4)
	fake.ControlKey(int16(kmsg.LeaveGroup), func(req kmsg.Request) (kmsg.Response, error, bool) {
		lr := req.(*kmsg.LeaveGroupRequest)
		for _, m := range lr.Members {
			if m.InstanceID != nil {
				select {
				case receivedInstanceIDs <- *m.InstanceID:
				default:
				}
			}
		}
		// Use ResponseKind() to get a correctly versioned response (version mirrors
		// the request version), then return success so the caller does not error.
		resp := lr.ResponseKind().(*kmsg.LeaveGroupResponse)
		resp.Default()
		return resp, nil, true
	})

	client, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.DisableClientMetrics())
	require.NoError(t, err)
	defer client.Close()

	err = ingest.LeaveConsumerGroupByInstanceID(t.Context(), client, leaveTestGroup, leaveTestInstance, log.NewNopLogger())
	require.NoError(t, err)

	select {
	case id := <-receivedInstanceIDs:
		assert.Equal(t, leaveTestInstance, id)
	case <-time.After(3 * time.Second):
		t.Fatal("broker did not receive a LeaveGroup request")
	}
}
