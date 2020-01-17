package ingester

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/joe-elliott/frigg/pkg/friggpb"
)

var ()

func init() {

}

type trace struct {
	batches []*friggpb.PushRequest
	fp      traceFingerprint
	lastAppend time.Time
}

func newTrace(fp model.Fingerprint) *trace {
	return &trace{
		fp:      fp,
		batches: []*friggpb.PushRequest,
		time.Time: time.Now(),
	}
}

func (t *trace) Push(_ context.Context, req *friggpb.PushRequest) error {
	t.batches = append(t.batches, req)
	t.lastAppend = time.Now()

	return nil
}
