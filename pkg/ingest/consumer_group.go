package ingest

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// LeaveConsumerGroupByInstanceID sends a LeaveGroup request for the given
// instance ID so the coordinator can rebalance without waiting for session
// timeout. Use this on shutdown when using static membership (InstanceID):
// franz-go does not send LeaveGroup on Close() when InstanceID is set.
//
// Requires Kafka 2.4+ (KIP-345), which introduced per-member InstanceID in
// the LeaveGroup request (protocol version 3). On older brokers, franz-go
// negotiates a lower request version that does not carry InstanceID; the
// resulting request will fail with UNKNOWN_MEMBER_ID and the error is
// returned to the caller. Partitions then fall back to the session timeout.
// Since static membership itself requires Kafka 2.4+, callers using
// instance_id are already on a compatible broker.
//
// No-op if instanceID is empty.
func LeaveConsumerGroupByInstanceID(ctx context.Context, client *kgo.Client, group, instanceID string, logger log.Logger) error {
	if instanceID == "" {
		return nil
	}
	req := kmsg.NewPtrLeaveGroupRequest()
	req.Group = group
	member := kmsg.NewLeaveGroupRequestMember()
	member.InstanceID = &instanceID
	req.Members = append(req.Members, member)
	resp, err := req.RequestWith(ctx, client)
	if err != nil {
		return err
	}
	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		return err
	}
	// v3+ responses carry per-member error codes; check those too.
	var memberErr error
	for _, m := range resp.Members {
		if m.ErrorCode == 0 {
			continue
		}
		if m.InstanceID != nil && *m.InstanceID == instanceID {
			return kerr.ErrorForCode(m.ErrorCode)
		}
		if memberErr == nil {
			memberErr = kerr.ErrorForCode(m.ErrorCode)
		}
	}
	if memberErr != nil {
		return memberErr
	}
	level.Info(logger).Log("msg", "left Kafka consumer group by instance ID", "group", group, "instance_id", instanceID)
	return nil
}
