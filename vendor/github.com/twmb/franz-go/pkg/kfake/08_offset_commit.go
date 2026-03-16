package kfake

import (
	"github.com/twmb/franz-go/pkg/kmsg"
)

// OffsetCommit: v0-9
//
// Version notes:
// * v1: Generation, MemberID
// * v2: RetentionTimeMillis (removed in v5)
// * v3: ThrottleMillis
// * v6: LeaderEpoch in request
// * v7: InstanceID - currently returns error if set
// * v8: Flexible versions

func init() { regKey(8, 0, 9) }

func (c *Cluster) handleOffsetCommit(creq *clientReq) (kmsg.Response, error) {
	req := creq.kreq.(*kmsg.OffsetCommitRequest)

	if err := c.checkReqVersion(req.Key(), req.Version); err != nil {
		return nil, err
	}

	c.groups.handleOffsetCommit(creq)
	return nil, nil
}
