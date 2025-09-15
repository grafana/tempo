package testkafka

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func NewKafkaClient(t testing.TB, address, topic string) *kgo.Client {
	writeClient, err := kgo.NewClient(
		kgo.SeedBrokers(address),
		kgo.AllowAutoTopicCreation(),
		kgo.DefaultProduceTopic(topic),
		// We will choose the Partition of each record.
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
	)
	require.NoError(t, err)
	t.Cleanup(writeClient.Close)

	return writeClient
}

type ReqOpts struct {
	Partition int32
	Time      time.Time
	TenantID  string
}

func (r *ReqOpts) applyDefaults() {
	if r.TenantID == "" {
		r.TenantID = util.FakeTenantID
	}
	if r.Time.IsZero() {
		r.Time = time.Now()
	}
}

type encodingFn func(partitionID int32, tenantID string, req *tempopb.PushBytesRequest, maxSize int) ([]*kgo.Record, error)

// nolint: revive
func SendReqWithOpts(ctx context.Context, t testing.TB, client *kgo.Client, encode encodingFn, opts ReqOpts) []*kgo.Record {
	traceID := generateTraceID(t)
	opts.applyDefaults()

	startTime := uint64(opts.Time.UnixNano())
	endTime := uint64(opts.Time.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 10, traceID, startTime, endTime)
	records, err := encode(opts.Partition, opts.TenantID, req, 1_000_000)
	require.NoError(t, err)

	res := client.ProduceSync(ctx, records...)
	require.NoError(t, res.FirstErr())

	return records
}

func SendReq(ctx context.Context, t testing.TB, client *kgo.Client, encode encodingFn, tenantID string) []*kgo.Record {
	return SendReqWithOpts(ctx, t, client, encode, ReqOpts{Partition: 0, Time: time.Now(), TenantID: tenantID})
}

// nolint: revive,unparam
func SendTracesFor(t *testing.T, ctx context.Context, client *kgo.Client, dur, interval time.Duration, encode encodingFn) []*kgo.Record {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timer := time.NewTimer(dur)
	defer timer.Stop()

	producedRecords := make([]*kgo.Record, 0)

	for {
		select {
		case <-ctx.Done(): // Exit the function if the context is done
			return producedRecords
		case <-timer.C: // Exit the function when the timer is done
			return producedRecords
		case <-ticker.C:
			records := SendReq(ctx, t, client, encode, util.FakeTenantID)
			producedRecords = append(producedRecords, records...)
		}
	}
}

func generateTraceID(t testing.TB) []byte {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)
	return traceID
}
