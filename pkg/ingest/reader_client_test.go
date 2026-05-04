package ingest_test

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
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
	fake.ControlKey(int16(kmsg.LeaveGroup), func(_ kmsg.Request) (kmsg.Response, error, bool) {
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

// TestLeaveConsumerGroupByInstanceID_OurMemberErrorPrioritized verifies that when
// the response contains errors for multiple members, our instance's error is
// returned even when an unrelated member's error appears first in the list. The loop
// must scan all members before deciding — it cannot short-circuit on the first
// foreign error.
func TestLeaveConsumerGroupByInstanceID_OurMemberErrorPrioritized(t *testing.T) {
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, leaveTestTopic))
	require.NoError(t, err)
	t.Cleanup(fake.Close)
	addr := fake.ListenAddrs()[0]

	otherInstance := "other-instance"
	fake.ControlKey(int16(kmsg.LeaveGroup), func(req kmsg.Request) (kmsg.Response, error, bool) {
		lr := req.(*kmsg.LeaveGroupRequest)
		resp := lr.ResponseKind().(*kmsg.LeaveGroupResponse)
		resp.Default()

		// Unrelated member first, our member second.
		other := kmsg.NewLeaveGroupResponseMember()
		other.InstanceID = &otherInstance
		other.ErrorCode = kerr.UnknownMemberID.Code

		ours := kmsg.NewLeaveGroupResponseMember()
		ours.InstanceID = &[]string{leaveTestInstance}[0]
		ours.ErrorCode = kerr.GroupAuthorizationFailed.Code

		resp.Members = append(resp.Members, other, ours)
		return resp, nil, true
	})

	client, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.DisableClientMetrics())
	require.NoError(t, err)
	defer client.Close()

	err = ingest.LeaveConsumerGroupByInstanceID(t.Context(), client, leaveTestGroup, leaveTestInstance, log.NewNopLogger())
	require.Error(t, err)
	assert.ErrorIs(t, err, kerr.GroupAuthorizationFailed, "our member's error must take priority over an unrelated member's error")
}

// TestLeaveConsumerGroupByInstanceID_MemberErrorCode verifies that a zero
// top-level ErrorCode does not mask a per-member error for our instance.
// LeaveGroup v3+ carries per-member error codes; a broker can return
// ErrorCode=0 at the top level while reporting UNKNOWN_MEMBER_ID for the
// specific instance that tried to leave.
func TestLeaveConsumerGroupByInstanceID_MemberErrorCode(t *testing.T) {
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, leaveTestTopic))
	require.NoError(t, err)
	t.Cleanup(fake.Close)
	addr := fake.ListenAddrs()[0]

	fake.ControlKey(int16(kmsg.LeaveGroup), func(req kmsg.Request) (kmsg.Response, error, bool) {
		lr := req.(*kmsg.LeaveGroupRequest)
		resp := lr.ResponseKind().(*kmsg.LeaveGroupResponse)
		resp.Default()
		resp.ErrorCode = 0 // top-level success
		for _, m := range lr.Members {
			rm := kmsg.NewLeaveGroupResponseMember()
			rm.MemberID = m.MemberID
			rm.InstanceID = m.InstanceID
			rm.ErrorCode = kerr.UnknownMemberID.Code
			resp.Members = append(resp.Members, rm)
		}
		return resp, nil, true
	})

	client, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.DisableClientMetrics())
	require.NoError(t, err)
	defer client.Close()

	err = ingest.LeaveConsumerGroupByInstanceID(t.Context(), client, leaveTestGroup, leaveTestInstance, log.NewNopLogger())
	require.Error(t, err, "expected error from per-member error code")
	assert.ErrorIs(t, err, kerr.UnknownMemberID)
}
