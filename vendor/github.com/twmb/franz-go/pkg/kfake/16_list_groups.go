package kfake

import (
	"github.com/twmb/franz-go/pkg/kmsg"
)

// ListGroups: v0-5
//
// Version notes:
// * v1: ThrottleMillis
// * v3: Flexible versions
// * v4: StatesFilter (KIP-518)
// * v5: TypesFilter for new consumer protocol (KIP-848)

func init() { regKey(16, 0, 5) }

func (c *Cluster) handleListGroups(creq *clientReq) (kmsg.Response, error) {
	req := creq.kreq.(*kmsg.ListGroupsRequest)

	if err := c.checkReqVersion(req.Key(), req.Version); err != nil {
		return nil, err
	}

	return c.groups.handleList(creq), nil
}
