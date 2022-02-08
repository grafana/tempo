package remotewrite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gokitlog "github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/stretchr/testify/assert"
)

func Test_remoteWriteAppendable_splitRequests(t *testing.T) {
	nowMs := time.Now().UnixMilli()

	// swap out global maxWriteRequestSize during test
	originalMaxWriteRequestSize := maxWriteRequestSize
	maxWriteRequestSize = 120 // roughly corresponds with 2 timeseries per request
	defer func() {
		maxWriteRequestSize = originalMaxWriteRequestSize
	}()

	mockWriteClient := &mockWriteClient{}

	appender := &remoteWriteAppender{
		logger:       gokitlog.NewLogfmtLogger(os.Stdout),
		ctx:          context.Background(),
		remoteWriter: &remoteWriteClient{WriteClient: mockWriteClient},
		userID:       "",
		metrics:      NewMetrics(prometheus.NewRegistry()),
	}

	// Send samples
	for i := 0; i < 3; i++ {
		_, err := appender.Append(0, labels.Labels{{"label", "value"}}, nowMs, float64(i+1))
		assert.NoError(t, err)
	}
	// Send exemplars
	for i := 0; i < 2; i++ {
		_, err := appender.AppendExemplar(0, labels.Labels{{"label", "value"}}, exemplar.Exemplar{
			Labels: labels.Labels{{"exemplarLabel", "exemplarValue"}},
			Value:  float64(i + 1),
			Ts:     nowMs,
			HasTs:  true,
		})
		assert.NoError(t, err)
	}

	err := appender.Commit()
	assert.NoError(t, err)

	assert.Equal(t, mockWriteClient.storeInvocations, 3)

	// Verify samples
	for i := 0; i < 3; i++ {
		timeseries := mockWriteClient.capturedTimeseries[i]

		assert.Equal(t, `name:"label" value:"value" `, timeseries.Labels[0].String())
		assert.Equal(t, fmt.Sprintf(`value:%d timestamp:%d `, i+1, nowMs), timeseries.Samples[0].String())
		assert.Len(t, timeseries.Exemplars, 0)
	}
	// Verify exemplars
	for i := 0; i < 2; i++ {
		timeseries := mockWriteClient.capturedTimeseries[i+3]

		assert.Equal(t, `name:"label" value:"value" `, timeseries.Labels[0].String())
		assert.Len(t, timeseries.Samples, 0)
		assert.Equal(t, fmt.Sprintf(`labels:<name:"exemplarLabel" value:"exemplarValue" > value:%d timestamp:%d `, i+1, nowMs), timeseries.Exemplars[0].String())
	}
}

type mockWriteClient struct {
	storeInvocations   int
	capturedTimeseries []prompb.TimeSeries
}

var _ remote.WriteClient = (*mockWriteClient)(nil)

func (m *mockWriteClient) Name() string {
	return "mockWriteClient"
}

func (m *mockWriteClient) Endpoint() string {
	return "mockEndpoint"
}

func (m *mockWriteClient) Store(ctx context.Context, b []byte) error {
	m.storeInvocations++

	writeRequest, err := remote.DecodeWriteRequest(bytes.NewReader(b))
	if err != nil {
		return err
	}
	m.capturedTimeseries = append(m.capturedTimeseries, writeRequest.Timeseries...)

	return nil
}
